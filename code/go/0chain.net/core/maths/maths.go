package maths

import (
	"fmt"
	"math"

	"github.com/0chain/common/core/currency"
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

// SafeAddUInt64 adds two uint64 and returns an error if there is an overflow
func SafeAddUInt64(left, right uint64) (uint64, error) {

	if left > math.MaxUint64-right {
		return 0, currency.ErrIntAddOverflow
	}

	return left + right, nil
}

// SafeAddInt32 adds two integers and returns an error if there is overflows
func SafeAddInt32(left, right int32) (int32, error) {
	if right > 0 {
		if left > math.MaxInt32-right {
			return 0, currency.ErrInt32AddOverflow
		}
	} else {
		if left < math.MinInt32-right {
			return 0, currency.ErrInt32AddOverflow
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
		return result, fmt.Errorf("overflow multiplying %v and %v", a, b)
	}
	return result, nil
}
