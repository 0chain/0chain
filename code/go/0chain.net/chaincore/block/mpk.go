package block

import (
	"encoding/json"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

type MPK struct {
	ID  string
	Mpk []string
}

type Mpks struct {
	Mpks map[string]*MPK
}

func NewMpks() *Mpks {
	return &Mpks{Mpks: make(map[string]*MPK)}
}

func (mpks *Mpks) Encode() []byte {
	buff, _ := json.Marshal(mpks)
	return buff
}

func (mpks *Mpks) Decode(input []byte) error {
	return json.Unmarshal(input, mpks)
}

func (mpks *Mpks) GetHash() string {
	return util.ToHex(mpks.GetHashBytes())
}

func (mpks *Mpks) GetHashBytes() []byte {
	return encryption.RawHash(mpks.Encode())
}

func (mpks *Mpks) GetMpkMap() map[bls.PartyID][]bls.PublicKey {
	mpkMap := make(map[bls.PartyID][]bls.PublicKey)
	for k, v := range mpks.Mpks {
		mpkMap[bls.ComputeIDdkg(k)] = bls.ConvertStringToMpk(v.Mpk)
	}
	return mpkMap
}

func (mpks *Mpks) GetMpks() map[string]*MPK {
	result := make(map[string]*MPK, len(mpks.Mpks))
	for k, v := range mpks.Mpks {
		result[k] = v
	}
	return result
}

func (mpk *MPK) Encode() []byte {
	buff, _ := json.Marshal(mpk)
	return buff
}

func (mpk *MPK) Decode(input []byte) error {
	return json.Unmarshal(input, mpk)
}
