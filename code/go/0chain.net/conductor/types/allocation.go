package types

import "github.com/0chain/common/core/currency"

type Allocation struct {
	AllocationID             string        `json:"allocation_id"`
	TransactionID            string        `json:"transaction_id"`
	DataShards               int           `json:"data_shards"`
	ParityShards             int           `json:"parity_shards"`
	Size                     int64         `json:"size"`
	Expiration               int64         `json:"expiration"`
	Owner                    string        `json:"owner" gorm:"index:idx_aowner"`
	OwnerPublicKey           string        `json:"owner_public_key"`
	ReadPriceMin             currency.Coin `json:"read_price_min"`
	ReadPriceMax             currency.Coin `json:"read_price_max"`
	WritePriceMin            currency.Coin `json:"write_price_min"`
	WritePriceMax            currency.Coin `json:"write_price_max"`
	StartTime                int64         `json:"start_time"`
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
	ThirdPartyExtendable     bool          `json:"third_party_extendable"`
	FileOptions              uint16        `json:"file_options"`
	MinLockDemand            float64       `json:"min_lock_demand"`
}
