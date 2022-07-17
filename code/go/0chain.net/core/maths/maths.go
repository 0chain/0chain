package maths

import (
	"fmt"
	"math"

	"0chain.net/chaincore/currency"
)

// GetGamma gets gamma for blobber block reward
// A, B, alpha are constants
// X is total data stored in blobber
// R is data read by blobber
func GetGamma(A, B, alpha, X, R float64) float64 {

	if X == 0 {
		return 0
	}

	factor := math.Abs((alpha*X - R) / (alpha*X + R))
	return A - B*factor
}

func GetZeta(i, k, mu, wp, rp float64) float64 {

	if wp == 0 {
		return 0
	}

	return i - (k * (rp / (rp + (mu * wp))))
}

// SafeAddInt64 adds two integers and returns an error if there is overflows
func SafeAddInt64(left, right int64) (int64, error) {
	if right > 0 {
		if left > math.MaxInt64-right {
			return 0, currency.ErrInt64AddOverflow
		}
	} else {
		if left < math.MinInt64-right {
			return 0, currency.ErrInt64AddOverflow
		}
	}
	return left + right, nil
}

// SafeMultInt64 multiplies two integers and returns an error if there is overflows
func SafeMultInt64(a, b int64) (int64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	result := a * b
	if result/b != a {
		return result, fmt.Errorf("Overflow multiplying %v and %v", a, b)
	}
	return result, nil
}

// SafeAddFloat64 adds two integers and returns an error if there is overflows
func SafeAddFloat64(left, right float64) (float64, error) {
	if right > 0 {
		if left > math.MaxFloat64-right {
			return 0, currency.ErrFloat64AddOverflow
		}
	} else {
		if left < -math.MaxFloat64-right { // for floating point numbers MinFloat64 == -MaxFloat64
			return 0, currency.ErrFloat64AddOverflow
		}
	}
	return left + right, nil
}

// SafeMultFloat64 multiplies two float64 and returns an error if there is overflows
func SafeMultFloat64(left, right float64) (float64, error) {
	if left == 0 || right == 0 {
		return 0, nil
	}

	result := left * right
	// if result == math.Inf(1) || result == math.Inf(-1) {
	// 	return result, fmt.Errorf("Overflow multiplying %v and %v, result: %v", left, right, result)
	// }
	return result, nil
}
