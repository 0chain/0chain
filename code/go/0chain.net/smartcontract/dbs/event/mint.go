package event

type Mint struct {
	Minter          string `json:"minter"`
	ToClientID      string `json:"to"`
	TransactionHash string `json:"transaction_hash"`
	Amount          int64  `json:"amount"`
}

func (edb *EventDb) addMint(m Mint) error {
	return edb.Get().Model(&Mint{}).Create(m).Error
}
