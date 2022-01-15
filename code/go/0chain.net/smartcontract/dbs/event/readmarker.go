package event

import "gorm.io/gorm"

type ReadMarker struct {
	gorm.Model
	ClientID      string `json:"client_id"`
	BlobberID     string `json:"blobber_id"`
	AllocationID  string `json:"allocation_id"`
	TransactionID string `json:"transaction_id"`
	OwnerID       string `json:"owner_id"`
	Timestamp     int64  `json:"timestamp"`
	ReadCounter   int64  `json:"read_counter"`
	ReadSize      int64  `json:"read_size"`
	Signature     string `json:"signature"`
	PayerID       string `json:"payer_id"`
	AuthTicket    string `json:"auth_ticket"`
	BlockNumber   int64  `json:"block_number"`
}
