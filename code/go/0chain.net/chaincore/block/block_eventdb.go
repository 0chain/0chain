package block

import (
	"fmt"

	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
)

func blockToBlockEvent(block *Block) *event.Block {
	return &event.Block{
		Hash:                  block.Hash,
		Version:               block.Version,
		CreationDate:          int64(block.CreationDate.Duration()),
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
		RunningTxnCount:       fmt.Sprintf("%d", block.RunningTxnCount),
		RoundTimeoutCount:     block.RoundTimeoutCount,
		CreatedAt:             block.CreationDateField.ToTime(),
	}
}

func CreateBlockEvent(block *Block) (error, event.Event) {
	// todo block.Round is zero, need to replace with block/round number
	return nil, event.Event{
		BlockNumber: block.Round,
		TxHash:      "",
		Round:       block.Round,
		Type:        int(event.TypeStats),
		Tag:         int(event.TagAddBlock),
		Index:       block.Hash,
		Data:        blockToBlockEvent(block),
	}
}
