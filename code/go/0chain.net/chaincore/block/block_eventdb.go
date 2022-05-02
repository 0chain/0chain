package block

import (
	"encoding/json"
	"fmt"

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
		StateHash:             string(block.ClientStateHash),
		ReceiptMerkleTreeRoot: block.GetReceiptsMerkleTree().GetRoot(),
		NumTxns:               len(block.Txns),
		MagicBlockHash:        block.MagicBlock.Hash,
		PrevHash:              block.PrevHash,
		Signature:             block.Signature,
		ChainId:               block.ChainID,
		RunningTxnCount:       fmt.Sprintf("%d", block.RunningTxnCount),
		RoundTimeoutCount:     block.RoundTimeoutCount,
		CreatedAt:             block.CreationDateField.ToTime(),
	}
}

func emitBlockEvent(block *Block) error {
	data, err := json.Marshal(blockToBlockEvent(block))
	if err != nil {
		return fmt.Errorf("error marshalling block: %v", err)
	}

	block.Events = append(block.Events, event.Event{
		BlockNumber: block.Round,
		TxHash:      "",
		Type:        int(event.TypeStats),
		Tag:         int(event.TagAddBlock),
		Index:       block.Hash,
		Data:        string(data),
	})

	return nil
}
