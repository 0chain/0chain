package encryption

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/herumi/mcl/ffi/go/mcl"
)

func TestBLS0ChainGenerateKeys(t *testing.T) {
	b0scheme := NewBLS0ChainScheme()
	b0scheme.GenerateKeys()
}

func TestBLS0ChainWriteKeys(t *testing.T) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sigScheme.WriteKeys(os.Stdout)
}

func TestBLS0ChainReadKeys(t *testing.T) {
	str := `4123d01678a8b9a9cec8315241710093bb50de802ec79cdb22df28d8ced81720f7637e5db8a4f6037f89daecaff7a223caee9d71cb101107e1da024545141883
30fb9f7b7228a53f383a4647e6694646ceee0bdc015cf42bc3bbec8326302613`
	reader := bytes.NewReader([]byte(str))
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.ReadKeys(reader)
	if err != nil {
		panic(err)
	}
}

func BenchmarkBLS0ChainGenerateKeys(b *testing.B) {
	sigScheme := NewBLS0ChainScheme()
	for i := 0; i < b.N; i++ {
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
	}
}

func TestBLS0ChainSignAndVerify(t *testing.T) {
	sigScheme := NewBLS0ChainScheme()
	sigScheme.GenerateKeys()
	signature, err := sigScheme.Sign(expectedHash)
	if err != nil {
		panic(err)
	}
	fmt.Printf("signature: %T %v\n", signature, signature)
	if ok, err := sigScheme.Verify(signature, expectedHash); err != nil || !ok {
		fmt.Printf("Verification failed\n")
	} else {
		fmt.Printf("Signing Verification successful\n")
	}
}

func BenchmarkBLS0ChainSign(b *testing.B) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		sigScheme.Sign(expectedHash)
	}
}

func BenchmarkBLS0ChainVerify(b *testing.B) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	signature, err := sigScheme.Sign(expectedHash)
	if err != nil {
		return
	}
	for i := 0; i < b.N; i++ {
		ok, err := sigScheme.Verify(signature, expectedHash)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("sig verification failed")
		}
	}
}

func BenchmarkBLS0ChainPairMessageHash(b *testing.B) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		sigScheme.PairMessageHash(expectedHash)
	}
}

func BenchmarkBLS0ChainG1HashToPoint(b *testing.B) {
	var g1 mcl.G1
	rawHash := RawHash("bls-0chain-signature-scheme")
	for i := 0; i < b.N; i++ {
		g1.HashAndMapTo(rawHash)
	}
}
