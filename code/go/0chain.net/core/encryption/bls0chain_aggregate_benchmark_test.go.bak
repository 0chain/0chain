package encryption

import (
	"fmt"
	"testing"

	"github.com/herumi/bls-go-binary/bls"
)

func setupBLSAggregateData(n int) ([]bls.PublicKey, []bls.Sign, [][]byte) {
	pubVec := make([]bls.PublicKey, n)
	sigVec := make([]bls.Sign, n)
	h := make([][]byte, n)

	for i := 0; i < n; i++ {
		sec := new(bls.SecretKey)
		sec.SetByCSPRNG()
		pubVec[i] = *sec.GetPublicKey()
		m := fmt.Sprintf("abc-%d", i)
		h[i] = []byte(Hash(m))
		sigVec[i] = *sec.SignHash(h[i])
	}

	return pubVec, sigVec, h
}

func BenchmarkBLSAggregateSignature(b *testing.B) {
	// Setup test data outside benchmark timing
	n := 1000
	pubVec, sigVec, h := setupBLSAggregateData(n)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var sig bls.Sign
		sig.Aggregate(sigVec)

		// Verify the aggregated signature
		if !sig.VerifyAggregateHashes(pubVec, h) {
			b.Fatalf("Signature verification failed")
		}
	}
}

// Benchmark just the aggregation operation
func BenchmarkBLSSignatureAggregationOnly(b *testing.B) {
	// Setup test data outside benchmark timing
	n := 1000
	_, sigVec, _ := setupBLSAggregateData(n)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var sig bls.Sign
		sig.Aggregate(sigVec)
	}
}

// Benchmark just the verification operation
func BenchmarkBLSAggregateVerificationOnly(b *testing.B) {
	// Setup test data outside benchmark timing
	n := 1000
	pubVec, sigVec, h := setupBLSAggregateData(n)

	var sig bls.Sign
	sig.Aggregate(sigVec)

	// Reset timer to exclude setup time
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if !sig.VerifyAggregateHashes(pubVec, h) {
			b.Fatalf("Signature verification failed")
		}
	}
}

// Benchmark with different numbers of signatures
func BenchmarkBLSAggregateSignatureScaling(b *testing.B) {
	sizes := []int{10, 100, 500, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size-%d", size), func(b *testing.B) {
			// Setup test data outside benchmark timing
			pubVec, sigVec, h := setupBLSAggregateData(size)

			// Reset timer to exclude setup time
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var sig bls.Sign
				sig.Aggregate(sigVec)

				if !sig.VerifyAggregateHashes(pubVec, h) {
					b.Fatalf("Signature verification failed")
				}
			}
		})
	}
}
