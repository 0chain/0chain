package block

import (
	"encoding/json"
	"sort"

	"github.com/0chain/common/core/logging"

	"go.uber.org/zap"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/encryption"
)

//go:generate msgp -io=false -tests=false -v

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
	var keys []string
	for key, share := range sos.ShareOrSigns {
		if share == nil {
			continue
		}
		if share.Sign != "" {
			signatureScheme := scheme
			pk, ok := publicKeys[key]
			if !ok {
				return nil, false
			}
			if err := signatureScheme.SetPublicKey(pk); err != nil {
				logging.Logger.Error("failed to validate share or signs", zap.Any("share", share), zap.String("message", share.Message), zap.String("sign", share.Sign))
				return nil, false
			}
			sigOK, err := signatureScheme.Verify(share.Sign, share.Message)
			if !sigOK || err != nil {
				logging.Logger.Error("failed to validate share or signs", zap.Any("share", share), zap.String("message", share.Message), zap.String("sign", share.Sign))
				return nil, false
			}
		} else {
			var sij bls.Key
			if err := sij.SetHexString(share.Share); err != nil {
				return nil, false
			}
			pks, err := bls.ConvertStringToMpk(mpks.Mpks[sos.ID].Mpk)
			if err != nil {
				logging.Logger.Error("failed to convert mpks", zap.Error(err))
				return nil, false
			}

			if !bls.ValidateShare(pks, sij, bls.ComputeIDdkg(key)) {
				logging.Logger.Error("failed to validate share or signs", zap.Any("share", share), zap.String("sij.pi", sij.GetPublicKey().GetHexString()))
				return nil, false
			}
			keys = append(keys, key)
		}
	}
	return keys, true
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
