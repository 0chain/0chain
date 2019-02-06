package pbc

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"testing"

	"0chain.net/encryption"
)

var expectedHash = encryption.Hash("0chain.net rocks")

const TestParams = `type a
q 5259155439993369056735419436966908943008385784785727827197299182131573002019230479029150212799344040225927213370752990286232124554252433487267771407370603
h 7196920353230679389589566924613173176268801991709735896178777296411299139041484748330445667584121333844332
r 730750818665451459101842416367364881864821047297
exp2 159
exp1 63
sign1 1
sign0 1`

var TestSharedG = []byte{73, 219, 220, 101, 1, 145, 12, 247, 92, 13, 59, 239, 243, 204, 117, 166, 73, 171, 160, 248, 126, 70, 242, 168, 39, 130, 197, 206, 127, 218, 95, 161, 233, 125, 5, 243, 119, 113, 171, 26, 52, 103, 179, 155, 109, 82, 176, 216, 12, 32, 31, 196, 118, 105, 85, 48, 230, 179, 159, 79, 127, 90, 121, 102, 72, 199, 39, 38, 184, 178, 200, 47, 243, 146, 205, 222, 113, 141, 253, 80, 48, 80, 101, 122, 86, 119, 197, 75, 182, 19, 196, 72, 102, 206, 253, 205, 179, 178, 34, 65, 128, 239, 128, 191, 47, 153, 254, 128, 255, 47, 255, 39, 141, 90, 78, 7, 185, 42, 42, 98, 235, 108, 194, 66, 109, 32, 111, 35}

func TestBLS0ChainInitSetup(t *testing.T) {
	params, sharedG := BLS0ChainInitialSetup()
	buf := bytes.NewBuffer(nil)
	writer := bufio.NewWriter(buf)
	err := BLS0ChainSerialize(params, sharedG, writer)
	if err != nil {
		panic(err)
	}
	writer.Flush()
	dsParams, err := BLS0ChainDeserialize(buf)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", dsParams)
}

func TestBLS0ChainGenerateKeys(t *testing.T) {
	bls0chainParams := &BLS0ChainParams{Params: TestParams, SharedG: TestSharedG}
	b0scheme := NewBLS0ChainScheme(bls0chainParams)
	b0scheme.GenerateKeys()
}

func TestBLS0ChainWriteKeys(t *testing.T) {
	bls0chainParams := &BLS0ChainParams{Params: TestParams, SharedG: TestSharedG}
	sigScheme := NewBLS0ChainScheme(bls0chainParams)
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sigScheme.WriteKeys(os.Stdout)
}

func TestBLS0ChainReadKeys(t *testing.T) {
	str := `16ad585d7ba4920de26c82a3fcffcb76650648fb7f2f42aa3ca3f8e58f32f6d4f6dfab1169c8c2049b45a42d2b6f7e95a8b601cd02a08ae8218e41aae80a9dc14d674cfd0f6b3eec35d87d031c1f2b2020c3b02c4c2de13af94c26859edd7bb647015b484603a3e330ee02a4ba50e4a346200fbc0a28ade80592463f12d6887f
5bb93c450e8bcb4b4b50799e70dfe10c60e7ef52`
	reader := bytes.NewReader([]byte(str))
	bls0chainParams := &BLS0ChainParams{Params: TestParams, SharedG: TestSharedG}
	sigScheme := NewBLS0ChainScheme(bls0chainParams)
	err := sigScheme.ReadKeys(reader)
	if err != nil {
		panic(err)
	}
}

func BenchmarkBLS0ChainGenerateKeys(b *testing.B) {
	bls0chainParams := &BLS0ChainParams{Params: TestParams, SharedG: TestSharedG}
	sigScheme := NewBLS0ChainScheme(bls0chainParams)
	for i := 0; i < b.N; i++ {
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
	}
}

func TestBLS0ChainSignAndVerify(t *testing.T) {
	bls0chainParams := &BLS0ChainParams{Params: TestParams, SharedG: TestSharedG}
	sigScheme := NewBLS0ChainScheme(bls0chainParams)
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
	bls0chainParams := &BLS0ChainParams{Params: TestParams, SharedG: TestSharedG}
	sigScheme := NewBLS0ChainScheme(bls0chainParams)
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		sigScheme.Sign(expectedHash)
	}
}

func BenchmarkBLS0ChainVerify(b *testing.B) {
	bls0chainParams := &BLS0ChainParams{Params: TestParams, SharedG: TestSharedG}
	sigScheme := NewBLS0ChainScheme(bls0chainParams)
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
	bls0chainParams := &BLS0ChainParams{Params: TestParams, SharedG: TestSharedG}
	sigScheme := NewBLS0ChainScheme(bls0chainParams)
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		sigScheme.PairMessageHash(expectedHash)
	}
}

func BenchmarkBLS0ChainG1HashToPoint(b *testing.B) {
	bls0chainParams := &BLS0ChainParams{Params: TestParams, SharedG: TestSharedG}
	sigScheme := NewBLS0ChainScheme(bls0chainParams)
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	rawHash := encryption.RawHash("bls-0chain-signature-scheme")
	for i := 0; i < b.N; i++ {
		sigScheme.pairing.NewG1().SetFromHash(rawHash)
	}
}
