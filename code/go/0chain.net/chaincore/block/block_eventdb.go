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
		CreationDate:          block.CreationDate.Duration().Nanoseconds(),
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
		ChainWeight:           fmt.Sprintf("%f", block.ChainWeight),
		RunningTxnCount:       fmt.Sprintf("%d", block.RunningTxnCount),
		RoundTimeoutCount:     block.RoundTimeoutCount,
		CreatedAt:             block.CreationDateField.ToTime(),
	}
}

func emitBlockEvent(block *Block) error {
	if len(block.Txns) == 0 {
		return fmt.Errorf("no transaction for the block")
	}

	data, err := json.Marshal(blockToBlockEvent(block))
	if err != nil {
		return fmt.Errorf("error marshalling block: %v", err)
	}

	t := block.Txns[len(block.Txns)-1]
	block.Events = append(block.Events, event.Event{
		BlockNumber: block.Round,
		TxHash:      t.Hash,
		Type:        int(event.TypeStats),
		Tag:         int(event.TagAddBlock),
		Index:       t.Hash,
		Data:        string(data),
	})

	return nil
}
