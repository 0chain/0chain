package encryption

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

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
		sigSchemes[i].GenerateKeys()
		msgs[i] = fmt.Sprintf("testing aggregate messages : %v", i)
		msgHashes[i] = Hash(msgs[i])
		sig, err := sigSchemes[i].Sign(msgHashes[i])
		if err != nil {
			panic(err)
		}
		msgSignatures[i] = sig
	}
	aggregate := true
	aggSigScheme := GetAggregateSignatureScheme(clientSignatureScheme, total, batchSize)
	if aggSigScheme == nil {
		aggregate = false
	}
	ts := time.Now()
	if aggregate {
		var wg sync.WaitGroup
		for t := 0; t < numBatches; t++ {
			wg.Add(1)
			go func(bn int) {
				start := bn * batchSize
				for i := 0; i < batchSize; i++ {
					aggSigScheme.Aggregate(sigSchemes[start+i], start+i, msgSignatures[start+i], msgHashes[start+i])
				}
				wg.Done()
			}(t)
		}
		wg.Wait()
		result, err := aggSigScheme.Verify()
		if err != nil {
			panic(err)
		}
		if !result {
			panic("signature verification failed")
		}
	} else {
		var wg sync.WaitGroup
		for t := 0; t < numBatches; t++ {
			wg.Add(1)
			go func(bn int) {
				start := bn * batchSize
				for i := 0; i < batchSize; i++ {
					result, err := sigSchemes[start+i].Verify(msgSignatures[start+i], msgHashes[start+i])
					if err != nil {
						panic(err)
					}
					if !result {
						panic("signature verification failed")
					}
				}
				wg.Done()
			}(t)
		}
		wg.Wait()
	}
	fmt.Printf("signature verification (scheme = %s , aggregate = %v) successful in %v\n", clientSignatureScheme, aggregate, time.Since(ts))
}
