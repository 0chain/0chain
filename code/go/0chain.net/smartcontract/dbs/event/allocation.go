package event

import (
	"0chain.net/chaincore/state"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type Allocation struct {
	AllocationID               string `gorm:"uniqueIndex"`
	TransactionID              string
	DataShards                 int
	ParityShards               int
	Size                       int64
	Expiration                 int64
	Terms                      []*AllocationTerm
	Owner                      string
	OwnerPublicKey             string
	IsImmutable                bool
	ReadPriceMin               state.Balance
	ReadPriceMax               state.Balance
	WritePriceMin              state.Balance
	WritePriceMax              state.Balance
	MaxChallengeCompletionTime int64
	ChallengeCompletionTime    int64
	StartTime                  int64
	Finalized                  bool
	Cancelled                  bool
	UsedSize                   int64
	MovedToChallenge           state.Balance
	MovedBack                  state.Balance
	MovedToValidators          state.Balance
	Curators                   []string
	TimeUnit                   int64
	NumWrites                  int64
	NumReads                   int64
	TotalChallenges            int64
	OpenChallenges             int64
	SuccessfulChallenges       int64
	FailedChallenges           int64
	LatestClosedChallengeTxn   string
}

type AllocationTerm struct {
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

func (edb *EventDb) overwriteAllocation(alloc *Allocation) error {

	result := edb.Store.Get().
		Model(&Allocation{}).
		Where(&Allocation{AllocationID: alloc.AllocationID}).
		Updates(alloc)

	return result.Error
}

func (edb *EventDb) addOrOverwriteAllocation(alloc *Allocation) error {

	exists, err := alloc.exists(edb)
	if err != nil {
		return err
	}

	if exists {
		return edb.overwriteAllocation(alloc)
	}

	result := edb.Store.Get().Create(&alloc)
	return result.Error
}

func (alloc *Allocation) exists(edb *EventDb) (bool, error) {

	var data Allocation

	result := edb.Store.Get().
		Model(&Allocation{}).
		Where(&Allocation{AllocationID: alloc.AllocationID}).
		Take(&data)

	if result.Error == gorm.ErrRecordNotFound {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for allocation %v, error %v",
			alloc.AllocationID, result.Error)
	}

	return true, nil
}
