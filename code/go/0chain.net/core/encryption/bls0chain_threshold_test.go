package encryption

import (
	"testing"
)

func TestThresholdSignatures(t *testing.T) {
	T := 7
	N := 10
	scheme := "bls0chain"
	msg := "1234567890"

	groupKey := GetSignatureScheme(scheme)
	err := groupKey.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	shares, err := GenerateThresholdKeyShares(scheme, T, N, groupKey)
	if err != nil {
		t.Fatal(err)
	}

	var sigs []string
	for _, share := range shares {
		sig, err := share.Sign(msg)
		if err != nil {
			t.Fatal(err)
		}

		sigs = append(sigs, sig)
	}

	rec := GetReconstructSignatureScheme(scheme, T, N)

	for i, share := range shares {
		err := rec.Add(share, sigs[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	recovered, err := rec.Reconstruct()
	if err != nil {
		t.Fatal(err)
	}

	ok, err := groupKey.Verify(recovered, msg)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Error("Reconstructed signature did not verify")
	}
}
