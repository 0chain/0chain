package event

import (
	"fmt"
	"time"

	"0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs"

	"0chain.net/chaincore/currency"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

type Allocation struct {
	gorm.Model
	AllocationID             string        `json:"allocation_id" gorm:"uniqueIndex"`
	AllocationName           string        `json:"allocation_name" gorm:"column:allocation_name;size:64;"`
	TransactionID            string        `json:"transaction_id"`
	DataShards               int           `json:"data_shards"`
	ParityShards             int           `json:"parity_shards"`
	Size                     int64         `json:"size"`
	Expiration               int64         `json:"expiration"`
	Terms                    string        `json:"terms"`
	Owner                    string        `json:"owner" gorm:"index:idx_aowner"`
	OwnerPublicKey           string        `json:"owner_public_key"`
	IsImmutable              bool          `json:"is_immutable"`
	ReadPriceMin             currency.Coin `json:"read_price_min"`
	ReadPriceMax             currency.Coin `json:"read_price_max"`
	WritePriceMin            currency.Coin `json:"write_price_min"`
	WritePriceMax            currency.Coin `json:"write_price_max"`
	ChallengeCompletionTime  int64         `json:"challenge_completion_time"`
	StartTime                int64         `json:"start_time" gorm:"index:idx_astart_time"`
	Finalized                bool          `json:"finalized"`
	Cancelled                bool          `json:"cancelled"`
	UsedSize                 int64         `json:"used_size"`
	MovedToChallenge         currency.Coin `json:"moved_to_challenge"`
	MovedBack                currency.Coin `json:"moved_back"`
	MovedToValidators        currency.Coin `json:"moved_to_validators"`
	TimeUnit                 int64         `json:"time_unit"`
	NumWrites                int64         `json:"num_writes"`
	NumReads                 int64         `json:"num_reads"`
	TotalChallenges          int64         `json:"total_challenges"`
	OpenChallenges           int64         `json:"open_challenges"`
	SuccessfulChallenges     int64         `json:"successful_challenges"`
	FailedChallenges         int64         `json:"failed_challenges"`
	LatestClosedChallengeTxn string        `json:"latest_closed_challenge_txn"`
	WritePool                currency.Coin `json:"write_pool"`
	//ref
	User User `gorm:"foreignKey:Owner;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type AllocationTerm struct {
	BlobberID    string `json:"blobber_id"`
	AllocationID string `json:"allocation_id"`
	// ReadPrice is price for reading. Token / GB (no time unit).
	ReadPrice currency.Coin `json:"read_price"`
	// WritePrice is price for reading. Token / GB / time unit. Also,
	// it used to calculate min_lock_demand value.
	WritePrice currency.Coin `json:"write_price"`
	// MinLockDemand in number in [0; 1] range. It represents part of
	// allocation should be locked for the blobber rewards even if
	// user never write something to the blobber.
	MinLockDemand float64 `json:"min_lock_demand"`
	// MaxOfferDuration with this prices and the demand.
	MaxOfferDuration time.Duration `json:"max_offer_duration"`
}

func (edb EventDb) GetAllocation(id string) (*Allocation, error) {
	var alloc Allocation
	err := edb.Store.Get().Model(&Allocation{}).Where("allocation_id = ?", id).First(&alloc).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving allocation: %v, error: %v", id, err)
	}

	return &alloc, nil
}

func (edb EventDb) GetClientsAllocation(clientID string, limit common.Pagination) ([]Allocation, error) {
	allocs := make([]Allocation, 0)

	query := edb.Store.Get().Model(&Allocation{}).Where("owner = ?", clientID).Limit(limit.Limit).Offset(limit.Offset).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "start_time"},
			Desc:   limit.IsDescending,
		})

	result := query.Scan(&allocs)
	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving allocation for client: %v, error: %v", clientID, result.Error)
	}

	return allocs, nil
}

func (edb EventDb) GetActiveAllocationsCount() (int64, error) {
	var count int64
	result := edb.Store.Get().Model(&Allocation{}).Where("finalized = ? AND cancelled = ?", false, false).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("error retrieving active allocations , error: %v", result.Error)
	}

	return count, nil
}

func (edb EventDb) GetActiveAllocsBlobberCount() (int64, error) {
	var count int64
	err := edb.Store.Get().
		Raw("SELECT SUM(parity_shards) + SUM(data_shards) FROM allocations WHERE finalized = ? AND cancelled = ?",
			false, false).
		Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("error retrieving blobber allocations count, error: %v", err)
	}

	return count, nil
}

func (edb *EventDb) updateAllocation(updates *dbs.DbUpdates) error {
	return edb.Store.Get().
		Model(&Allocation{}).
		Where(&Allocation{AllocationID: updates.Id}).
		Updates(updates.Updates).Error
}

func (edb *EventDb) addAllocation(alloc *Allocation) error {
	return edb.Store.Get().Create(&alloc).Error
}
