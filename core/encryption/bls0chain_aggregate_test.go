package encryption

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/herumi/bls/ffi/go/bls"
	"github.com/stretchr/testify/require"
)

func TestBLS0ChainAggSignature(t *testing.T) {
	total := 10
	sigSchemes := make([]SignatureScheme, total)
	msgs := make([]string, total)
	msgHashes := make([]string, total)
	msgSignatures := make([]string, total)
	pubKeys := make([]string, total)
	clientSignatureScheme := "bls0chain"
	for i := 0; i < total; i++ {
		sigSchemes[i] = GetSignatureScheme(clientSignatureScheme)
		err := sigSchemes[i].GenerateKeys()
		require.NoError(t, err)
		pubKeys[i] = sigSchemes[i].GetPublicKey()
		msgs[i] = fmt.Sprintf("testing aggregate messages : %v", i)
		msgHashes[i] = Hash(msgs[i])
		sig, err := sigSchemes[i].Sign(msgHashes[i])
		if err != nil {
			t.Fatal(err)
		}
		msgSignatures[i] = sig
	}

	//require.True(t, BLS0ChainAggregateHashesVerify(msgSignatures, msgHashes, pubKeys))
	//aggSign, err := BLS0ChainAggregateSignatures(msgSignatures)
	//require.NoError(t, err)
	//require.True(t, aggSign.BLS0ChainAggregateHashesVerify(pubKeys, msgHashes))
}

func HashA(buf []byte) []byte {
	if bls.GetOpUnitSize() == 4 {
		d := sha256.Sum256([]byte(buf))
		return d[:]
	}
	// use SHA512 if bitSize > 256
	d := sha512.Sum512([]byte(buf))
	return d[:]
}

func TestAAAA(t *testing.T) {
	n := 1000
	pubVec := make([]bls.PublicKey, n)
	sigVec := make([]bls.Sign, n)
	h := make([][]byte, n)
	for i := 0; i < n; i++ {
		//sigScheme := GetSignatureScheme(clientSignatureScheme)
		//err := sigScheme.GenerateKeys()
		//require.NoError(t, err)
		//sigScheme.(BLS)
		//
		sec := new(bls.SecretKey)
		sec.SetByCSPRNG()
		pubVec[i] = *sec.GetPublicKey()
		m := fmt.Sprintf("abc-%d", i)
		h[i] = []byte(Hash(m))
		sigVec[i] = *sec.SignHash(h[i])
	}

	var sig bls.Sign
	sig.Aggregate(sigVec)
	// aggregate sig
	//sig := sigVec[0]
	//for i := 1; i < n; i++ {
	//	sig.Add(sigVec[i])
	//}

	if !sig.VerifyAggregateHashes(pubVec, h) {
		t.Errorf("sig.VerifyAggregateHashes")
	}
}

//func BenchmarkAggregateSignaturesV2(b *testing.B) {
//	for j := 0; j < b.N; j++ {
//		total := 10
//		msgs := make([]string, total)
//		msgHashes := make([]string, total)
//		msgSignatures := make([]string, total)
//		clientSignatureScheme := "bls0chain"
//		pubkeys := make([]bls.PublicKey, total)
//		for i := 0; i < total; i++ {
//			sigScheme := GetSignatureScheme(clientSignatureScheme)
//			err := sigScheme.GenerateKeys()
//			require.NoError(b, err)
//
//			var pk bls.PublicKey
//			err = pk.DeserializeHexStr(sigScheme.GetPublicKey())
//			require.NoError(b, err)
//
//			pubkeys[i] = pk
//			msgs[i] = fmt.Sprintf("testing aggregate messages : %v", i)
//			msgHashes[i] = Hash(msgs[i])
//			sig, err := sigScheme.Sign(Hash(msgs[i]))
//			if err != nil {
//				b.Fatal(err)
//			}
//			msgSignatures[i] = sig
//		}
//
//		aggSig, err := BLS0ChainAggregateSignatures(msgSignatures)
//		require.NoError(b, err)
//		require.True(b, aggSig.VerifyAggregate(pubkeys, msgHashes))
//	}
//}

//func init() {
//gSeckeys := make([]bls.SecretKey, 1000)
//gPubkeys := make([]bls.PublicKey, 1000)
//}

