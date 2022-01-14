package event

import (
	"time"

	"gorm.io/gorm"
)

type Block struct {
	gorm.Model

	Hash                  string    `json:"hash"`
	Version               string    `json:"version"`
	CreationDate          int64     `json:"creation_date"`
	Round                 int64     `json:"round"`
	MinerID               string    `json:"miner_id"`
	RoundRandomSeed       int64     `json:"round_random_seed"`
	MerkleTreeRoot        string    `json:"merkle_tree_root"`
	StateHash             string    `json:"state_hash"`
	ReceiptMerkleTreeRoot string    `json:"receipt_merkle_tree_root"`
	NumTxns               int       `json:"num_txns"`
	MagicBlockHash        string    `json:"magic_block_hash"`
	PrevHash              string    `json:"prev_hash"`
	Signature             string    `json:"signature"`
	ChainId               string    `json:"chain_id"`
	ChainWeight           string    `json:"chain_weight"`
	RunningTxnCount       string    `json:"running_txn_count"`
	RoundTimeoutCount     int       `json:"round_timeout_count"`
	CreatedAt             time.Time `json:"created_at"`
}

//func (edb *EventDb) overwriteWriteMarker(wm WriteMarker) error {
//	result := edb.Store.Get().
//		Model(&WriteMarker{}).
//		Where(&WriteMarker{TransactionID: wm.TransactionID}).
//		Updates(&wm)
//	return result.Error
//}
//
func (edb *EventDb) addBlock(block Block) error {
	result := edb.Store.Get().Create(&block)
	return result.Error
}

//
//func (wm *WriteMarker) exists(edb *EventDb) (bool, error) {
//	var count int64
//	result := edb.Get().
//		Model(&WriteMarker{}).
//		Where(&WriteMarker{TransactionID: wm.TransactionID}).
//		Count(&count)
//	if result.Error != nil {
//		return false, fmt.Errorf("error searching for write marker txn: %v, error %v",
//			wm.TransactionID, result.Error)
//	}
//	return count > 0, nil
//
