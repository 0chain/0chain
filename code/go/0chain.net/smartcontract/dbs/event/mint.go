package event

import "0chain.net/core/datastore"

type Mint struct {
	Minter     datastore.Key `json:"minter"`
	ToClientID datastore.Key `json:"to"`
	Amount     int64         `json:"amount"`
}

func (edb *EventDb) addMint(m Mint) error {
	return edb.Get().Model(&Mint{}).Create(m).Error
}
