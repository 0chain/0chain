package event

import (
	"time"

	"0chain.net/smartcontract/common"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

// swagger:model Block
type Block struct {
	gorm.Model

	Hash                  string    `json:"hash" gorm:"uniqueIndex:idx_bhash"`
	Version               string    `json:"version"`
	CreationDate          int64     `json:"creation_date" gorm:"index:idx_bcreation_date"`
	Round                 int64     `json:"round" gorm:"index:idx_bround"`
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
	RunningTxnCount       string    `json:"running_txn_count"`
	RoundTimeoutCount     int       `json:"round_timeout_count"`
	CreatedAt             time.Time `json:"created_at"`
}

func (edb *EventDb) GetBlockByHash(hash string) (Block, error) {
	block := Block{}
	res := edb.Store.Get().Table("blocks").Where("hash = ?", hash).First(&block)
	return block, res.Error
}

func (edb *EventDb) GetBlockByRound(round int64) (Block, error) {
	block := Block{}
	res := edb.Store.Get().Table("blocks").Where(Block{Round: round}).First(&block)
	return block, res.Error
}

func (edb *EventDb) GetBlockByDate(date string) (Block, error) {
	block := Block{}

	return block, edb.Store.Get().Table("blocks").Where("creation_date <= ?", date).Limit(1).Order(
		clause.OrderByColumn{
			Column: clause.Column{Name: "creation_date"},
			Desc:   true,
		},
	).Scan(&block).Error
}

func (edb *EventDb) GetBlocksByRound(round string) (Block, error) {
	block := Block{}
	res := edb.Store.Get().Table("blocks").Where("round = ?", round).Scan(&block)
	return block, res.Error
}

func (edb *EventDb) GetBlocks(start, end int64, limit common.Pagination) ([]Block, error) {
	var blocks []Block
	res := edb.Store.Get().Table("blocks").
		Where("round >= ? AND round < ?", start, end).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "round"},
			Desc:   limit.IsDescending,
		}).Find(&blocks)
	return blocks, res.Error
}

func (edb *EventDb) addBlock(block Block) error {
	result := edb.Store.Get().Create(&block)
	return result.Error
}
