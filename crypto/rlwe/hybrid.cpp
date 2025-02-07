#include <gmp.h>
#include <seal/seal.h>
#include <seal/util/polyarithsmallmod.h>
#include "seal/util/rlwe.h"

#include "ffi.h"
#include "common.h"

using namespace seal;


/*
 * Templated helper functions
 */

template<typename T>
void mod_switch(context_t *ctx, util::ConstRNSIter inp, T* out, uint64_t samples) {
    auto ctx_data = ctx->ctx->context->first_context_data();
    auto num_moduli = ctx_data->parms().coeff_modulus().size();
    auto q_rns_base = ctx_data->rns_tool()->base_q();
    auto inv_punc_prods = q_rns_base->inv_punctured_prod_mod_base_array();
    auto moduli = q_rns_base->base();

    // Allocate a temporary MPF for intermediate results
    SEAL_ITERATE(util::SeqIter(0), samples, [&](auto I) {
        // We modulus switch one CRT component at a time and sum all the
        // results
        T result = 0;
        SEAL_ITERATE(util::SeqIter(0), num_moduli, [&](auto J) {
            // t1 = x * q^i_inv mod q_i
            uint64_t t1 = util::multiply_uint_mod(
                inp[J][I],
                inv_punc_prods[J],
                moduli[J]
            );

            // t2 = t1 * q_t / q_i
            mpf_mul_ui(ctx->ctx->tmp_mpf, ctx->ctx->moduli_inv[J], t1);
            result += mpf_get_ui(ctx->ctx->tmp_mpf);
        });
        out[I] = result;
    });
}

// Forward declare the helper function below
void a_transpose(std::shared_ptr<const SEALContext::ContextData> ctx_data, DynArray<uint64_t> &a);

// TODO: Make this more SEAL-y
template<typename T>
void mul_matrix_As(
    context_t *ctx,
    uint32_t *matrix,
    uint64_t *seeds,
    T *dst,
    uint64_t rows,
    uint64_t cols,
    uint64_t num_seeds
) {
    auto ctx_data = ctx->ctx->context->first_context_data();
    auto poly_degree = ctx_data->parms().poly_modulus_degree();
    auto moduli = ctx_data->parms().coeff_modulus();

    // Compute the `a` polynomials
    CryptoKey key(*ctx->ctx->context);
    std::vector<DynArray<uint64_t>> As(num_seeds);
    
    for (size_t i = 0; i < As.size(); i++) {
        // Expand the seed to a poly (each seed is 512 bits)
        prng_seed_type seed;
        std::copy(seeds + i * 8, seeds + (i + 1) * 8, seed.begin());
        key.encryptor.get_a(As[i], seed);
        
        // Transpose the poly
        a_transpose(ctx_data, As[i]);
    }
    
    // Temporary variables used during iteration
    auto size = util::mul_safe(poly_degree, moduli.size());
    DynArray<uint64_t> accum(size);
    DynArray<uint64_t> tmp(size);
    DynArray<uint64_t> row_pt(size);

    // For each row of the matrix, compute the product with all `a` polys
    for (size_t row = 0; row < rows; row++) {
        // Clear the accumulator 
        std::fill(accum.begin(), accum.end(), 0);

        // Compute a row of the hint
        for (size_t i = 0; i < As.size(); i++) {
            std::fill(row_pt.begin(), row_pt.end(), 0);
            
            // Extract the current row and encode in RNS form
            auto stop = std::min(static_cast<uint64_t>(poly_degree), cols - i * poly_degree);
            for (size_t j = 0; j < moduli.size(); j++) {
                for (size_t z = 0; z < stop; z++) {
                    auto idx = row * cols + i * poly_degree + z;
                    uint64_t val = static_cast<uint64_t>(matrix[idx]);
                    row_pt[j * poly_degree + z] = val % moduli[j].value();
                }
            }

            // Apply NTT
            util::ntt_negacyclic_harvey_lazy(
                util::RNSIter(row_pt.begin(), poly_degree),
                moduli.size(),
                iter(ctx_data->small_ntt_tables())
            );

            // Compute the product with the relevant `a` polynomial and store
            // in `tmp`
            util::dyadic_product_coeffmod(
                util::RNSIter(As[i].begin(), poly_degree),
                util::ConstRNSIter(row_pt.begin(), poly_degree),
                moduli.size(),
                moduli,
                util::RNSIter(tmp.begin(), poly_degree)
            );
           
            // Add the contents of `tmp` into `accum`
            for (size_t j = 0; j < size; j++) {
                accum[j] += tmp[j];
            }
        }

        // Reduce accum before applying inverse NTT
        for (size_t j = 0; j < moduli.size(); j++) {
            for (size_t z = 0; z < poly_degree; z++) {
                accum[j * poly_degree + z] %= moduli[j].value();
            }
        }
        util::inverse_ntt_negacyclic_harvey_lazy(
            util::RNSIter(accum.begin(), poly_degree),
            moduli.size(),
            iter(ctx_data->small_ntt_tables())
        );

        // Modulus switch `accum` and store in `dst`
        mod_switch<T>(
            ctx,
            util::ConstRNSIter(accum.begin(), poly_degree),
            dst + row * poly_degree,
            poly_degree
        );
    }
}

