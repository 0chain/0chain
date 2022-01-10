package event

import (
	"0chain.net/chaincore/state"
	"time"
)

type Allocation struct {
	AllocationID   string `gorm:"uniqueIndex"`
	TransactionID  string
	DataShards     int
	ParityShards   int
	Size           int64
	Expiration     int64
	Terms          []*AllocationTerms
	Owner          string
	OwnerPublicKey string
	IsImmutable    bool
}

type AllocationTerms struct {
	BlobberID string
	// ReadPrice is price for reading. Token / GB (no time unit).
	ReadPrice state.Balance `json:"read_price"`
	// WritePrice is price for reading. Token / GB / time unit. Also,
	// it used to calculate min_lock_demand value.
	WritePrice state.Balance `json:"write_price"`
	// MinLockDemand in number in [0; 1] range. It represents part of
	// allocation should be locked for the blobber rewards even if
	// user never write something to the blobber.
	MinLockDemand float64 `json:"min_lock_demand"`
	// MaxOfferDuration with this prices and the demand.
	MaxOfferDuration time.Duration `json:"max_offer_duration"`
	// ChallengeCompletionTime is duration required to complete a challenge.
	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
}
