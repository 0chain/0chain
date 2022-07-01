package event

type Mint struct {
	BlockHash string `json:"block_hash"`
	Round     int64  `json:"round"`
	Amount    int64  `json:"amount"`
}