template<typename T>
void key_extract_lwe(context_t *ctx, skey_t *key, T *lwe_s) {
    auto ctx_data = ctx->ctx->context->first_context_data();
    auto poly_degree = ctx_data->parms().poly_modulus_degree();

    // Extract LWE secret key (i.e. modulus switch secret key)
    //
    // Note that the secret key is decomposed with CRT. When the absolute value
    // of the value being decomposed is less than any of the limbs, then they
    // all have an equivalent value. Thus, since we're working with a
    // small-norm secret key we can just extract from one of the limbs. This
    // would _not_ hold for a uniform secret key.
    auto q0 = ctx_data->parms().coeff_modulus()[0].value();
    auto boundary = (q0 - 1) / 2;
    auto raw_sk = key->key.keygen.secret_key_coeff().data().data();
    for (uint64_t i = 0; i < poly_degree; i++) {
        // Handle negative entries correctly since key is gaussian
        if (raw_sk[i] > boundary) {
            lwe_s[i] = T(0) - (q0 - raw_sk[i]);
        } else {
            lwe_s[i] = static_cast<T>(raw_sk[i]);
        }
    }
}

template <typename T>
void ct_extract_lwe(
    context_t *ctx,
    uint8_t *src,
    size_t size,
    uint64_t samples,
    T *lwe_ct
) {
    auto ctx_data = ctx->ctx->context->first_context_data();
    size_t coeff_modulus_size = ctx_data->parms().coeff_modulus().size();

    // Deserialize ciphertext data
    DynArray<uint64_t> ct;
    ct.load((const seal_byte*) src, size);

    // Modulus switch
    // 
    // TODO: Hacky way we initialize the RNSIter to account for ciphertexts < n
    mod_switch<T>(
        ctx,
        util::ConstRNSIter(ct.begin(), ct.size() / coeff_modulus_size), // TODO: Hacky
        lwe_ct,
        samples
    );
}

// TODO: This could be sped up if it's a bottleneck
template<typename T>
void round_lwe(context_t *ctx, T *lwe, size_t length) {
    auto ctx_data = ctx->ctx->context->first_context_data();
    auto poly_modulus = ctx_data->parms().plain_modulus();

    for (size_t i = 0; i < length; i++) {
        // result = b * p / q2
        mpf_set_d(ctx->ctx->tmp_mpf, poly_modulus.value());
        mpf_mul_ui(ctx->ctx->tmp_mpf, ctx->ctx->tmp_mpf, lwe[i]);
        mpf_add_ui(ctx->ctx->tmp_mpf, ctx->ctx->tmp_mpf, ctx->ctx->q_div_2);
        mpf_div(ctx->ctx->tmp_mpf, ctx->ctx->tmp_mpf, ctx->ctx->q);
        lwe[i] = static_cast<T>(mpf_get_ui(ctx->ctx->tmp_mpf) % poly_modulus.value());
    }
}

