package block

import (
	"fmt"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/util"
)

func blockToBlockEvent(block *Block) *event.Block {
	return &event.Block{
		Hash:                  block.Hash,
		Version:               block.Version,
		CreationDate:          int64(block.CreationDate.Duration().Seconds()),
		Round:                 block.Round,
		MinerID:               block.MinerID,
		RoundRandomSeed:       block.RoundRandomSeed,
		MerkleTreeRoot:        block.GetMerkleTree().GetRoot(),
		StateHash:             util.ToHex(block.ClientStateHash),
		ReceiptMerkleTreeRoot: block.GetReceiptsMerkleTree().GetRoot(),
		NumTxns:               len(block.Txns),
		MagicBlockHash:        block.LatestFinalizedMagicBlockHash,
		PrevHash:              block.PrevHash,
		Signature:             block.Signature,
		ChainId:               block.ChainID,
		StateChangesCount:     block.StateChangesCount,
		RunningTxnCount:       fmt.Sprintf("%d", block.RunningTxnCount),
		RoundTimeoutCount:     block.RoundTimeoutCount,
		IsFinalised:           block.IsBlockFinalised(),
	}
}

func CreateBlockEvent(block *Block) (error, event.Event) {
	logging.Logger.Info("create block event", zap.String("blockHash", block.Hash), zap.Int64("round", block.Round))
	// todo block.Round is zero, need to replace with block/round number
	return nil, event.Event{
		BlockNumber: block.Round,
		TxHash:      "",
		Type:        event.TypeChain,
		Tag:         event.TagAddBlock,
		Index:       block.Hash,
		Data:        blockToBlockEvent(block),
	}
}

func CreateFinalizeBlockEvent(block *Block) event.Event {
	return event.Event{
		BlockNumber: block.Round,
		TxHash:      "",
		Type:        event.TypeChain,
		Tag:         event.TagFinalizeBlock,
		Index:       block.Hash,
		Data:        blockToBlockEvent(block),
	}
}
