package encryption

import (
	"fmt"
	"os"
	"testing"
)

var numSplits = 2

func TestGenerateSplitKeys(t *testing.T) {
	b0Sig := NewBLS0ChainScheme()
	b0Sig.GenerateKeys()
	splittableSigScheme := b0Sig
	fmt.Printf("split key set 1\n")
	genSplitKeys(splittableSigScheme)
	fmt.Printf("split key set 2\n")
	genSplitKeys(splittableSigScheme)
}

func genSplitKeys(splittableSigScheme SplittableSignatureScheme) {
	splitKeys, err := splittableSigScheme.GenerateSplitKeys(numSplits)
	if err != nil {
		panic(err)
	}
	if len(splitKeys) != numSplits {
		panic("Num split keys not same as numSplits")
	}
	for _, splitKey := range splitKeys {
		splitKey.WriteKeys(os.Stdout)
		fmt.Println()
	}
}

func TestValidateSplitKeys(t *testing.T) {
	b0Sig := NewBLS0ChainScheme()
	b0Sig.GenerateKeys()
	splittableSigScheme := b0Sig
	splitKeys, err := splittableSigScheme.GenerateSplitKeys(numSplits)
	if err != nil {
		panic(err)
	}
	signature, err := b0Sig.Sign(expectedHash)
	if err != nil {
		panic(err)
	}

	signatures := make([]string, numSplits)
	for idx, splitKey := range splitKeys {
		signature, err := splitKey.Sign(expectedHash)
		if err != nil {
			panic(err)
		}
		signatures[idx] = signature
	}
	aggSignature, err := splittableSigScheme.AggregateSignatures(signatures)
	if signature != aggSignature {
		panic("signature mismatch!")
	} else {
		fmt.Printf("%v\n%v\n", signature, aggSignature)
	}
}
