#pragma once

#include <stdint.h>
#include <stdlib.h>
#include <stdbool.h>

// These guards are needed for CGO
#ifdef __cplusplus
extern "C" {
#endif
    /*
     * ======== Structs ========
     */
    
    // Lazily defined so that we don't need to link to SEAL at runtime
    typedef struct context_s context_t;
    typedef struct skey_s skey_t;
    typedef struct ciphertext_s ciphertext_t;
    typedef struct a_s a_t;
    typedef struct plaintext_s plaintext_t;

    /*
     * ======== Context ========
     */
    
    // Initialization
    context_t *ctx_new(uint64_t p_mod, uint64_t n, uint64_t logq, bool mod_switch);
    void ctx_free(context_t *ctx);

    // Getters
    size_t ctx_n(context_t *ctx);
    size_t ctx_p(context_t *ctx);

    /*
     * ======== Secret Key ========
     */

    // Initialization
    skey_t *key_new(context_t *ctx);
    void key_free(skey_t *key);

    // Getters
    size_t key_size(skey_t *key);

    /*
     * ======== Ciphertext ========
     */
    
    // Initialization
    ciphertext_t *ct_new(void);
    void ct_free(ciphertext_t *ct);

    // Getters
    size_t ct_size(ciphertext_t *ct);

    // Sample dummy ciphertexts
    size_t dummy_ct_size(context_t *ctx);
    void store_dummy_cts(context_t *ctx, uint64_t *c_seed, size_t samples, uint8_t *dst, size_t *sizes);

    // Truncate a ciphertext to a particular size
    void truncate_ct(context_t *ctx, ciphertext_t *ct, size_t size);

    /*
     * ======== A Poly ========
     */

    // Initialization
    a_t *a_new(skey_t *sk, uint64_t *c_seed);
    void a_free(a_t *a);

    /*
     * ======== Plaintext ========
     */
   
    // Initialization
    plaintext_t *pt_new(void);
    void pt_set_32(plaintext_t *pt, const uint32_t *vals, size_t slots);
    void pt_set_64(plaintext_t *pt, const uint64_t *vals, size_t slots);
    void pt_free(plaintext_t *pt);

    /*
     * ======== Hybrid Mode ========
     */
    
    // This function computes a product of the form `H_i = D * a_i`, where the
    // various `i`s are stacked vertically. This is slightly tricky since
    // we know how to _right-multiplication_ of a polynomial by a matrix over a
    // negacyclic ring, but not left. To get around this we instead compute
    // `(a_i)^T * D^T` and then transpose the final product: observing that the
    // "transpose" operator on a polynomial here corresponds to a substitution
    // operation. 
    void mul_matrix_As_32(
        context_t *ctx,
        uint32_t *matrix,
        uint64_t *seeds,
        uint32_t *dst,
        uint64_t rows,
        uint64_t cols,
        uint64_t num_seeds
    );
    void mul_matrix_As_64(
        context_t *ctx,
        uint32_t *matrix,
        uint64_t *seeds,
        uint64_t *dst,
        uint64_t rows,
        uint64_t cols,
        uint64_t num_seeds
    );

    // Convert from RLWE -> LWE via modulus switching using  a new modulus of
    // 2^32 or 2^64
    //
    // NOTE: Assumes that all destination pointers have already been allocated to
    // have the correct size (we do this so that Go can manage the memory vs.
    // C++)
    void key_extract_lwe_32(context_t *ctx, skey_t *key, uint32_t *lwe_s);
    void key_extract_lwe_64(context_t *ctx, skey_t *key, uint64_t *lwe_s);
 
    void ct_extract_lwe_32(
        context_t *ctx,
        uint8_t *src,
        size_t size,
        uint64_t samples,
        uint32_t *lwe_ct
    );
    void ct_extract_lwe_64(
        context_t *ctx,
        uint8_t *src,
        size_t size,
        uint64_t samples,
        uint64_t *lwe_ct
    );

    // Preprocess `A * s` of the ciphertext computation
    void key_preprocess_enc(skey_t *key, a_t *a, ciphertext_t *ct);

    // Generate an RLWE encryption given a preprocessed ciphertext
    void key_enc_preprocessed(skey_t *key, plaintext_t *pt, ciphertext_t *ct);

    // Store a ciphertext as a raw byte array 
    size_t ct_data_size(ciphertext_t *ct);
    void ct_store_data(ciphertext_t *ct, uint8_t *dst, size_t sz, size_t *written);

    // Helper functions for rounding / lifting (in place) on modulus-switched
    // LWE objects
    //
    // TODO: Figure out how to remove these
    void round_lwe_32(context_t *ctx, uint32_t *lwe, size_t length);
    void round_lwe_64(context_t *ctx, uint64_t *lwe, size_t length);
    void lift_lwe_32(context_t *ctx, uint32_t *lwe, size_t length);
    void lift_lwe_64(context_t *ctx, uint64_t *lwe, size_t length);

#ifdef __cplusplus
}
#endif
