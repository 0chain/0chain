package chain

import (
	"fmt"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

var (
	LFBRoundKey = encryption.RawHash("latest_finalized_block_round")
)

//go:generate msgp -v -io=false -tests=false
type LfbRound struct {
	Round int64  `msg:"r"`
	Hash  string `msg:"b"`
}

func (c *Chain) StoreLFBRound(round int64, blockHash string) error {
	lfbr := &LfbRound{
		Round: round,
		Hash:  blockHash,
	}
	vn := util.NewValueNode()
	vn.SetValue(lfbr)

	return c.stateDB.PutNode(LFBRoundKey, vn)
}

func (c *Chain) LoadLFBRound(round int64, blockHash string) (*LfbRound, error) {
	nd, err := c.stateDB.GetNode(LFBRoundKey)
	if err != nil {
		return nil, fmt.Errorf("cound not found LFB round: %v", err)
	}

	lfbr := &LfbRound{}
	_, err = lfbr.UnmarshalMsg(nd.Encode())
	if err != nil {
		return nil, fmt.Errorf("could not decode LFBRound: %v", err)
	}

	return lfbr, nil
}
