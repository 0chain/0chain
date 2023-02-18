package entity

// BurnTicket represents client's burn ticket details
type BurnTicket struct {
	Hash  string `json:"hash"`
	Nonce int64  `json:"nonce"`
}