// TODO: This could be sped up if it's a bottleneck
template<typename T>
void lift_lwe(context_t *ctx, T *lwe, size_t length) {
    auto ctx_data = ctx->ctx->context->first_context_data();
    auto poly_modulus = ctx_data->parms().plain_modulus();

    for (size_t i = 0; i < length; i++) {
        // result = b * q2 / p
        mpf_mul_ui(ctx->ctx->tmp_mpf, ctx->ctx->q, lwe[i]);
        mpf_add_ui(ctx->ctx->tmp_mpf, ctx->ctx->tmp_mpf, poly_modulus.value() / 2);
        mpf_div_ui(ctx->ctx->tmp_mpf, ctx->ctx->tmp_mpf, poly_modulus.value());
        lwe[i] = static_cast<T>(mpf_get_ui(ctx->ctx->tmp_mpf));
    }
}


/*
 * Instantiate non-templated parts of the FFI
 */

void a_transpose(std::shared_ptr<const SEALContext::ContextData> ctx_data, DynArray<uint64_t> &a) {
    auto poly_degree = ctx_data->parms().poly_modulus_degree();
    auto moduli = ctx_data->parms().coeff_modulus();

    // Invert NTT representation of `a`
    util::inverse_ntt_negacyclic_harvey_lazy(
        util::RNSIter(a.begin(), poly_degree),
        moduli.size(),
        iter(ctx_data->small_ntt_tables())
    );

    // Re-arrange the coefficients.
    // We can swap the `i`-th and `n - 1`th coefficients, negating both
    // 
    // TODO: Benchmark whether to use SEAL_iterate or something different here
    util::RNSIter a_iter(a.begin(), poly_degree);
    util::ConstModulusIter coeff_iter(moduli);
    SEAL_ITERATE(iter(a_iter, coeff_iter), moduli.size(), [&](auto I) {
        SEAL_ITERATE(util::SeqIter(1), poly_degree / 2, [&](auto J) {
            uint64_t neg_t1 = get<1>(I).value() - get<0>(I)[J];
            get<0>(I)[J] = get<1>(I).value() - get<0>(I)[poly_degree - J];
            get<0>(I)[poly_degree - J] = neg_t1;
        });
    });

    // NTT transform the result
    util::ntt_negacyclic_harvey_lazy(
        util::RNSIter(a.begin(), poly_degree),
        moduli.size(),
        iter(ctx_data->small_ntt_tables())
    );
}

void key_preprocess_enc(skey_t *key, a_t *a, ciphertext_t *ct) {
    key->key.encryptor.preprocess_encrypt_symmetric(ct->ct, a->a);
}

void key_enc_preprocessed(skey_t *key, plaintext_t *pt, ciphertext_t *ct) {
    key->key.encryptor.encrypt_symmetric_preprocessed(pt->pt, ct->ct);
}

size_t ct_data_size(ciphertext_t *ct) {
    return static_cast<size_t>(ct->ct.dyn_array().save_size(seal::compr_mode_type::none));
}

void ct_store_data(ciphertext_t *ct, uint8_t *dst, size_t sz, size_t *written) {
    *written = ct->ct.dyn_array().save((seal_byte*) dst, sz, seal::compr_mode_type::none);
}

size_t dummy_ct_size(context_t *ctx) {
    // Get context data
    auto ctx_data = ctx->ctx->context->first_context_data();
    auto &parms = ctx_data->parms();
    size_t coeff_modulus_size = parms.coeff_modulus().size();
    size_t coeff_count = parms.poly_modulus_degree();
    size_t ct_size = util::mul_safe(coeff_count, coeff_modulus_size);

    // Sample a new dummy array
    DynArray<uint64_t> buf(ct_size);
    return buf.save_size(seal::compr_mode_type::none);
}

