package transaction

import (
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/util"
)

/*Confirmation - a data structure that provides the confirmation that a transaction is included into the block chain */
type Confirmation struct {
	Version           string       `json:"version"`
	Hash              string       `json:"hash"`
	BlockHash         string       `json:"block_hash"`
	PreviousBlockHash string       `json:"previous_block_hash"`
	Transaction       *Transaction `json:"txn,omitempty"`
	datastore.CreationDateField
	MinerID               datastore.Key `json:"miner_id"`
	Round                 int64         `json:"round"`
	Status                int           `json:"transaction_status" msgpack:"sot"`
	RoundRandomSeed       int64         `json:"round_random_seed"`
	StateChangesCount     int           `json:"state_changes_count"`
	MerkleTreeRoot        string        `json:"merkle_tree_root"`
	MerkleTreePath        *util.MTPath  `json:"merkle_tree_path"`
	ReceiptMerkleTreeRoot string        `json:"receipt_merkle_tree_root"`
	ReceiptMerkleTreePath *util.MTPath  `json:"receipt_merkle_tree_path"`
}

func (c *Confirmation) GetHash() string {
	return c.Hash
}

func (c *Confirmation) GetHashBytes() []byte {
	return util.HashStringToBytes(c.Hash)
}
