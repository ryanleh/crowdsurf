#include <iostream>

#include "cutlass/cutlass.h"
#include "cutlass/gemm/device/gemm.h"
#include "cutlass/util/host_tensor.h"
#include "cutlass/util/reference/device/gemm.h"
#include "cutlass/util/reference/host/tensor_fill.h"
#include "utils.cuh"

template<typename T>
class GPUMul {
public:
    using Rows = cutlass::layout::RowMajor;
    using Cols = cutlass::layout::ColumnMajor;
    using Gemm = cutlass::gemm::device::Gemm<uint32_t, Rows, T, Cols, T, Rows>;

private:
    // The plaintext space is always < 32-bits for the parameters we consider
    cutlass::HostTensor<uint32_t, Rows> a_;
    // This is stored in column-major format so that when we modulus
    // switch a batch of queries, we can write them directly into host
    // memory without fragmented caching
    cutlass::HostTensor<T, Cols> b_;
    cutlass::HostTensor<T, Rows> c_;
    cutlass::HostTensor<T, Rows> d_;

    cutlass::device_memory::allocation<uint8_t> workspace_;
    Gemm gemm_op_;
    T alpha_ = T(1);
    T beta_ = T(1);
    int split_k_slices_ = 1;
    uint64_t _m, _k;

public:
    GPUMul() {
        // Check that our GPU supports the necessary operations
        cudaDeviceProp props;
        CUDA_CHECK(cudaGetDeviceProperties(&props, 0));
        if (props.major != 7) {
            std::cerr << "GPU must have compute capability >70" << std::endl;
            exit(1);
        }
    }

    GPUMul(uint64_t m, uint64_t k, uint64_t n): GPUMul() {
        allocate(m, k, n);
    }

    void allocate(uint64_t m, uint64_t k, uint64_t n) {
        _m = m;
        _k = k;
        cutlass::gemm::GemmCoord problem_size(m, n, k);
        
        // Allocate memory for each tensor
        a_.reset(problem_size.mk());
        b_.reset(problem_size.kn());
        c_.reset(problem_size.mn());
        d_.reset(problem_size.mn());

        // Zero-out tensor data
        cutlass::reference::host::TensorFill(a_.host_view());
        cutlass::reference::host::TensorFill(b_.host_view());
        cutlass::reference::host::TensorFill(c_.host_view());
        cutlass::reference::host::TensorFill(d_.host_view());
        for (int i = 0; i < 4; i++)
            sync_device(i);
                
        // Create a tuple of GEMM kernel arguments
        typename Gemm::Arguments arguments{
            problem_size,
            a_.device_ref(),
            b_.device_ref(),
            c_.device_ref(),
            d_.device_ref(),
            {alpha_, beta_},
            split_k_slices_
        };

        // Allocate necessary workspace on device
        int workspace_size = Gemm::get_workspace_size(arguments);
        workspace_ = cutlass::device_memory::allocation<uint8_t>(workspace_size);

        // Instantiate CUTLASS kernel
        cutlass::Status status = gemm_op_.can_implement(arguments);
        CUTLASS_CHECK(status);

        status = gemm_op_.initialize(arguments, workspace_.get());
        CUTLASS_CHECK(status);
    }

    void set_batch(uint64_t n) {
        cutlass::gemm::GemmCoord problem_size(_m, n, _k);
        
        // Allocate memory for each tensor
        b_.reset(problem_size.kn());
        c_.reset(problem_size.mn());
        d_.reset(problem_size.mn());

        // Zero-out tensor data
        cutlass::reference::host::TensorFill(b_.host_view());
        cutlass::reference::host::TensorFill(c_.host_view());
        cutlass::reference::host::TensorFill(d_.host_view());
        for (int i = 1; i < 4; i++)
            sync_device(i);
                
        // Create a tuple of GEMM kernel arguments
        typename Gemm::Arguments arguments{
            problem_size,
            a_.device_ref(),
            b_.device_ref(),
            c_.device_ref(),
            d_.device_ref(),
            {alpha_, beta_},
            split_k_slices_
        };

        // Allocate necessary workspace on device
        int workspace_size = Gemm::get_workspace_size(arguments);
        workspace_ = cutlass::device_memory::allocation<uint8_t>(workspace_size);

        // Instantiate CUTLASS kernel
        cutlass::Status status = gemm_op_.can_implement(arguments);
        CUTLASS_CHECK(status);

        status = gemm_op_.initialize(arguments, workspace_.get());
        CUTLASS_CHECK(status);
    }


    uint32_t* raw_host_a() {
        return a_.host_data();
    }

    T* raw_host_data(int index) {
        switch (index) {
            case 1:
                return b_.host_data();
            case 2:
                return c_.host_data();
            case 3:
                return d_.host_data();
        }
        return nullptr;
    }

    void sync_device(int index) {
        switch (index) {
            case 0:
                a_.sync_device();
                break;
            case 1:
                b_.sync_device();
                break;
            case 2:
                c_.sync_device();
                break;
            case 3:
                d_.sync_device();
                break;
        }
    }

    void gemm() {
        // Execute kernel
        auto status = gemm_op_();
        CUTLASS_CHECK(status);
      
        // wait for kernels to finish
        cudaDeviceSynchronize();
      
        // Sync host with result device data
        d_.sync_host();
    }
};
