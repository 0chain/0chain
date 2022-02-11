package maths

import "math"

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