func TestAggregateSignatures(t *testing.T) {
	total := 1000
	batchSize := 250
	numBatches := total / batchSize
	sigSchemes := make([]SignatureScheme, total)
	msgs := make([]string, total)
	msgHashes := make([]string, total)
	msgSignatures := make([]string, total)
	clientSignatureScheme := "bls0chain"
	for i := 0; i < total; i++ {
		sigSchemes[i] = GetSignatureScheme(clientSignatureScheme)
		err := sigSchemes[i].GenerateKeys()
		require.NoError(t, err)
		msgs[i] = fmt.Sprintf("testing aggregate messages : %v", i)
		msgHashes[i] = Hash(msgs[i])
		sig, err := sigSchemes[i].Sign(msgHashes[i])
		if err != nil {
			t.Fatal(err)
		}
		msgSignatures[i] = sig
	}
	aggregate := true
	aggSigScheme := GetAggregateSignatureScheme(clientSignatureScheme, total, batchSize)
	if aggSigScheme == nil {
		aggregate = false
	}
	if aggregate {
		var wg sync.WaitGroup
		for i := 0; i < numBatches; i++ {
			wg.Add(1)
			go func(bn int) {
				start := bn * batchSize
				for i := 0; i < batchSize; i++ {
					err := aggSigScheme.Aggregate(sigSchemes[start+i], start+i, msgSignatures[start+i], msgHashes[start+i])
					require.NoError(t, err)
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
		result, err := aggSigScheme.Verify()
		if err != nil {
			t.Fatal(err)
		}
		if !result {
			t.Error("signature verification failed")
		}
	} else {
		var wg sync.WaitGroup
		for tr := 0; tr < numBatches; tr++ {
			wg.Add(1)
			go func(bn int) {
				start := bn * batchSize
				for i := 0; i < batchSize; i++ {
					result, err := sigSchemes[start+i].Verify(msgSignatures[start+i], msgHashes[start+i])
					if err != nil {
						t.Error(err)
						return
					}
					if !result {
						t.Error("signature verification failed")
					}
				}
				wg.Done()
			}(tr)
		}
		wg.Wait()
	}
}

func TestNewBLS0ChainAggregateSignature(t *testing.T) {
	t.Parallel()

	type args struct {
		total     int
		batchSize int
	}
	tests := []struct {
		name string
		args args
		want *BLS0ChainAggregateSignatureScheme
	}{
		{
			name: "Test_NewBLS0ChainAggregateSignature_OK",
			args: args{total: 1, batchSize: 2},
			want: &BLS0ChainAggregateSignatureScheme{
				Total:     1,
				BatchSize: 2,
				ASigs:     make([]*bls.Sign, 1),
				AGt:       make([]*bls.GT, 1),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewBLS0ChainAggregateSignature(tt.args.total, tt.args.batchSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBLS0ChainAggregateSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBLS0ChainAggregateSignatureScheme_Aggregate(t *testing.T) {
	t.Parallel()

	scheme := NewBLS0ChainScheme()
	if err := scheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}

	hash := Hash("data")
	sign, err := scheme.Sign(hash)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Total     int
		BatchSize int
		ASigs     []*bls.Sign
		AGt       []*bls.GT
	}
	type args struct {
		ss        SignatureScheme
		idx       int
		signature string
		hash      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_BLS0ChainAggregateSignatureScheme_Aggregate_Invalid_Signature_Scheme_ERR",
			args:    args{ss: &ED25519Scheme{}},
			wantErr: true,
		},
		{
			name:    "Test_BLS0ChainAggregateSignatureScheme_Aggregate_Get_Signature_ERR",
			args:    args{ss: &BLS0ChainScheme{}, signature: ""},
			wantErr: true,
		},
		{
			name:    "Test_BLS0ChainAggregateSignatureScheme_Aggregate_Hex_Decoding_Hash_ERR",
			fields:  fields{BatchSize: 1, ASigs: make([]*bls.Sign, 2)},
			args:    args{ss: scheme, signature: sign, hash: "!", idx: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b0a := BLS0ChainAggregateSignatureScheme{
				Total:     tt.fields.Total,
				BatchSize: tt.fields.BatchSize,
				ASigs:     tt.fields.ASigs,
				AGt:       tt.fields.AGt,
			}
			if err := b0a.Aggregate(tt.args.ss, tt.args.idx, tt.args.signature, tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("Aggregate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
