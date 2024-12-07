package event

import (
	"errors"
	"time"

	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"gorm.io/gorm/clause"
)

// swagger:model Block
type Block struct {
	model.UpdatableModel

	Hash                  string    `json:"hash" gorm:"uniqueIndex:idx_bhash"`
	Version               string    `json:"version"`
	CreationDate          int64     `json:"creation_date"`
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
	StateChangesCount     int       `json:"state_changes_count"`
	RunningTxnCount       string    `json:"running_txn_count"`
	RoundTimeoutCount     int       `json:"round_timeout_count"`
	FinalityDuration      int64     `json:"finality_duration"`
	FinalizationTime      time.Time `json:"-" gorm:"-"` //NOT TO BE STORED in db, in state, or to be sent in kafka
}

func (edb *EventDb) GetRoundFromTime(at time.Time, asc bool) (int64, error) {
	round := struct {
		Round int64 `json:"round" gorm:"index:idx_bround"`
	}{}
	var direction, sign string
	if asc {
		direction = "asc"
		sign = ">="
	} else {
		sign = "<="
		direction = "desc"
	}

	if res := edb.Store.Get().
		Table("blocks").
		Where("created_at "+sign+" ?", at).
		Order("round " + direction).
		First(&round); res.Error != nil {
		return 0, res.Error
	}
	return round.Round, nil
}

func (edb *EventDb) GetBlockByHash(hash string) (Block, error) {
	var b Block
	res := edb.Store.Get().
		Where(&Block{Hash: hash}).
		Find(&b)
	if res.RowsAffected == 0 {
		return b, errors.New("record not found")
	}
	return b, res.Error
}

func (edb *EventDb) GetBlockByRound(round int64) (Block, error) {
	var b Block
	res := edb.Store.Get().
		Where(&Block{Round: round}).
		Find(&b)
	if res.RowsAffected == 0 {
		return b, errors.New("record not found")
	}
	return b, res.Error
}

func (edb *EventDb) GetBlockByDate(date string) (Block, error) {
	block := Block{}

	return block, edb.Store.Get().Table("blocks").
		Where("creation_date <= ?", date).
		Limit(1).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "round"},
			Desc:   true,
		}).
		Scan(&block).Error
}

func (edb *EventDb) GetBlocksByRound(round string) (Block, error) {
	block := Block{}
	res := edb.Store.Get().Table("blocks").Where("round = ?", round).Scan(&block)
	return block, res.Error
}

func (edb *EventDb) GetBlocksByBlockNumbers(start, end int64, limit common.Pagination) ([]Block, error) {
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

func (edb *EventDb) GetBlocks(limit common.Pagination) ([]Block, error) {
	var blocks []Block
	res := edb.Store.Get().Table("blocks").
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "round"},
			Desc:   limit.IsDescending,
		}).Find(&blocks)
	return blocks, res.Error
}

func (edb *EventDb) addOrUpdateBlock(block Block) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hash"}, {Name: "round"}},
		UpdateAll: true,
	}).Create(&block).Error
}
