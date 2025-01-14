package lwe

import (
	mrand "math/rand"
)

// CDF table for gaussian with std-dev 3.2 to match SEALs parameters
var cdf_table = [...]float64{
	0.5, 0.952345, 0.822578, 0.644389, 0.457833,
	0.295023, 0.172422, 0.0913938, 0.0439369, 0.0191572,
	0.00757568, 0.00271706, 0.000883826, 0.000260749, 6.97696e-05,
	1.69316e-05, 3.72665e-06, 7.43923e-07, 1.34687e-07, 2.21163e-08,
	3.29371e-09, 4.44886e-10, 5.45004e-11, 6.05535e-12, 6.10194e-13,
	5.57679e-14, 4.62263e-15, 3.47522e-16, 2.36954e-17, 1.46533e-18,
	8.21851e-20, 4.18062e-21, 1.92875e-22, 8.07049e-24, 3.06275e-25,
	1.05418e-26, 3.29081e-28, 9.31708e-30, 2.39247e-31, 5.57187e-33,
	1.17691e-34, 2.25463e-36, 3.91737e-38, 6.1731e-40, 8.82266e-42,
	1.14363e-43, 1.34449e-45, 1.43357e-47, 1.38634e-49, 1.21593e-51,
	9.67246e-54, 6.97835e-56, 4.56622e-58, 2.70987e-60, 1.45858e-62,
	7.12032e-65, 3.15252e-67, 1.26591e-69, 4.6104e-72, 1.52287e-74,
	4.56219e-77, 1.23958e-79, 3.05465e-82, 6.82713e-85, 1.3839e-87,
}

// The function below is modeled on Martin Albrecht's discrete-Gaussian
// sampler included in his dgs library:
//
//	https://github.com/malb/dgs
func gaussSample(src mrand.Source, cdf_table []float64) int64 {
	math_src := mrand.New(src)

	var x int64
	var y float64
	for {
		x = int64(math_src.Intn(len(cdf_table)))
		y = math_src.Float64()

		if y < cdf_table[x] {
			break
		}
	}

	if math_src.Uint64()%2 == 0 {
		x = -x
	}

	return x
}

func GaussSample32(src mrand.Source) int64 {
	return gaussSample(src, cdf_table[:])
}

func GaussSample64(src mrand.Source) int64 {
	return gaussSample(src, cdf_table[:])
}
