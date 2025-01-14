#include "matrix.h"
#include <stdio.h>

void matMul32by64(Elem64 *out, const Elem32 *a, const Elem64 *b,
    size_t aRows, size_t aCols, size_t bCols)
{
  for (size_t i = 0; i < aRows; i++) {
    for (size_t k = 0; k < aCols; k++) {
      for (size_t j = 0; j < bCols; j++) {
        out[bCols*i + j] += ((Elem64)a[aCols*i + k])*b[bCols*k + j];
      }
    }
  }
}

void matMul64(Elem64 *out, const Elem64 *a, const Elem64 *b,
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

void matMul32Vec64(Elem64 *out, const Elem32 *a, const Elem64 *b,
    size_t aRows, size_t aCols)
{
  Elem64 tmp;
  for (size_t i = 0; i < aRows; i++) {
    tmp = 0;
    for (size_t j = 0; j < aCols; j++) {
      tmp += ((Elem64) a[aCols*i + j])*b[j];
    }
    out[i] = tmp;
  }
}

void matMulVec64(Elem64 *out, const Elem64 *a, const Elem64 *b,
    size_t aRows, size_t aCols)
{
  Elem64 tmp;
  for (size_t i = 0; i < aRows; i++) {
    tmp = 0;
    for (size_t j = 0; j < aCols; j++) {
      tmp += a[aCols*i + j]*b[j];
    }
    out[i] = tmp;
  }
}

void randMatMul64(Elem64* out, const uint8_t *a, const Elem64 *b,
    size_t aRows, size_t aCols, size_t bCols)
{
  Elem64 val;
  Elem64 start = 0;

  for (size_t i = 0; i < aRows; i++) {
    for (size_t j = 0; j < aCols; j++) {
      val = ((Elem64)a[start+0]) |
	    (((Elem64)a[start+1]) << 8) |
	    (((Elem64)a[start+2]) << 16) |
	    (((Elem64)a[start+3]) << 24) |
	    (((Elem64)a[start+4]) << 32) |
	    (((Elem64)a[start+5]) << 40) |
	    (((Elem64)a[start+6]) << 48) |
	    (((Elem64)a[start+7]) << 56);

      start += 8;

      for (size_t k = 0; k < bCols; k++) {
        out[bCols*i + k] += val * b[bCols*j + k];
      }
    }
  }
}

void matMul32VecPacked64(Elem64 *out, const Elem32 *a, const Elem64 *b,
    size_t aRows, size_t aCols)
{
  Elem32 db, db2, db3, db4, db5, db6, db7, db8;
  Elem64 val, val2, val3, val4, val5, val6, val7, val8;
  Elem64 tmp, tmp2, tmp3, tmp4, tmp5, tmp6, tmp7, tmp8;
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

      val  = (Elem64)(db & MASK_32);
      val2 = (Elem64)(db2 & MASK_32);
      val3 = (Elem64)(db3 & MASK_32);
      val4 = (Elem64)(db4 & MASK_32);
      val5 = (Elem64)(db5 & MASK_32);
      val6 = (Elem64)(db6 & MASK_32);
      val7 = (Elem64)(db7 & MASK_32);
      val8 = (Elem64)(db8 & MASK_32);
      tmp  += val*b[index2];
      tmp2 += val2*b[index2];
      tmp3 += val3*b[index2];
      tmp4 += val4*b[index2];
      tmp5 += val5*b[index2];
      tmp6 += val6*b[index2];
      tmp7 += val7*b[index2];
      tmp8 += val8*b[index2];
      index2 += 1;

      val  = (Elem64)((db >> BASIS_32) & MASK_32);
      val2 = (Elem64)((db2 >> BASIS_32) & MASK_32);
      val3 = (Elem64)((db3 >> BASIS_32) & MASK_32);
      val4 = (Elem64)((db4 >> BASIS_32) & MASK_32);
      val5 = (Elem64)((db5 >> BASIS_32) & MASK_32);
      val6 = (Elem64)((db6 >> BASIS_32) & MASK_32);
      val7 = (Elem64)((db7 >> BASIS_32) & MASK_32);
      val8 = (Elem64)((db8 >> BASIS_32) & MASK_32);
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


void matMulVecPacked64(Elem64 *out, const Elem64 *a, const Elem64 *b,
    size_t aRows, size_t aCols)
{
  Elem64 db, db2, db3, db4, db5, db6, db7, db8;
  Elem64 val, val2, val3, val4, val5, val6, val7, val8;
  Elem64 tmp, tmp2, tmp3, tmp4, tmp5, tmp6, tmp7, tmp8;
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

      val  = db & MASK_64;
      val2 = db2 & MASK_64;
      val3 = db3 & MASK_64;
      val4 = db4 & MASK_64;
      val5 = db5 & MASK_64;
      val6 = db6 & MASK_64;
      val7 = db7 & MASK_64;
      val8 = db8 & MASK_64;
      tmp  += val*b[index2];
      tmp2 += val2*b[index2];
      tmp3 += val3*b[index2];
      tmp4 += val4*b[index2];
      tmp5 += val5*b[index2];
      tmp6 += val6*b[index2];
      tmp7 += val7*b[index2];
      tmp8 += val8*b[index2];
      index2 += 1;

      val  = (db >> BASIS_64) & MASK_64;
      val2 = (db2 >> BASIS_64) & MASK_64;
      val3 = (db3 >> BASIS_64) & MASK_64;
      val4 = (db4 >> BASIS_64) & MASK_64;
      val5 = (db5 >> BASIS_64) & MASK_64;
      val6 = (db6 >> BASIS_64) & MASK_64;
      val7 = (db7 >> BASIS_64) & MASK_64;
      val8 = (db8 >> BASIS_64) & MASK_64;
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

