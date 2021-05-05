package encryption

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/herumi/mcl/ffi/go/mcl"
	"github.com/herumi/bls/ffi/go/bls"
	"github.com/stretchr/testify/require"
)

func TestMiraclToHerumiPK(t *testing.T) {
	miraclpk1 := `0418a02c6bd223ae0dfda1d2f9a3c81726ab436ce5e9d17c531ff0a385a13a0b491bdfed3a85690775ee35c61678957aaba7b1a1899438829f1dc94248d87ed36817f6dfafec19bfa87bf791a4d694f43fec227ae6f5a867490e30328cac05eaff039ac7dfc3364e851ebd2631ea6f1685609fc66d50223cc696cb59ff2fee47ac`
	pk1 := MiraclToHerumiPK(miraclpk1)

	// Assert DeserializeHexStr works on the output of MiraclToHerumiPK
	var pk bls.PublicKey
	err := pk.DeserializeHexStr(pk1)
	require.NoError(t, err)
}

func TestMiraclToHerumiSig(t *testing.T) {
	miraclsig1 := `(0d4dbad6d2586d5e01b6b7fbad77e4adfa81212c52b4a0b885e19c58e0944764,110061aa16d5ba36eef0ad4503be346908d3513c0a2aedfd0d2923411b420eca)`
	sig1 := MiraclToHerumiSig(miraclsig1)

	// Assert DeserializeHexStr works on the output of MiraclToHerumiSig
	var sig bls.Sign
	err := sig.DeserializeHexStr(sig1)
	require.NoError(t, err)

	// Test that passing in normal herumi sig just gets back the original.
	sig2 := MiraclToHerumiSig(sig1)
	if sig1 != sig2 {
		panic("Sigs should've been the same")
	}
}

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
