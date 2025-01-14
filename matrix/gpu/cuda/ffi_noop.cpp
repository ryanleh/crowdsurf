#include <iostream>
#include "ffi.h"

/*
 * This is a NOOP implementation of `ffi.h` that simply fails on being called.
 *
 * We need to generate a libgpu.so library for the Go code to link to (since Go
 * doesn't have conditional compilation), however properly building the library
 * requires a CUDA compiler and we don't want this to be a hard requirement.
 * Thus, this file allows us to generate a dummy library to link to.
 */

void fail() {
    std::cerr << "ERROR: GPU not supported" << std::endl;
    exit(1);
} 

#define BuildGPUMul(bits)        \
struct GPUMul##bits {};          \
\
GPUMul##bits##_t *mul_new_##bits(uint64_t m, uint64_t k, uint64_t n) {  \
    fail();                                                             \
    return nullptr;                                                     \
}                                                                       \
\
void mul_free_##bits(GPUMul##bits##_t *ctx) { \
    fail();                                   \
}                                             \
\
void allocate_##bits(GPUMul##bits##_t *ctx, uint64_t m, uint64_t k, uint64_t n) { \
    fail();                                                                       \
}                                                                                 \
\
void set_batch_##bits(GPUMul##bits##_t *ctx, uint64_t n) { \
    fail();                                                \
}                                                          \
\
uint32_t* get_host_a_##bits(GPUMul##bits##_t *ctx) { \
    fail();                                                              \
    return 0;                                                            \
}                                                                        \
\
uint##bits##_t* get_host_data_##bits(GPUMul##bits##_t *ctx, int index) { \
    fail();                                                              \
    return 0;                                                            \
}                                                                        \
\
void sync_device_##bits(GPUMul##bits##_t *ctx, int index) { \
    fail();                                                 \
}                                                           \
\
void gemm_##bits(GPUMul##bits##_t *ctx) {                             \
    fail();                                                           \
}                                                                     \

bool use_gpu() {
    return false;
}

BuildGPUMul(32);
BuildGPUMul(64);
