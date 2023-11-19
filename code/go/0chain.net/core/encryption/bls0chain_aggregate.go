package encryption

import (
	"errors"

	"github.com/herumi/bls/ffi/go/bls"
)

// BLS0ChainAggregateSignatureScheme - a scheme that can aggregate signatures for BLS0Chain signature scheme
type BLS0ChainAggregateSignatureScheme struct {
	Total     int
	BatchSize int
	ASigs     []*bls.Sign
	AGt       []*bls.GT
}

// NewBLS0ChainAggregateSignature - create a new instance
func NewBLS0ChainAggregateSignature(total int, batchSize int) *BLS0ChainAggregateSignatureScheme {
	b0a := &BLS0ChainAggregateSignatureScheme{Total: total, BatchSize: batchSize}
	numBatches := total / batchSize
	if numBatches*batchSize < total {
		numBatches++
	}
	b0a.ASigs = make([]*bls.Sign, numBatches)
	b0a.AGt = make([]*bls.GT, numBatches)
	return b0a
}

// Aggregate - implement interface
func (b0a BLS0ChainAggregateSignatureScheme) Aggregate(ss SignatureScheme, idx int, signature string, hash string) error {
	b0sig, ok := ss.(*BLS0ChainScheme)
	if !ok {
		return ErrInvalidSignatureScheme
	}
	sig, err := b0sig.GetSignature(signature)
	if err != nil {
		return err
	}
	batch := idx / b0a.BatchSize
	if b0a.ASigs[batch] == nil {
		b0a.ASigs[batch] = sig
	} else {
		b0a.ASigs[batch].Add(sig)
	}
	gt, err := b0sig.PairMessageHash(hash)
	if err != nil {
		return err
	}
	if b0a.AGt[batch] == nil {
		b0a.AGt[batch] = gt
	} else {
		bls.GTMul(b0a.AGt[batch], b0a.AGt[batch], gt)
	}
	return nil
}

// Verify - implement interface
func (b0a BLS0ChainAggregateSignatureScheme) Verify() (bool, error) {
	agtmul := b0a.AGt[0]
	asig := b0a.ASigs[0]
	for i := 1; i < len(b0a.AGt); i++ {
		bls.GTMul(agtmul, agtmul, b0a.AGt[i])
		asig.Add(b0a.ASigs[i])
	}
	var agg bls.GT
	var asigG1 bls.G1
	if err := asigG1.Deserialize(asig.Serialize()); err != nil {
		return false, err
	}
	bls.Pairing(&agg, &asigG1, GenG2)
	if !agg.IsEqual(agtmul) {
		return false, errors.New("aggregate signature validation failed")
	}
	return true, nil
}
