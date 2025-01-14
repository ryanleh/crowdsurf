#include <cuda.h>
#include "ffi.h"
#include "gemm.cu"

#define BuildGPUMul(bits)        \
struct GPUMul##bits {            \
    GPUMul<uint##bits##_t> *ctx; \
};                               \
\
GPUMul##bits##_t *mul_new_##bits(uint64_t m, uint64_t k, uint64_t n) {  \
    auto *ctx = (GPUMul##bits##_t*) malloc(sizeof(GPUMul##bits##_t));   \
    ctx->ctx = new GPUMul<uint##bits##_t>(m, k, n);                     \
    return ctx;                                                         \
}                                                                       \
\
void mul_free_##bits(GPUMul##bits##_t *ctx) { \
    delete ctx->ctx;                          \
    free(ctx);                                \
}                                             \
\
void allocate_##bits(GPUMul##bits##_t *ctx, uint64_t m, uint64_t k, uint64_t n) { \
    ctx->ctx->allocate(m, k, n);                                                  \
}                                                                                 \
\
void set_batch_##bits(GPUMul##bits##_t *ctx, uint64_t n) { \
    ctx->ctx->set_batch(n);                                \
}                                                          \
\
uint32_t* get_host_a_##bits(GPUMul##bits##_t *ctx) { \
    return ctx->ctx->raw_host_a();                   \
}                                                    \
\
uint##bits##_t* get_host_data_##bits(GPUMul##bits##_t *ctx, int index) { \
    return ctx->ctx->raw_host_data(index);                               \
}                                                                        \
\
void sync_device_##bits(GPUMul##bits##_t *ctx, int index) { \
    ctx->ctx->sync_device(index);                           \
}                                                           \
\
void gemm_##bits(GPUMul##bits##_t *ctx) {                             \
    ctx->ctx->gemm();                                                 \
}                                                                     \

bool use_gpu() {
    int deviceCount;
    cudaError_t e = cudaGetDeviceCount(&deviceCount);
    return e == cudaSuccess ? deviceCount > 0 : -1;
}

BuildGPUMul(32);
BuildGPUMul(64);
