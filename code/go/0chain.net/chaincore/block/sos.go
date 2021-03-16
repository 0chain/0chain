package block

import (
	"encoding/json"
	"sort"

	. "0chain.net/core/logging"

	"go.uber.org/zap"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/encryption"
)

type ShareOrSigns struct {
	ID           string                      `json:"id"`
	ShareOrSigns map[string]*bls.DKGKeyShare `json:"share_or_sign"`
}

func NewShareOrSigns() *ShareOrSigns {
	return &ShareOrSigns{ShareOrSigns: make(map[string]*bls.DKGKeyShare)}
}

func (sos *ShareOrSigns) Hash() string {
	data := sos.ID
	var keys []string
	for k := range sos.ShareOrSigns {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		data += string(sos.ShareOrSigns[k].Encode())
	}
	return encryption.Hash(data)
}

func (sos *ShareOrSigns) Validate(mpks *Mpks, publicKeys map[string]string, scheme encryption.SignatureScheme) ([]string, bool) {
	var shares []string
	for key, share := range sos.ShareOrSigns {
		if share.Sign != "" {
			signatureScheme := scheme
			pk, ok := publicKeys[key]
			if !ok {
				return nil, false
			}
			if err := signatureScheme.SetPublicKey(pk); err != nil {
				return nil, false
			}
			sigOK, err := signatureScheme.Verify(share.Sign, share.Message)
			if !sigOK || err != nil {
				Logger.Error("failed to validate share or sings", zap.Any("share", share), zap.Any("message", share.Message), zap.Any("sign", share.Sign))
				return nil, false
			}
		} else {
			var sij bls.Key
			if err := sij.SetHexString(share.Share); err != nil {
				return nil, false
			}
			if !bls.ValidateShare(bls.ConvertStringToMpk(mpks.Mpks[sos.ID].Mpk), sij, bls.ComputeIDdkg(key)) {
				Logger.Error("failed to validate share or sings", zap.Any("share", share), zap.Any("sij.pi", sij.GetPublicKey().GetHexString()))
				return nil, false
			}
			shares = append(shares, key)
		}
	}
	return shares, true
}

func (sos *ShareOrSigns) Encode() []byte {
	buff, _ := json.Marshal(sos)
	return buff
}

func (sos *ShareOrSigns) Decode(input []byte) error {
	return json.Unmarshal(input, sos)
}

func (sos *ShareOrSigns) Clone() *ShareOrSigns {
	clone := &ShareOrSigns{
		ID:           sos.ID,
		ShareOrSigns: make(map[string]*bls.DKGKeyShare, len(sos.ShareOrSigns)),
	}
	for key, dkg := range sos.ShareOrSigns {
		clone.ShareOrSigns[key] = &bls.DKGKeyShare{
			IDField: dkg.IDField,
			Message: dkg.Message,
			Share:   dkg.Share,
			Sign:    dkg.Sign,
		}
	}
	return clone
}
