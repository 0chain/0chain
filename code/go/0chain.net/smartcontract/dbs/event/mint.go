package event

type Mint struct {
	TransactionHash string `json:"transaction_hash"`
	Amount          int64  `json:"amount"`
}

func (edb *EventDb) addMint(mint Mint) error {
	return edb.Store.Get().Exec("UPDATE transactions SET mint_total_amount = mint_total_amount + ? WHERE hash = ?;", mint.Amount, mint.TransactionHash).Error
}
