#pragma once

#include <stdint.h>
#include <stdbool.h>

// These guards are needed for CGO
#ifdef __cplusplus
extern "C" {
#endif
    // CGO doesn't work properly with NVCC (the CUDA compiler) so our header
    // file here needs to be completely oblivious to anything CUDA-specific.
    // This typedef allows us to define the struct for our Go code but defers
    // its actual definition to the separately-compiled CUDA library.
    typedef struct GPUMul32 GPUMul32_t; 
    typedef struct GPUMul64 GPUMul64_t; 

    // Initialize a new GPU multiplier
    GPUMul32_t *mul_new_32(uint64_t m, uint64_t k, uint64_t n);
    GPUMul64_t *mul_new_64(uint64_t m, uint64_t k, uint64_t n);
    
    // Free a GPU multiplier
    void mul_free_32(GPUMul32_t *ctx);
    void mul_free_64(GPUMul64_t *ctx);

    // Init state for performing a GEMM with matrices of size:
    //     (m x k) * (k x n) + (m x n)
    void allocate_32(GPUMul32_t *ctx, uint64_t m, uint64_t k, uint64_t n);
    void allocate_64(GPUMul64_t *ctx, uint64_t m, uint64_t k, uint64_t n);

    // Set a new batch size for the GEMM computation
    void set_batch_32(GPUMul32_t *ctx, uint64_t n);
    void set_batch_64(GPUMul64_t *ctx, uint64_t n);

    // Get a raw pointer to host data of a specified matrix in
    // D = AB + C
    //  * 0 -> A
    //  * 1 -> B
    //  * 2 -> C
    //  * 3 -> D
    uint32_t* get_host_a_32(GPUMul32_t *ctx);
    uint32_t* get_host_a_64(GPUMul64_t *ctx);
    uint32_t* get_host_data_32(GPUMul32_t *ctx, int index);
    uint64_t* get_host_data_64(GPUMul64_t *ctx, int index);

    // Copy host data to the GPU device for the specified matrix
    void sync_device_32(GPUMul32_t *ctx, int index);
    void sync_device_64(GPUMul64_t *ctx, int index);

    // Execute GEMM
    void gemm_32(GPUMul32_t *ctx);
    void gemm_64(GPUMul64_t *ctx);

    // Check if a GPU is available to use
    bool use_gpu();
#ifdef __cplusplus
}
#endif


