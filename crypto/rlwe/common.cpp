#include "ffi.h"
#include "common.h"

/*
 * ======== Context ========
 */

context_t *ctx_new(uint64_t p_mod, uint64_t n, uint64_t logq, bool mod_switch) {
    context_t *ctx = (context_t*)malloc(sizeof(context_t));
    ctx->ctx = new CryptoContext(p_mod, n, logq, mod_switch);
    return ctx;
}

void ctx_free(context_t *ctx) {
    delete ctx->ctx;
    free(ctx);
}

size_t ctx_n(context_t *ctx) {
    return ctx->ctx->n;
}

size_t ctx_p(context_t *ctx) {
    return ctx->ctx->p;
}

/*
 * ======== Secret Key ========
 */

skey_t *key_new(context_t *ctx) {
    return new skey_s(*(ctx->ctx->context));
}

void key_free(skey_t *key) {
    delete key;
}

size_t key_size(skey_t *key) {
    return static_cast<size_t>(key->key.sk.save_size(compr_mode_type::none));
}

/*
 * ======== Ciphertext ========
 */

ciphertext_t *ct_new(void) {
    return new ciphertext_s();
}

void ct_free(ciphertext_t *ct) {
    delete ct;
}

size_t ct_size(ciphertext_t *ct) {
    return static_cast<size_t>(ct->ct.save_size(compr_mode_type::none));
}
    
/*
 * ======== A Poly ========
 */

a_t *a_new(skey_t *sk, uint64_t *c_seed) {
    prng_seed_type seed;
    std::copy(c_seed, c_seed + 8, seed.begin());
    return new a_s(sk->key, seed);
}
    
void a_free(a_t *a) {
    delete a;
}

/*
 * ======== Plaintext ========
 */

plaintext_t *pt_new(void) {
    return new plaintext_s();
}
   
#define BuildPTSet(bits) \
\
void pt_set_##bits(plaintext_t *pt, const uint##bits##_t *vals, size_t slots) { \
    pt->pt.resize(slots);                                                       \
    for (size_t i=0; i < slots; i++) {                                          \
        pt->pt[i] = static_cast<uint64_t>(vals[i]);                             \
    }                                                                           \
}
BuildPTSet(32);
BuildPTSet(64);

void pt_free(plaintext_t *pt) {
    delete pt;
}