void store_dummy_cts(context_t *ctx, uint64_t *c_seed, size_t samples, uint8_t *dst, size_t *sizes) {
    // Get context data
    auto ctx_data = ctx->ctx->context->first_context_data();
    auto &parms = ctx_data->parms();
    size_t coeff_modulus_size = parms.coeff_modulus().size();
    size_t coeff_count = parms.poly_modulus_degree();
    size_t ser_size = dummy_ct_size(ctx);

    // Initialize PRG
    prng_seed_type seed;
    std::copy(c_seed, c_seed + 8, seed.begin());
    auto prg = UniformRandomGeneratorFactory::DefaultFactory()->create(seed);

    for (int i = 0; samples > 0; i++) {
        auto to_encrypt = std::min(samples, coeff_count);
        size_t ct_size = util::mul_safe(to_encrypt, coeff_modulus_size);

        // Sample a new dummy array
        DynArray<uint64_t> buf(ct_size);
        util::sample_poly_uniform(prg, parms, (std::size_t)samples, buf.begin());
        sizes[i] = buf.save((seal_byte*) (dst + i * ser_size), ser_size, seal::compr_mode_type::none);
        samples -= to_encrypt;
    }
}

void truncate_ct(context_t *ctx, ciphertext_t *ct, size_t size) {
    auto ctx_data = ctx->ctx->context->first_context_data();
    auto &parms = ctx_data->parms();
    size_t coeff_modulus_size = parms.coeff_modulus().size();
    size_t coeff_count = parms.poly_modulus_degree();

    // Create a new backing array of the right size
    DynArray<uint64_t> tmp(size * coeff_modulus_size);
    const auto orig = ct->ct.dyn_array();

    // Copy the relevant elements
    for (size_t i = 0; i < coeff_modulus_size; i++) {
        for (size_t j = 0; j < size; j++) {
            tmp[i * size + j] = orig[i * coeff_count + j];
        }
    }
   
    ct->ct.set_array(tmp);
}

/*
 * Macro to instantiate the rest of the FFI
 */

#define BuildFFI(bits) \
\
void mul_matrix_As_##bits(                                                          \
    context_t *ctx,                                                                 \
    uint32_t *matrix,                                                         \
    uint64_t *seeds,                                                                \
    uint##bits##_t *dst,                                                            \
    uint64_t rows,                                                                  \
    uint64_t cols,                                                                  \
    uint64_t num_seeds                                                              \
) {                                                                                 \
    mul_matrix_As<uint##bits##_t>(ctx, matrix, seeds, dst, rows, cols, num_seeds);  \
}                                                                                   \
\
void key_extract_lwe_##bits(                            \
    context_t *ctx,                                     \
    skey_t *key,                                        \
    uint##bits##_t *lwe_s                               \
) {                                                     \
    key_extract_lwe<uint##bits##_t>(ctx, key, lwe_s);   \
}                                                       \
\
void ct_extract_lwe_##bits(                                          \
    context_t *ctx,                                                  \
    uint8_t *src,                                                    \
    size_t size,                                                     \
    uint64_t samples,                                                \
    uint##bits##_t *lwe_ct                                           \
) {                                                                  \
    ct_extract_lwe<uint##bits##_t>(ctx, src, size, samples, lwe_ct); \
}                                                                    \
\
void round_lwe_##bits(context_t *ctx, uint##bits##_t *lwe, size_t length) { \
    round_lwe<uint##bits##_t>(ctx, lwe, length);                            \
}                                                                           \
\
void lift_lwe_##bits(context_t *ctx, uint##bits##_t *lwe, size_t length) { \
    lift_lwe<uint##bits##_t>(ctx, lwe, length);                  \
}                                                                \

BuildFFI(32);
BuildFFI(64);
