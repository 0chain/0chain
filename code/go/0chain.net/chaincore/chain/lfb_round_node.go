package chain

import (
	"fmt"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

var (
	LFBRoundKey = encryption.RawHash("latest_finalized_block_round")
)

// LfbRound represents the LFB round info
//
//go:generate msgp -v -io=false -tests=false
type LfbRound struct {
	Round            int64  `msg:"r"`
	Hash             string `msg:"b"`
	MagicBlockNumber int64  `msg:"mb_num"`
}

// StoreLFBRound stores LFB round to state DB
func (c *Chain) StoreLFBRound(round, magicBlockNum int64, blockHash string) error {
	logging.Logger.Debug("[mvc] store lfb",
		zap.Int64("round", round),
		zap.String("block", blockHash),
		zap.Int64("mb number", magicBlockNum))
	lfbr := &LfbRound{
		Round:            round,
		Hash:             blockHash,
		MagicBlockNumber: magicBlockNum,
	}
	vn := util.NewValueNode()
	vn.SetValue(lfbr)

	return c.stateDB.PutNode(LFBRoundKey, vn)
}

// LoadLFBRound loads LFB round info from state DB
func (c *Chain) LoadLFBRound() (*LfbRound, error) {
	nd, err := c.stateDB.GetNode(LFBRoundKey)
	if err != nil {
		return nil, err
	}

	vn, ok := nd.(*util.ValueNode)
	if !ok {
		return nil, fmt.Errorf("invalid node type")
	}

	lfbr := &LfbRound{}
	d, err := vn.GetValue().MarshalMsg(nil)
	if err != nil {
		return nil, fmt.Errorf("encode value node for lfb failed: %v", err)
	}

	_, err = lfbr.UnmarshalMsg(d)
	if err != nil {
		return nil, fmt.Errorf("could not decode LFBRound: %v", err)
	}

	return lfbr, nil
}
