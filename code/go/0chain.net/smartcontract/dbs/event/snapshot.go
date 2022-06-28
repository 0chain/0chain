package event

// swagger:model Block
type Snapshot struct {
	Round           int64  `gorm:"primaryKey;autoIncrement:false" json:"round"`
	BlockHash       string `json:"block_hash"`
	MintTotalAmount int64  `json:"mint_total_amount"`
}

func (edb *EventDb) GetRoundsMintTotal(from, to int64) (int64, error) {
	var total int64
	return total, edb.Store.Get().Model(&Snapshot{}).Where("round between ? and ?", from, to).Select("sum(mint_total_amount)").Scan(&total).Error
}

func (edb *EventDb) addOrUpdateTotalMint(mint *Mint) error {
	res := edb.Store.Get().Table("snapshots").Where("round = ?", mint.Round).Update("block_hash", "mint_total_amount")
	return res.Error
}

func (edb *EventDb) addSnapshot(snapshot *Snapshot) error {
	result := edb.Store.Get().Create(snapshot)
	return result.Error
}
