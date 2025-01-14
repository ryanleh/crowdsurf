#include <stdint.h>
#include <stddef.h>

// Hard-coded, to allow for compiler optimizations:
#define COMPRESSION_32 3
#define BASIS_32       10
#define BASIS2_32      BASIS_32*2
#define MASK_32        (1<<BASIS_32)-1

#define COMPRESSION_64 2
#define BASIS_64       30
#define MASK_64        (1<<BASIS_64)-1

typedef uint32_t Elem32;
typedef uint64_t Elem64;

void matMul32(Elem32 *out, const Elem32 *a, const Elem32 *b,
    size_t aRows, size_t aCols, size_t bCols);

void matMulVec32(Elem32 *out, const Elem32 *a, const Elem32 *b,
    size_t aRows, size_t aCols);

void matMulVecPacked32(Elem32 *out, const Elem32 *a, const Elem32 *b,
    size_t aRows, size_t aCols);

void randMatMul32(Elem32* out, const uint8_t *a, const Elem32 *b,
    size_t aRows, size_t aCols, size_t bCols);

void matMul32by64(Elem64 *out, const Elem32 *a, const Elem64 *b,
    size_t aRows, size_t aCols, size_t bCols);

void matMul64(Elem64 *out, const Elem64 *a, const Elem64 *b,
    size_t aRows, size_t aCols, size_t bCols);

void matMul32Vec64(Elem64 *out, const Elem32 *a, const Elem64 *b,
    size_t aRows, size_t aCols);

void matMulVec64(Elem64 *out, const Elem64 *a, const Elem64 *b,
    size_t aRows, size_t aCols);

void matMul32VecPacked64(Elem64 *out, const Elem32 *a, const Elem64 *b,
    size_t aRows, size_t aCols);

void matMulVecPacked64(Elem64 *out, const Elem64 *a, const Elem64 *b,
    size_t aRows, size_t aCols);

void randMatMul64(Elem64* out, const uint8_t *a, const Elem64 *b,
    size_t aRows, size_t aCols, size_t bCols);
