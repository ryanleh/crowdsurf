#include "matrix.h"
#include <stdio.h>

void matMul32(Elem32 *out, const Elem32 *a, const Elem32 *b,
    size_t aRows, size_t aCols, size_t bCols)
{
  for (size_t i = 0; i < aRows; i++) {
    for (size_t k = 0; k < aCols; k++) {
      for (size_t j = 0; j < bCols; j++) {
        out[bCols*i + j] += a[aCols*i + k]*b[bCols*k + j];
      }
    }
  }
}

void matMulVec32(Elem32 *out, const Elem32 *a, const Elem32 *b,
    size_t aRows, size_t aCols)
{
  Elem32 tmp;
  for (size_t i = 0; i < aRows; i++) {
    tmp = 0;
    for (size_t j = 0; j < aCols; j++) {
      tmp += a[aCols*i + j]*b[j];
    }
    out[i] = tmp;
  }
}

void randMatMul32(Elem32* out, const uint8_t *a, const Elem32 *b,
    size_t aRows, size_t aCols, size_t bCols)
{
  Elem32 val;
  Elem64 start = 0;

  for (size_t i = 0; i < aRows; i++) {
    for (size_t j = 0; j < aCols; j++) {
      val = ((Elem32)a[start+0]) |
	    (((Elem32)a[start+1]) << 8) |
	    (((Elem32)a[start+2]) << 16) |
	    (((Elem32)a[start+3]) << 24);

      start += 4;

      for (size_t k = 0; k < bCols; k++) {
        out[bCols*i + k] += val * b[bCols*j + k];
      }
    }
  }
}

void matMulVecPacked32(Elem32 *out, const Elem32 *a, const Elem32 *b,
    size_t aRows, size_t aCols)
{
  Elem32 db, db2, db3, db4, db5, db6, db7, db8;
  Elem32 val, val2, val3, val4, val5, val6, val7, val8;
  Elem32 tmp, tmp2, tmp3, tmp4, tmp5, tmp6, tmp7, tmp8;
  size_t index = 0;
  size_t index2;

  for (size_t i = 0; i < aRows; i += 8) {
    tmp  = 0;
    tmp2 = 0;
    tmp3 = 0;
    tmp4 = 0;
    tmp5 = 0;
    tmp6 = 0;
    tmp7 = 0;
    tmp8 = 0;

    index2 = 0;
    for (size_t j = 0; j < aCols; j++) {
      db  = a[index];
      db2 = a[index+1*aCols];
      db3 = a[index+2*aCols];
      db4 = a[index+3*aCols];
      db5 = a[index+4*aCols];
      db6 = a[index+5*aCols];
      db7 = a[index+6*aCols];
      db8 = a[index+7*aCols];

      val  = db & MASK_32;
      val2 = db2 & MASK_32;
      val3 = db3 & MASK_32;
      val4 = db4 & MASK_32;
      val5 = db5 & MASK_32;
      val6 = db6 & MASK_32;
      val7 = db7 & MASK_32;
      val8 = db8 & MASK_32;
      tmp  += val*b[index2];
      tmp2 += val2*b[index2];
      tmp3 += val3*b[index2];
      tmp4 += val4*b[index2];
      tmp5 += val5*b[index2];
      tmp6 += val6*b[index2];
      tmp7 += val7*b[index2];
      tmp8 += val8*b[index2];
      index2 += 1;

      val  = (db >> BASIS_32) & MASK_32;
      val2 = (db2 >> BASIS_32) & MASK_32;
      val3 = (db3 >> BASIS_32) & MASK_32;
      val4 = (db4 >> BASIS_32) & MASK_32;
      val5 = (db5 >> BASIS_32) & MASK_32;
      val6 = (db6 >> BASIS_32) & MASK_32;
      val7 = (db7 >> BASIS_32) & MASK_32;
      val8 = (db8 >> BASIS_32) & MASK_32;
      tmp  += val*b[index2];
      tmp2 += val2*b[index2];
      tmp3 += val3*b[index2];
      tmp4 += val4*b[index2];
      tmp5 += val5*b[index2];
      tmp6 += val6*b[index2];
      tmp7 += val7*b[index2];
      tmp8 += val8*b[index2];
      index2 += 1;

      val  = (db >> BASIS2_32) & MASK_32;
      val2 = (db2 >> BASIS2_32) & MASK_32;
      val3 = (db3 >> BASIS2_32) & MASK_32;
      val4 = (db4 >> BASIS2_32) & MASK_32;
      val5 = (db5 >> BASIS2_32) & MASK_32;
      val6 = (db6 >> BASIS2_32) & MASK_32;
      val7 = (db7 >> BASIS2_32) & MASK_32;
      val8 = (db8 >> BASIS2_32) & MASK_32;
      tmp  += val*b[index2];
      tmp2 += val2*b[index2];
      tmp3 += val3*b[index2];
      tmp4 += val4*b[index2];
      tmp5 += val5*b[index2];
      tmp6 += val6*b[index2];
      tmp7 += val7*b[index2];
      tmp8 += val8*b[index2];
      index2 += 1;
      index += 1;
    }
    out[i]   += tmp;
    out[i+1] += tmp2;
    out[i+2] += tmp3;
    out[i+3] += tmp4;
    out[i+4] += tmp5;
    out[i+5] += tmp6;
    out[i+6] += tmp7;
    out[i+7] += tmp8;
    index += aCols*7;
  }
}

