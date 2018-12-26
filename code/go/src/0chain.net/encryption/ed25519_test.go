package encryption

import (
	"bytes"
	"fmt"
	"testing"
)

func TestED25519GenerateKeys(t *testing.T) {
	sigScheme := NewED25519Scheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
}

func BenchmarkED25519GenerateKeys(b *testing.B) {
	sigScheme := NewED25519Scheme()
	for i := 0; i < b.N; i++ {
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
	}
}

func TestED25519SignAndVerify(t *testing.T) {
	sigScheme := NewED25519Scheme()
	buffer := bytes.NewBuffer([]byte("e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0\naa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"))
	sigScheme.ReadKeys(buffer)
	signature, err := sigScheme.Sign(expectedHash)
	if err != nil {
		panic(err)
	}
	fmt.Printf("signature: %v\n", signature)
	if ok, err := sigScheme.Verify(signature, expectedHash); err != nil || !ok {
		fmt.Printf("Verification failed\n")
	} else {
		fmt.Printf("Signing Verification successful\n")
	}
}

func BenchmarkED25519Sign(b *testing.B) {
	sigScheme := NewED25519Scheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		sigScheme.Sign(expectedHash)
	}
}

func BenchmarkED25519Verify(b *testing.B) {
	sigScheme := NewED25519Scheme()
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
