#include <gmp.h>
#include <seal/seal.h>
#include <assert.h>

#include "ffi.h"

using namespace seal;

/*
 * Main class for managing SEAL context
 */
class CryptoContext {
public: 
    CryptoContext(uint64_t p_mod, uint64_t n, uint64_t logq, bool mod_switch)
        : n(n), p(p_mod), mod_switch(mod_switch), q_div_2(pow(2, logq-1))
    {
        // Generate encryption params
        EncryptionParameters parms(scheme_type::bfv);
        parms.set_poly_modulus_degree(n);
        parms.set_plain_modulus(p_mod);

        // Set the ciphertext modulus. The MPF precision needs to be at least the
        // number of bits of the initial modulus.
        switch(logq) {
            case 32:
                // 33-bit coeff modulus
                parms.set_coeff_modulus(CoeffModulus::Create(n, {33}));
                mpf_set_default_prec(33);
                break;
            case 64:
                // 65-bit coeff modulus.
                //
                // If there are > 2 moduli, SEAL designates the last prime as
                // 'special' and uses it to speed up various operations we
                // don't need.
                parms.set_coeff_modulus(CoeffModulus::Create(n, {32, 33, 33}));
                mpf_set_default_prec(65);
                break;
            default:
                std::cerr << "Unsupported RLWE modulus: " << logq << std::endl;
                exit(1);
        }

        // Build the SEAL context
        context = std::unique_ptr<SEALContext>(new SEALContext(parms));
        if (!context->parameters_set()) {
            std::cerr << "Invalid SEAL params ("
                << context->parameter_error_name() << "): "
                << context->parameter_error_message() << std::endl;
            exit(1);
        }

        // Initialize other fields
        evaluator = std::unique_ptr<Evaluator>(new Evaluator(*context));
        parms_id = parms.parms_id();

        // If modulus switching, preprocess MPFs
        if (mod_switch) {
            auto ctx_data = context->first_context_data();
            auto coeff_modulus = ctx_data->parms().coeff_modulus();

            // Initialize MPFs
            mpf_inits(tmp_mpf, q, moduli_inv[0], moduli_inv[1], NULL);

            // Precompute q / q_i for each moduli
            switch(logq) {
                case 32:
                    mpf_set_d(q, pow(2, 32));
                    break;
                case 64:
                    mpf_set_d(q, pow(2, 64));
                    break;
                default:
                    std::cerr << "Unsupported RLWE modulus: " << logq << std::endl;
                    exit(1);
            }
            for (size_t i = 0; i < coeff_modulus.size(); i++) {
                mpf_div_ui(moduli_inv[i], q, coeff_modulus[i].value());
            }
        }
    };

    virtual ~CryptoContext() {
        // If modulus switching, free allocating MPFs
        if (mod_switch) {
            mpf_clears(tmp_mpf, q, moduli_inv[0], moduli_inv[1], NULL);
        }
    }

    std::unique_ptr<SEALContext> context;
    std::unique_ptr<Evaluator> evaluator;
    parms_id_type parms_id;

    size_t n;
    size_t p;

    // Stuff for modulus switching
    bool mod_switch;
    uint64_t q_div_2;

    mpf_t tmp_mpf;
    mpf_t q;
    std::array<mpf_t, 2> moduli_inv;
};

/*
 * Manages a SEAL secret key
 */ 
class CryptoKey {
public: 
    CryptoKey(SEALContext &context) :
        keygen(KeyGenerator(context, true, true)), // NOTE: Always generating gaussian key
        sk(keygen.secret_key()),
        encryptor(context, sk),
        decryptor(context, sk) {};

    inline void set_key(SEALContext &context, uint8_t *src, size_t sz) {
        this->sk.load(context, (const seal_byte*) src, sz);
        new (&encryptor) Encryptor(context, this->sk);
        new (&decryptor) Decryptor(context, this->sk);
    };

    //virtual ~CryptoKey() {};

    KeyGenerator keygen;
    SecretKey sk;
    Encryptor encryptor;
    Decryptor decryptor;
};

/*
 * C-wrapper types
 */
struct context_s {
    CryptoContext *ctx; 
}; 

struct skey_s {
    CryptoKey key;
    skey_s(SEALContext &ctx) : key(ctx) {};
};

struct ciphertext_s {
    Ciphertext ct;
    ciphertext_s() : ct(Ciphertext(MemoryPoolHandle::Global())) {};
}; 

struct a_s {
    DynArray<uint64_t> a;
    prng_seed_type seed;
    a_s(CryptoKey &key, prng_seed_type &seed) : a(MemoryPoolHandle::Global()) {
        key.encryptor.get_a(a, seed);
    };
};

struct plaintext_s {
    Plaintext pt;  
    plaintext_s() : pt(Plaintext(MemoryPoolHandle::Global())) {};
};
