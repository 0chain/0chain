package event

type Mint struct {
	BlockHash string `json:"block_hash"`
	Amount    int64  `json:"amount"`
}

func (edb *EventDb) addMint(mint Mint) error {
	return edb.Store.Get().Model(&Block{}).Where(&Block{Hash: mint.BlockHash}).Update("mint_total_amount", mint.Amount).Error
}
