package block

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"fmt"
	"github.com/0chain/common/core/util"
	"time"
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
		FinalityDuration:      time.Since(common.ToTime(block.CreationDate)).Milliseconds(),
		FinalizationTime:      time.Now(),
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
		Version:     event.Version1,
	}
}
