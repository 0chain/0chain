package entity

// BurnTicketDetails represents client's burn ticket details
type BurnTicketDetails struct {
	Hash  string `json:"hash"`
	Nonce int64  `json:"nonce"`
}
