package block

import (
	"encoding/json"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

//go:generate msgp -io=false -tests=false -v

type MPK struct {
	ID  string
	Mpk []string
}

// swagger:model Mpks
type Mpks struct {
	Mpks map[string]*MPK
}

func NewMpks() *Mpks {
	return &Mpks{Mpks: make(map[string]*MPK)}
}

func (mpks *Mpks) Delete(id string) {
	delete(mpks.Mpks, id)
}

func (mpks *Mpks) Encode() []byte {
	buff, _ := json.Marshal(mpks)
	return buff
}

func (mpks *Mpks) Decode(input []byte) error {
	err := json.Unmarshal(input, mpks)
	if err != nil {
		return err
	}
	return nil
}

func (mpks *Mpks) GetHash() string {
	return util.ToHex(mpks.GetHashBytes())
}

func (mpks *Mpks) GetHashBytes() []byte {
	return encryption.RawHash(mpks.Encode())
}

func (mpks *Mpks) GetMpkMap() (map[bls.PartyID][]bls.PublicKey, error) {
	mpkMap := make(map[bls.PartyID][]bls.PublicKey)
	for k, v := range mpks.Mpks {
		mpk, err := bls.ConvertStringToMpk(v.Mpk)
		if err != nil {
			return nil, err
		}

		mpkMap[bls.ComputeIDdkg(k)] = mpk
	}
	return mpkMap, nil
}

func (mpks *Mpks) GetMpks() map[string]*MPK {
	result := make(map[string]*MPK, len(mpks.Mpks))
	for k, v := range mpks.Mpks {
		result[k] = v
	}
	return result
}

// Clone returns a clone of Mpks instance
func (mpks *Mpks) Clone() *Mpks {
	clone := &Mpks{Mpks: make(map[string]*MPK, len(mpks.Mpks))}
	for k, v := range mpks.Mpks {
		nv := *v
		nv.Mpk = make([]string, len(v.Mpk))
		copy(nv.Mpk, v.Mpk)
		clone.Mpks[k] = &nv
	}

	return clone
}

func (mpk *MPK) Encode() []byte {
	buff, _ := json.Marshal(mpk)
	return buff
}

func (mpk *MPK) Decode(input []byte) error {
	err := json.Unmarshal(input, mpk)
	if err != nil {
		return err
	}
	return nil
}
