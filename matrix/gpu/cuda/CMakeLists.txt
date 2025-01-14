cmake_minimum_required(VERSION 3.11.0)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# There are better ways to check for CUDA but they aren't working atm
if("$ENV{CUDACXX}" STREQUAL "")
    project(gpu LANGUAGES CXX)
    add_library(${PROJECT_NAME} SHARED ffi_noop.cpp)
else()
    set(CMAKE_CUDA_ARCHITECTURES 70)
    project(gpu LANGUAGES CUDA)
    add_library(${PROJECT_NAME} SHARED ffi.cu gemm.cu)
    target_include_directories(${PROJECT_NAME} PRIVATE
        ${CMAKE_SOURCE_DIR}/../../../external/cutlass/include
        ${CMAKE_SOURCE_DIR}/../../../external/cutlass/tools/util/include
    )

endif()
