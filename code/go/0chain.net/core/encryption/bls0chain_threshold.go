package encryption

import "github.com/herumi/bls/ffi/go/bls"

//BLS0ChainThresholdScheme - a scheme that can create threshold signature shares for BLS0Chain signature scheme
type BLS0ChainThresholdScheme struct {
	BLS0ChainScheme
	id bls.ID
}

type BLS0ChainReconstruction struct {
	t, n int
	ids  []bls.ID
	sigs []bls.Sign
}

//NewBLS0ChainThresholdScheme - create a new instance
func NewBLS0ChainThresholdScheme() *BLS0ChainThresholdScheme {
	return &BLS0ChainThresholdScheme{}
}

func (tss *BLS0ChainThresholdScheme) SetID(id string) error {
	return tss.id.SetHexString(id)
}

func (tss *BLS0ChainThresholdScheme) GetID() string {
	return tss.id.GetHexString()
}

//NewBLS0ChainReconstruction - create a new instance
func NewBLS0ChainReconstruction(t, n int) *BLS0ChainReconstruction {
	return &BLS0ChainReconstruction{
		t:    t,
		n:    n,
		ids:  []bls.ID{},
		sigs: []bls.Sign{},
	}
}

//Add - implement interface
func (rec *BLS0ChainReconstruction) Add(tss ThresholdSignatureScheme, signature string) error {
	b0tss, ok := tss.(*BLS0ChainThresholdScheme)
	if !ok {
		return ErrInvalidSignatureScheme
	}

	sig, err := b0tss.GetSignature(signature)
	if err != nil {
		return err
	}

	rec.ids = append(rec.ids, b0tss.id)
	rec.sigs = append(rec.sigs, *sig)

	return nil
}

//Reconstruct - implement interface
func (rec BLS0ChainReconstruction) Reconstruct() (string, error) {
	var s bls.Sign

	err := s.Recover(rec.sigs, rec.ids)
	if err != nil {
		return "", err
	}

	return s.SerializeToHexStr(), nil
}
