package event

type Mint struct {
	BlockHash string `json:"block_hash"`
	Amount    int64  `json:"amount"`
}

func (edb *EventDb) addMint(mint Mint) error {
	return edb.Store.Get().Exec("UPDATE blocks SET mint_total_amount = mint_total_amount + ? WHERE hash = ?;", mint.Amount, mint.BlockHash).Error
}
