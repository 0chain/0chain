package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"

	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/threshold/bls"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

const confMaxChallengeCompletionTime = "smart_contracts.storagesc.max_challenge_completion_time"

//msgp:ignore StorageAllocation AllocationChallenges
//go:generate msgp -io=false -tests=false -unexported -v

var (
	ALL_VALIDATORS_KEY               = ADDRESS + encryption.Hash("all_validators")
	ALL_CHALLENGE_READY_BLOBBERS_KEY = ADDRESS + encryption.Hash("all_challenge_ready_blobbers")
	BLOBBER_REWARD_KEY               = ADDRESS + encryption.Hash("blobber_rewards")
)

func getBlobberAllocationsKey(blobberID string) string {
	return ADDRESS + encryption.Hash("blobber_allocations_"+blobberID)
}

type Allocations struct {
	List SortedList
}

func (a *Allocations) has(id string) (ok bool) {
	_, ok = a.List.getIndex(id)
	return // false
}

func (an *Allocations) Encode() []byte {
	buff, _ := json.Marshal(an)
	return buff
}

func (an *Allocations) Decode(input []byte) error {
	err := json.Unmarshal(input, an)
	if err != nil {
		return err
	}
	return nil
}

func (an *Allocations) GetHash() string {
	return util.ToHex(an.GetHashBytes())
}

func (an *Allocations) GetHashBytes() []byte {
	return encryption.RawHash(an.Encode())
}

type ChallengeResponse struct {
	ID                string              `json:"challenge_id"`
	ValidationTickets []*ValidationTicket `json:"validation_tickets"`
}

type AllocOpenChallenge struct {
	ID        string           `json:"id"`
	CreatedAt common.Timestamp `json:"created_at"`
	BlobberID string           `json:"blobber_id"` // blobber id
}

type AllocationChallenges struct {
	AllocationID   string                `json:"allocation_id"`
	OpenChallenges []*AllocOpenChallenge `json:"open_challenges"`
	//OpenChallenges               []*StorageChallenge          `json:"challenges"`
	ChallengeMap             map[string]*AllocOpenChallenge `json:"-" msg:"-"`
	LatestCompletedChallenge *StorageChallenge              `json:"latest_completed_challenge"`
}

func (acs *AllocationChallenges) GetKey(globalKey string) datastore.Key {
	return globalKey + ":allocation_challenges:" + acs.AllocationID
}

func (acs *AllocationChallenges) MarshalMsg(b []byte) ([]byte, error) {
	d := allocationChallengesDecoder(*acs)
	return d.MarshalMsg(b)
}

func (acs *AllocationChallenges) UnmarshalMsg(b []byte) ([]byte, error) {
	d := &allocationChallengesDecoder{}
	v, err := d.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	*acs = AllocationChallenges(*d)
	acs.ChallengeMap = make(map[string]*AllocOpenChallenge)
	for _, challenge := range acs.OpenChallenges {
		acs.ChallengeMap[challenge.ID] = challenge
	}

	return v, nil
}

func (acs *AllocationChallenges) addChallenge(challenge *StorageChallenge) bool {
	if acs.ChallengeMap == nil {
		acs.ChallengeMap = make(map[string]*AllocOpenChallenge)
	}

	if _, ok := acs.ChallengeMap[challenge.ID]; !ok {
		oc := &AllocOpenChallenge{
			ID:        challenge.ID,
			BlobberID: challenge.BlobberID,
			CreatedAt: challenge.Created,
		}
		acs.OpenChallenges = append(acs.OpenChallenges, oc)
		acs.ChallengeMap[challenge.ID] = oc
		return true
	}

	return false
}

// Save saves the AllocationChallenges to MPT state
func (acs *AllocationChallenges) Save(state cstate.StateContextI, scAddress string) error {
	_, err := state.InsertTrieNode(acs.GetKey(scAddress), acs)
	return err
}

func (acs *AllocationChallenges) removeChallenge(challenge *StorageChallenge) bool {
	if _, ok := acs.ChallengeMap[challenge.ID]; !ok {
		return false
	}

	delete(acs.ChallengeMap, challenge.ID)
	for i := range acs.OpenChallenges {
		if acs.OpenChallenges[i].ID == challenge.ID {
			acs.OpenChallenges = append(
				acs.OpenChallenges[:i], acs.OpenChallenges[i+1:]...)

			acs.LatestCompletedChallenge = challenge
			return true
		}
	}

	return true
}

type allocationChallengesDecoder AllocationChallenges

// swagger:model StorageChallenge
type StorageChallenge struct {
	Created         common.Timestamp    `json:"created"`
	ID              string              `json:"id"`
	TotalValidators int                 `json:"total_validators"`
	ValidatorIDs    []string            `json:"validator_ids"`
	ValidatorIDMap  map[string]struct{} `json:"-" msg:"-"`
	AllocationID    string              `json:"allocation_id"`
	BlobberID       string              `json:"blobber_id"`
	Responded       bool                `json:"responded"`
}

func (sc *StorageChallenge) GetKey(globalKey string) datastore.Key {
	return storageChallengeKey(globalKey, sc.ID)
}

func storageChallengeKey(globalKey, challengeID string) datastore.Key {
	return globalKey + "storage_challenge:" + challengeID
}

// Save saves the storage challenge to MPT state
func (sc *StorageChallenge) Save(state cstate.StateContextI, scAddress string) error {
	_, err := state.InsertTrieNode(sc.GetKey(scAddress), sc)
	return err
}

type ValidationNode struct {
	ID                string             `json:"id"`
	BaseURL           string             `json:"url"`
	PublicKey         string             `json:"-" msg:"-"`
	StakePoolSettings stakepool.Settings `json:"stake_pool_settings"`
}

// validate the validator configurations
func (sn *ValidationNode) validate(conf *Config) (err error) {
	if strings.Contains(sn.BaseURL, "localhost") &&
		node.Self.Host != "localhost" {
		return errors.New("invalid validator base url")
	}

	return
}

func (sn *ValidationNode) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + "validator:" + sn.ID)
}

func (sn *ValidationNode) GetUrlKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + "validator:" + sn.BaseURL)
}

func (sn *ValidationNode) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *ValidationNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func (sn *ValidationNode) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *ValidationNode) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
}

type ValidatorNodes struct {
	Nodes []*ValidationNode
}

// Terms represents Blobber terms. A Blobber can update its terms,
// but any existing offer will use terms of offer signing time.
type Terms struct {
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

// The minLockDemand returns min lock demand value for this Terms (the
// WritePrice and the MinLockDemand must be already set). Given size in GB and
// rest of allocation duration in time units are used.
func (t *Terms) minLockDemand(gbSize, rdtu float64) (mdl currency.Coin) {

	var mldf = float64(t.WritePrice) * gbSize * t.MinLockDemand //
	return currency.Coin(mldf * rdtu)                           //
}

// validate a received terms
func (t *Terms) validate(conf *Config) (err error) {
	if t.MinLockDemand < 0.0 || t.MinLockDemand > 1.0 {
		return errors.New("invalid min_lock_demand")
	}
	if t.MaxOfferDuration < conf.MinOfferDuration {
		return errors.New("insufficient max_offer_duration")
	}

	if t.ReadPrice > conf.MaxReadPrice {
		return errors.New("read_price is greater than max_read_price allowed")
	}
	if t.WritePrice < conf.MinWritePrice {
		return errors.New("write_price is greater than max_write_price allowed")
	}
	if t.WritePrice > conf.MaxWritePrice {
		return errors.New("write_price is greater than max_write_price allowed")
	}

	return // nil
}

const (
	MaxLatitude  = 90
	MinLatitude  = -90
	MaxLongitude = 180
	MinLongitude = -180
)

// Move to the core, in case of multi-entity use of geo data
type StorageNodeGeolocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	// reserved / Accuracy float64 `mapstructure:"accuracy"`
}

func (sng StorageNodeGeolocation) validate() error {
	if sng.Latitude < MinLatitude || MaxLatitude < sng.Latitude {
		return common.NewErrorf("out_of_range_geolocation",
			"latitude %f should be in range [-90, 90]", sng.Latitude)
	}
	if sng.Longitude < MinLongitude || MaxLongitude < sng.Longitude {
		return common.NewErrorf("out_of_range_geolocation",
			"latitude %f should be in range [-180, 180]", sng.Longitude)
	}
	return nil
}

type RewardPartitionLocation struct {
	Index      int              `json:"index"`
	StartRound int64            `json:"start_round"`
	Timestamp  common.Timestamp `json:"timestamp"`
}

// Info represents general information about blobber node
type Info struct {
	Name        string `json:"name"`
	WebsiteUrl  string `json:"website_url"`
	LogoUrl     string `json:"logo_url"`
	Description string `json:"description"`
}

// StorageNode represents Blobber configurations.
type StorageNode struct {
	ID                      string                 `json:"id"`
	BaseURL                 string                 `json:"url"`
	Geolocation             StorageNodeGeolocation `json:"geolocation"`
	Terms                   Terms                  `json:"terms"`     // terms
	Capacity                int64                  `json:"capacity"`  // total blobber capacity
	Allocated               int64                  `json:"allocated"` // allocated capacity
	DataRead                float64                `json:"data_read"` // in GB
	LastHealthCheck         common.Timestamp       `json:"last_health_check"`
	PublicKey               string                 `json:"-"`
	SavedData               int64                  `json:"saved_data"`
	DataReadLastRewardRound float64                `json:"data_read_last_reward_round"` // in GB
	LastRewardDataReadRound int64                  `json:"last_reward_data_read_round"` // last round when data read was updated
	// StakePoolSettings used initially to create and setup stake pool.
	StakePoolSettings stakepool.Settings      `json:"stake_pool_settings"`
	RewardPartition   RewardPartitionLocation `json:"reward_partition"`
	Information       Info                    `json:"info"`
}

// validate the blobber configurations
func (sn *StorageNode) validate(conf *Config) (err error) {
	if err = sn.Terms.validate(conf); err != nil {
		return
	}
	if sn.Capacity <= conf.MinBlobberCapacity {
		return errors.New("insufficient blobber capacity")
	}

	if strings.Contains(sn.BaseURL, "localhost") &&
		node.Self.Host != "localhost" {
		return errors.New("invalid blobber base url")
	}

	if err := sn.Geolocation.validate(); err != nil {
		return err
	}

	return
}

func (sn *StorageNode) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + sn.ID)
}

func (sn *StorageNode) GetUrlKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + sn.BaseURL)
}

func (sn *StorageNode) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *StorageNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

type StorageNodes struct {
	Nodes SortedBlobbers
}

func (sn *StorageNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func (sn *StorageNodes) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *StorageNodes) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *StorageNodes) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
}

type StorageAllocationStats struct {
	UsedSize                  int64  `json:"used_size"`
	NumWrites                 int64  `json:"num_of_writes"`
	NumReads                  int64  `json:"num_of_reads"`
	TotalChallenges           int64  `json:"total_challenges"`
	OpenChallenges            int64  `json:"num_open_challenges"`
	SuccessChallenges         int64  `json:"num_success_challenges"`
	FailedChallenges          int64  `json:"num_failed_challenges"`
	LastestClosedChallengeTxn string `json:"latest_closed_challenge"`
}

type BlobberAllocation struct {
	BlobberID    string `json:"blobber_id"`
	AllocationID string `json:"allocation_id"`
	// Size is blobber allocation maximum size
	Size            int64                   `json:"size"`
	AllocationRoot  string                  `json:"allocation_root"`
	LastWriteMarker *WriteMarker            `json:"write_marker"`
	Stats           *StorageAllocationStats `json:"stats"`
	// Terms of the BlobberAllocation represents weighted average terms
	// for the allocation. The MinLockDemand can be increased only,
	// to prevent some attacks. If a user extends an allocation then
	// we calculate new weighted average terms based on previous terms,
	// size and expiration and new terms size and expiration.
	Terms Terms `json:"terms"`
	// MinLockDemand for the allocation in tokens.
	MinLockDemand currency.Coin `json:"min_lock_demand"`
	// Spent is number of tokens sent from write pool to challenge pool
	// for this blobber. It's used to calculate min lock demand left
	// for this blobber. For a case, where a client uses > 1 parity shards
	// and don't sends a data to one of blobbers, the blobber should
	// receive its min_lock_demand tokens. Thus, we can't use shared
	// (for allocation) min_lock_demand and spent.
	Spent currency.Coin `json:"spent"`
	// Penalty o the blobber for the allocation in tokens.
	Penalty currency.Coin `json:"penalty"`
	// ReadReward of the blobber.
	ReadReward currency.Coin `json:"read_reward"`
	// Returned back to write pool on challenge failed.
	Returned currency.Coin `json:"returned"`
	// ChallengeReward of the blobber.
	ChallengeReward currency.Coin `json:"challenge_reward"`
	// FinalReward is number of tokens moved to the blobber on finalization.
	// It can be greater than zero, if user didn't spent the min lock demand
	// during the allocation.
	FinalReward currency.Coin `json:"final_reward"`

	// ChallengePoolIntegralValue represents integral price * size * dt for this
	// blobber. Since, a user can upload and delete file, and a challenge
	// request can be invoked at any time, then we have to use integral blobber
	// value.
	//
	// For example, if user uploads a file 100 GB for 100 time_units (until
	// allocation ends). Challenge pool value increased by
	//
	//     challenge_pool_value += 100 GB * 100 time_units * blobber_write_price
	//
	// Then a challenge (a challenge is for entire allocation of the blobber)
	// will affect
	//
	//     100 GB * ((chall_time - prev_chall_time) / time_unit)
	//
	// For example, for 1 time_unit a challenge moves to blobber
	//
	//     100 GB * 1 time_unit * blobber_write_price
	//
	// Then, after the challenge if user waits 1 time_unit and deletes the file,
	// the challenge pool will contain
	//
	//     100 GB * 1 time_unit * blobber_write_price
	//
	// And after one more time unit next challenge (after 2 time_units) will
	// want
	//
	//     100 GB * 2 time_unit * blobber_write_price
	//
	// But the challenge pool have only the same for 1 time_unit (file has
	// deleted a time_unit ago).
	//
	// Thus, we have to use this integral size that is affected by
	//
	//     - challenges
	//     - uploads
	//     - deletions
	//
	// A challenge reduces the integral size. An upload increases it. A deletion
	// reduces it too as a challenge.
	//
	// The integral value is price*size*dt. E.g. its formulas for every of the
	// operations:
	//
	//     1. Upload
	//
	//         integral_value += file_size * rest_dtu * blobber_write_price
	//
	//     2. Delete
	//
	//         integral_value -= file_size * rest_dtu * blobber_write_price
	//
	//     3. Challenge (successful or failed)
	//
	//         integral_value -= (chall_dtu / rest_dtu) * integral_value
	//
	// So, the integral value needed to calculate challenges values properly.
	//
	// Also, the integral value is challenge pool for this blobber-allocation.
	// Since, a challenge pool of an allocation contains tokens for all related
	// blobbers, then we should track value of every blobber to calculate
	// rewards and penalties properly.
	//
	// For any case, total value of all ChallengePoolIntegralValue of all
	// blobber of an allocation should be equal to related challenge pool
	// balance.
	ChallengePoolIntegralValue currency.Coin `json:"challenge_pool_integral_value"`
	// BlobberAllocationsPartitionLoc indicates the partition location for the allocation that
	// saved in blobber allocations partitions.
	BlobberAllocationsPartitionLoc *partitions.PartitionLocation `json:"blobber_allocs_partition_loc"`
}

// blobberPartitionsLocations that any blobber related partition locations could be
// stored in this node
type blobberPartitionsLocations struct {
	ID                         string // blobber id
	ChallengeReadyPartitionLoc *partitions.PartitionLocation
}

func (bpl *blobberPartitionsLocations) getKey(scAddress string) datastore.Key {
	return fmt.Sprintf("%s:blobber_partitions_locations:%s", scAddress, bpl.ID)
}

func (bpl *blobberPartitionsLocations) save(state cstate.StateContextI, scAddress string) error {
	_, err := state.InsertTrieNode(bpl.getKey(scAddress), bpl)
	return err
}

func (bpl *blobberPartitionsLocations) load(state cstate.StateContextI, scAddress string) error {
	return state.GetTrieNode(bpl.getKey(scAddress), bpl)
}

func newBlobberAllocation(
	size int64,
	allocation *StorageAllocation,
	blobber *StorageNode,
	date common.Timestamp,
) *BlobberAllocation {
	ba := &BlobberAllocation{}
	ba.Stats = &StorageAllocationStats{}
	ba.Size = size
	ba.Terms = blobber.Terms
	ba.AllocationID = allocation.ID
	ba.BlobberID = blobber.ID
	ba.MinLockDemand = blobber.Terms.minLockDemand(
		sizeInGB(size), allocation.restDurationInTimeUnits(date),
	)
	return ba
}

// The upload used after commitBlobberConnection (size > 0) to calculate
// internal integral value.
func (d *BlobberAllocation) upload(size int64, now common.Timestamp,
	rdtu float64) (move currency.Coin) {

	move = currency.Coin(sizeInGB(size) * float64(d.Terms.WritePrice) * rdtu)
	d.ChallengePoolIntegralValue += move
	return
}

func (d *BlobberAllocation) Offer() currency.Coin {
	return currency.Coin(sizeInGB(d.Size) * float64(d.Terms.WritePrice))
}

// The upload used after commitBlobberConnection (size < 0) to calculate
// internal integral value. The size argument expected to be positive (not
// negative).
func (d *BlobberAllocation) delete(size int64, now common.Timestamp,
	rdtu float64) (move currency.Coin) {

	move = currency.Coin(sizeInGB(size) * float64(d.Terms.WritePrice) * rdtu)
	d.ChallengePoolIntegralValue -= move
	return
}

// The upload used after commitBlobberConnection (size < 0) to calculate
// internal integral value. It returns tokens should be moved for the blobber
// challenge (doesn't matter rewards or penalty). The RDTU should be based on
// previous challenge time. And the DTU should be based on previous - current
// challenge time.
func (d *BlobberAllocation) challenge(dtu, rdtu float64) (move currency.Coin) {
	move = currency.Coin((dtu / rdtu) * float64(d.ChallengePoolIntegralValue))
	d.ChallengePoolIntegralValue -= move
	return
}

// PriceRange represents a price range allowed by user to filter blobbers.
type PriceRange struct {
	Min currency.Coin `json:"min"`
	Max currency.Coin `json:"max"`
}

// isValid price range.
func (pr *PriceRange) isValid() bool {
	return pr.Min <= pr.Max
}

// isMatch given price
func (pr *PriceRange) isMatch(price currency.Coin) bool {
	return pr.Min <= price && price <= pr.Max
}

// StorageAllocation request and entity.
// swagger:model StorageAllocation
type StorageAllocation struct {
	// ID is unique allocation ID that is equal to hash of transaction with
	// which the allocation has created.
	ID string `json:"id"`
	// Tx keeps hash with which the allocation has created or updated.
	Tx string `json:"tx"`

	DataShards        int                     `json:"data_shards"`
	ParityShards      int                     `json:"parity_shards"`
	Size              int64                   `json:"size"`
	Expiration        common.Timestamp        `json:"expiration_date"`
	Owner             string                  `json:"owner_id"`
	OwnerPublicKey    string                  `json:"owner_public_key"`
	Stats             *StorageAllocationStats `json:"stats"`
	DiverseBlobbers   bool                    `json:"diverse_blobbers"`
	PreferredBlobbers []string                `json:"preferred_blobbers"`
	// Blobbers not to be used anywhere except /allocation and /allocations table
	// if Blobbers are getting used in any smart-contract, we should avoid.
	BlobberAllocs    []*BlobberAllocation          `json:"blobber_details"`
	BlobberAllocsMap map[string]*BlobberAllocation `json:"-" msg:"-"`

	// Defines mutability of the files in the allocation, used by blobber on CommitWrite
	IsImmutable bool `json:"is_immutable"`

	// Flag to determine if anyone can extend this allocation
	ThirdPartyExtendable bool `json:"third_party_extendable"`

	// FileOptions to define file restrictions on an allocation for third-parties
	// default 00000000 for all crud operations suggesting only owner has the below listed abilities.
	// enabling option/s allows any third party to perform certain ops
	// 00000001 - 1  - upload
	// 00000010 - 2  - delete
	// 00000100 - 4  - update
	// 00001000 - 8  - move
	// 00010000 - 16 - copy
	// 00100000 - 32 - rename
	FileOptions uint8 `json:"file_options"`

	WritePool currency.Coin `json:"write_pool"`

	// Requested ranges.
	ReadPriceRange  PriceRange `json:"read_price_range"`
	WritePriceRange PriceRange `json:"write_price_range"`

	// ChallengeCompletionTime is max challenge completion time of
	// all blobbers of the allocation.
	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
	// StartTime is time when the allocation has been created. We will
	// use it to check blobber's MaxOfferTime extending the allocation.
	StartTime common.Timestamp `json:"start_time"`
	// Finalized is true where allocation has been finalized.
	Finalized bool `json:"finalized,omitempty"`
	// Canceled set to true where allocation finalized by cancel_allocation
	// transaction.
	Canceled bool `json:"canceled,omitempty"`
	// UsedSize used to calculate blobber reward ratio.
	UsedSize int64 `json:"-" msg:"-"`

	// MovedToChallenge is number of tokens moved to challenge pool.
	MovedToChallenge currency.Coin `json:"moved_to_challenge,omitempty"`
	// MovedBack is number of tokens moved from challenge pool to
	// related write pool (the Back) if a data has deleted.
	MovedBack currency.Coin `json:"moved_back,omitempty"`
	// MovedToValidators is total number of tokens moved to validators
	// of the allocation.
	MovedToValidators currency.Coin `json:"moved_to_validators,omitempty"`

	// TimeUnit configured in Storage SC when the allocation created. It can't
	// be changed for this allocation anymore. Even using expire allocation.
	TimeUnit time.Duration `json:"time_unit"`

	Curators []string `json:"curators"`
	// Name is the name of an allocation
	Name string `json:"name"`
}

func (sa *StorageAllocation) addToWritePool(
	txn *transaction.Transaction,
	mintNewTokens bool,
	balances cstate.StateContextI,
) error {
	if !mintNewTokens {
		if err := stakepool.CheckClientBalance(txn, balances); err != nil {
			return fmt.Errorf("client balance check failed: %v", err)
		}
	}

	if mintNewTokens {
		if err := balances.AddMint(&state.Mint{
			Minter:     ADDRESS,
			ToClientID: ADDRESS,
			Amount:     txn.Value,
		}); err != nil {
			return fmt.Errorf("minting tokens for write pool: %v", err)
		}
	} else {
		transfer := state.NewTransfer(txn.ClientID, txn.ToClientID, currency.Coin(txn.Value))
		if err := balances.AddTransfer(transfer); err != nil {
			return fmt.Errorf("adding transfer to allocation pool: %v", err)
		}
	}

	if writePool, err := currency.AddCoin(sa.WritePool, txn.Value); err != nil {
		return err
	} else {
		sa.WritePool = writePool
	}
	return nil
}

func (sa *StorageAllocation) moveToChallengePool(
	cp *challengePool,
	value currency.Coin,
) error {
	if cp == nil {
		return errors.New("invalid challenge pool")
	}
	if value > sa.WritePool {
		return fmt.Errorf("insufficent funds %v in write pool to pay %v", sa.WritePool, value)
	}

	if balance, err := currency.AddCoin(cp.Balance, value); err != nil {
		return err
	} else {
		cp.Balance = balance
	}
	if writePool, err := currency.MinusCoin(sa.WritePool, value); err != nil {
		return err
	} else {
		sa.WritePool = writePool
	}
	return nil
}

func (sa *StorageAllocation) moveFromChallengePool(
	cp *challengePool,
	value currency.Coin,
) error {
	if cp == nil {
		return errors.New("invalid challenge pool")
	}

	if cp.Balance < value {
		return fmt.Errorf("not enough tokens in challenge pool %s: %d < %d",
			cp.ID, cp.Balance, value)
	}

	if balance, err := currency.MinusCoin(cp.Balance, value); err != nil {
		return err
	} else {
		cp.Balance = balance
	}
	if writePool, err := currency.AddCoin(sa.WritePool, value); err != nil {
		return err
	} else {
		sa.WritePool = writePool
	}
	return nil
}

func (sa *StorageAllocation) validateAllocationBlobber(
	blobber *StorageNode,
	sp *stakePool,
	now common.Timestamp,
) error {
	bSize := sa.bSize()
	duration := common.ToTime(sa.Expiration).Sub(common.ToTime(now))

	// filter by max offer duration
	if blobber.Terms.MaxOfferDuration < duration {
		return fmt.Errorf("duration %v exceeds blobber %s maximum %v",
			duration, blobber.ID, blobber.Terms.MaxOfferDuration)
	}
	// filter by read price
	if !sa.ReadPriceRange.isMatch(blobber.Terms.ReadPrice) {
		return fmt.Errorf("read price range %v does not match blobber %s read price %v",
			sa.ReadPriceRange, blobber.ID, blobber.Terms.ReadPrice)
	}
	// filter by write price
	if !sa.WritePriceRange.isMatch(blobber.Terms.WritePrice) {
		return fmt.Errorf("read price range %v does not match blobber %s write price %v",
			sa.ReadPriceRange, blobber.ID, blobber.Terms.ReadPrice)
	}
	// filter by blobber's capacity left
	if blobber.Capacity-blobber.Allocated < bSize {
		return fmt.Errorf("blobber %s free capacity %v insufficent, wanted %v",
			blobber.ID, blobber.Capacity-blobber.Allocated, bSize)
	}

	if blobber.LastHealthCheck <= (now - blobberHealthTime) {
		return fmt.Errorf("blobber %s failed health check", blobber.ID)
	}

	if blobber.Terms.WritePrice > 0 && sp.unallocatedCapacity(blobber.Terms.WritePrice) < bSize {

		return fmt.Errorf("blobber %v staked capacity %v is insufficent, wanted %v",
			blobber.ID, sp.unallocatedCapacity(blobber.Terms.WritePrice), bSize)
	}

	return nil
}

func (sa *StorageAllocation) bSize() int64 {
	return int64(math.Ceil(float64(sa.Size) / float64(sa.DataShards)))
}

func (sa *StorageAllocation) removeBlobber(
	blobbers []*StorageNode,
	blobberID string,
	ssc *StorageSmartContract,
	balances cstate.StateContextI,
) ([]*StorageNode, error) {
	blobAlloc, found := sa.BlobberAllocsMap[blobberID]
	if !found {
		return nil, fmt.Errorf("cannot find blobber %s in allocation", blobberID)
	}
	delete(sa.BlobberAllocsMap, blobberID)

	var removedBlobber *StorageNode
	found = false
	for i, d := range blobbers {
		if d.ID == blobberID {
			removedBlobber = blobbers[i]
			blobbers[i] = blobbers[len(blobbers)-1]
			blobbers = blobbers[:len(blobbers)-1]
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("cannot find blobber %s in allocation", blobAlloc.BlobberID)
	}

	found = false
	for i, d := range sa.BlobberAllocs {
		if d.BlobberID == blobberID {
			sa.BlobberAllocs[i] = sa.BlobberAllocs[len(sa.BlobberAllocs)-1]
			sa.BlobberAllocs = sa.BlobberAllocs[:len(sa.BlobberAllocs)-1]

			if err := removeAllocationFromBlobber(ssc,
				d,
				d.AllocationID,
				balances); err != nil {
				return nil, err
			}

			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("cannot find blobber %s in allocation", blobAlloc.BlobberID)
	}

	if _, err := balances.InsertTrieNode(removedBlobber.GetKey(ADDRESS), removedBlobber); err != nil {
		return nil, fmt.Errorf("saving blobber %v, error: %v", removedBlobber.ID, err)
	}

	return blobbers, nil
}

func (sa *StorageAllocation) changeBlobbers(
	blobbers []*StorageNode,
	addId, removeId string,
	ssc *StorageSmartContract,
	now common.Timestamp,
	balances cstate.StateContextI,
) ([]*StorageNode, error) {
	var err error
	if len(removeId) > 0 {
		if blobbers, err = sa.removeBlobber(blobbers, removeId, ssc, balances); err != nil {
			return nil, err
		}
	} else {
		// If we are not removing a blobber, then the number of shards must increase.
		sa.ParityShards++
	}

	_, found := sa.BlobberAllocsMap[addId]
	if found {
		return nil, fmt.Errorf("allocation already has blobber %s", addId)
	}

	addedBlobber, err := ssc.getBlobber(addId, balances)
	if err != nil {
		return nil, err
	}
	addedBlobber.Allocated += sa.bSize()
	afterSize := sa.bSize()

	blobbers = append(blobbers, addedBlobber)
	ba := newBlobberAllocation(afterSize, sa, addedBlobber, now)
	sa.BlobberAllocsMap[addId] = ba
	sa.BlobberAllocs = append(sa.BlobberAllocs, ba)

	var sp *stakePool
	if sp, err = ssc.getStakePool(addedBlobber.ID, balances); err != nil {
		return nil, fmt.Errorf("can't get blobber's stake pool: %v", err)
	}
	if err := sa.validateAllocationBlobber(addedBlobber, sp, now); err != nil {
		return nil, err
	}

	return blobbers, nil
}

func (sa *StorageAllocation) save(state cstate.StateContextI, scAddress string) error {
	_, err := state.InsertTrieNode(sa.GetKey(scAddress), sa)
	return err
}

// removeAllocationFromBlobber removes the allocation from blobber
func removeAllocationFromBlobber(
	ssc *StorageSmartContract,
	blobAlloc *BlobberAllocation,
	allocID string,
	balances cstate.StateContextI) error {

	blobberID := blobAlloc.BlobberID
	if blobAlloc.BlobberAllocationsPartitionLoc == nil {
		logging.Logger.Error("skipping removing allocation from blobber partition" +
			"empty blobber allocation partition location")
		return nil
	}

	blobAllocsParts, err := partitionsBlobberAllocations(blobberID, balances)
	if err != nil {
		return fmt.Errorf("cannot fetch blobber allocation partition: %v", err)
	}

	if err := blobAllocsParts.RemoveItem(balances, blobAlloc.BlobberAllocationsPartitionLoc.Location, allocID); err != nil {
		return fmt.Errorf("could not remove allocation from blobber allocations partitions: %v", err)
	}

	if err := blobAllocsParts.Save(balances); err != nil {
		return fmt.Errorf("could not update blobber allocation partitions: %v", err)
	}
	// nullifying the blobber alloc challenge pasrtition location
	blobAlloc.BlobberAllocationsPartitionLoc = nil

	allocNum, err := blobAllocsParts.Size(balances)
	if err != nil {
		return fmt.Errorf("error getting size of challenge partition: %v", err)
	}

	if allocNum > 0 {
		return nil
	}

	// remove the blobber related resources fom MPT when there's no allocation attached to it
	blobPartsLocs := &blobberPartitionsLocations{ID: blobberID}
	if err := blobPartsLocs.load(balances, ssc.ID); err != nil {
		return fmt.Errorf("could not load blobber patitions locations: %v", err)
	}

	if blobPartsLocs.ChallengeReadyPartitionLoc == nil {
		return fmt.Errorf("challenge ready partition location is empty")
	}

	if err := partitionsChallengeReadyBlobbersRemove(balances,
		blobPartsLocs.ChallengeReadyPartitionLoc, blobberID); err != nil {
		return fmt.Errorf("could not remove blobber from challenge ready partitions: %v", err)
	}

	blobPartsLocs.ChallengeReadyPartitionLoc = nil
	if err := blobPartsLocs.save(balances, ssc.ID); err != nil {
		return fmt.Errorf("could not update blobber partitions locations node: %v", err)
	}

	return nil
}

type StorageAllocationDecode StorageAllocation

// The restMinLockDemand returns number of tokens required as min_lock_demand;
// if a blobber receive write marker, then some token moves to related
// challenge pool and 'Spent' of this blobber is increased; thus, the 'Spent'
// reduces the rest of min_lock_demand of this blobber; but, if a malfunctioning
// client doesn't send a data to a blobber (or blobbers) then this blobbers
// don't receive tokens, their spent will be zero, and the min lock demand
// will be blobber reward anyway.
func (sa *StorageAllocation) restMinLockDemand() (rest currency.Coin) {
	for _, details := range sa.BlobberAllocs {
		if details.MinLockDemand > details.Spent {
			rest += details.MinLockDemand - details.Spent
		}
	}
	return
}

func (sa *StorageAllocation) validate(now time.Time,
	conf *Config) (err error) {

	if !sa.ReadPriceRange.isValid() {
		return errors.New("invalid read_price range")
	}
	if !sa.WritePriceRange.isValid() {
		return errors.New("invalid write_price range")
	}
	if sa.Size < conf.MinAllocSize {
		return errors.New("insufficient allocation size")
	}
	dur := common.ToTime(sa.Expiration).Sub(now)
	if dur < conf.MinAllocDuration {
		return errors.New("insufficient allocation duration")
	}

	if sa.DataShards <= 0 {
		return errors.New("invalid number of data shards")
	}

	if sa.OwnerPublicKey == "" {
		return errors.New("missing owner public key")
	}

	if sa.Owner == "" {
		return errors.New("missing owner id")
	}

	return // nil
}

type filterBlobberFunc func(blobber *StorageNode) (kick bool)

func (sa *StorageAllocation) filterBlobbers(list []*StorageNode,
	creationDate common.Timestamp, bsize int64, filters ...filterBlobberFunc) (
	filtered []*StorageNode) {

	var (
		dur = common.ToTime(sa.Expiration).Sub(common.ToTime(creationDate))
		i   int
	)

List:
	for _, b := range list {
		// filter by max offer duration
		if b.Terms.MaxOfferDuration < dur {
			continue
		}
		// filter by read price
		if !sa.ReadPriceRange.isMatch(b.Terms.ReadPrice) {
			continue
		}
		// filter by write price
		if !sa.WritePriceRange.isMatch(b.Terms.WritePrice) {
			continue
		}
		// filter by blobber's capacity left
		if b.Capacity-b.Allocated < bsize {
			continue
		}

		for _, filter := range filters {
			if filter(b) {
				continue List
			}
		}
		list[i] = b
		i++
	}

	return list[:i]
}

// validateEachBlobber (this is a copy paste version of filterBlobbers with minute modification for verifications)
func (sa *StorageAllocation) validateEachBlobber(ssc *StorageSmartContract, blobbers []*blobberWithPool,
	creationDate common.Timestamp, balances cstate.StateContextI) (
	[]*blobberWithPool, []string) {

	var (
		errors   = make([]string, 0, len(blobbers))
		filtered = make([]*blobberWithPool, 0, len(blobbers))
	)
	for _, b := range blobbers {
		err := sa.validateAllocationBlobber(b.StorageNode, b.Pool, creationDate)
		if err != nil {
			logging.Logger.Debug("error validating blobber", zap.String("id", b.ID), zap.Error(err))
			errors = append(errors, err.Error())
			continue
		}
		filtered = append(filtered, b)
	}
	return filtered, errors
}

// Until returns allocation expiration.
func (sa *StorageAllocation) Until() common.Timestamp {
	return sa.Expiration + toSeconds(sa.ChallengeCompletionTime)
}

// The durationInTimeUnits returns given duration (represented as
// common.Timestamp) as duration in time units (float point value) for
// this allocation (time units for the moment of the allocation creation).
func (sa *StorageAllocation) durationInTimeUnits(dur common.Timestamp) (
	dtu float64) {

	dtu = float64(dur.Duration()) / float64(sa.TimeUnit)
	return
}

// The restDurationInTimeUnits return rest duration of the allocation in time
// units as a float64 value.
func (sa *StorageAllocation) restDurationInTimeUnits(now common.Timestamp) (
	rdtu float64) {

	rdtu = sa.durationInTimeUnits(sa.Expiration - now)
	return
}

// For a stored files (size). Changing an allocation duration and terms
// (weighted average). We need to move more tokens to related challenge pool.
// Or move some tokens from the challenge pool back.
//
// For example, we have allocation for 1 time unit (let it be mouth), with
// 1 GB of stored files. For the 1GB related challenge pool originally filled
// up with
//
//     (integral): write_price * size * duration
//     e.g.: (integral) write_price * 1 GB * 1 month
//
// After some time (a half or the month, for example) some tokens from the
// challenge pool moved back to write_pool. Some tokens moved to blobbers. And
// the challenge pool contains the rest (rest_challenge_pool).
//
// Then, we are extending the allocation to:
//
// 1) 2 months, write_price changed
// 2) 0.7 month, write_price changed
//
// For (1) case, we should move more tokens to the challenge pool. The
// difference is
//
//     a = old_write_price * size * old_duration_remaining (old expiration)
//     b = new_write_price * size * new_duration_remaining (new expiration)
//
//  And the difference is
//
//     b - a (move to challenge pool, or move back from challenge pool)
//
// This movement should be performed during allocation extension or reduction.
// So, if positive, then we should add more tokens to related challenge pool.
// Otherwise, move some tokens back to write pool.
//
// In result, the changes is ordered as BlobberAllocs field is ordered.
//
// For a case of allocation reducing, where no expiration, nor size changed
// we are using the same terms. And for this method, the oterms argument is
// nil for this case (meaning, terms hasn't changed).
func (sa *StorageAllocation) challengePoolChanges(odr, ndr common.Timestamp,
	oterms []Terms) (values []currency.Coin) {

	// odr -- old duration remaining
	// ndr -- new duration remaining

	// in time units, instead of common.Timestamp
	var (
		odrtu = sa.durationInTimeUnits(odr)
		ndrtu = sa.durationInTimeUnits(ndr)
	)

	values = make([]currency.Coin, 0, len(sa.BlobberAllocs))

	for i, d := range sa.BlobberAllocs {
		if d.Stats == nil || d.Stats.UsedSize == 0 {
			values = append(values, 0) // no data, no changes
			continue
		}
		var (
			size = sizeInGB(d.Stats.UsedSize)  // in GB
			nwp  = float64(d.Terms.WritePrice) // new write price
			owp  float64                       // original write price

			a, b, diff float64 // original value, new value, value difference
		)

		if oterms != nil {
			owp = float64(oterms[i].WritePrice) // original write price
		} else {
			owp = float64(d.Terms.WritePrice) // terms weren't changed
		}

		a = owp * size * odrtu // original value (by original terms)
		b = nwp * size * ndrtu // new value (by new terms)

		diff = b - a // value difference

		values = append(values, currency.Coin(diff))
	}

	return
}

func (sa *StorageAllocation) IsValidFinalizer(id string) bool {
	if sa.Owner == id {
		return true // finalizing by owner
	}
	for _, d := range sa.BlobberAllocs {
		if d.BlobberID == id {
			return true // one of blobbers
		}
	}
	return false // unknown
}

func (sn *StorageAllocation) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + sn.ID)
}

func (sn *StorageAllocation) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	sn.BlobberAllocsMap = make(map[string]*BlobberAllocation)
	for _, blobberAllocation := range sn.BlobberAllocs {
		if blobberAllocation.Stats != nil {
			sn.UsedSize += blobberAllocation.Stats.UsedSize // total used
		}
		sn.BlobberAllocsMap[blobberAllocation.BlobberID] = blobberAllocation
	}
	return nil
}

func (sn *StorageAllocation) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *StorageAllocation) MarshalMsg(o []byte) ([]byte, error) {
	d := StorageAllocationDecode(*sn)
	return d.MarshalMsg(o)
}

func (sn *StorageAllocation) UnmarshalMsg(data []byte) ([]byte, error) {
	d := &StorageAllocationDecode{}
	o, err := d.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}

	*sn = StorageAllocation(*d)

	sn.BlobberAllocsMap = make(map[string]*BlobberAllocation)
	for _, blobberAllocation := range sn.BlobberAllocs {
		if blobberAllocation.Stats != nil {
			sn.UsedSize += blobberAllocation.Stats.UsedSize // total used
		}
		sn.BlobberAllocsMap[blobberAllocation.BlobberID] = blobberAllocation
	}
	return o, nil
}

func getMaxChallengeCompletionTime() time.Duration {
	return config.SmartContractConfig.GetDuration(confMaxChallengeCompletionTime)
}

// removeExpiredChallenges removes all expired challenges from the allocation,
// return the expired challenge ids, or error if any.
// the expired challenge ids could be used to delete the challenge node from MPT when needed
func (sa *StorageAllocation) removeExpiredChallenges(allocChallenges *AllocationChallenges,
	now common.Timestamp) ([]string, error) {
	var (
		expiredChallengeIDs = make([]string, 0, len(allocChallenges.OpenChallenges))
	)

	cct := getMaxChallengeCompletionTime()
	for _, oc := range allocChallenges.OpenChallenges {
		if !isChallengeExpired(now, oc.CreatedAt, cct) {
			// not expired, following open challenges would not expire too, so break here
			break
		}

		// expired
		expiredChallengeIDs = append(expiredChallengeIDs, oc.ID)

		ba, ok := sa.BlobberAllocsMap[oc.BlobberID]
		if ok {
			ba.Stats.FailedChallenges++
			ba.Stats.OpenChallenges--
			sa.Stats.FailedChallenges++
			sa.Stats.OpenChallenges--
		}
	}

	allocChallenges.OpenChallenges = allocChallenges.OpenChallenges[len(expiredChallengeIDs):]

	return expiredChallengeIDs, nil
}

type BlobberCloseConnection struct {
	AllocationRoot     string       `json:"allocation_root"`
	PrevAllocationRoot string       `json:"prev_allocation_root"`
	WriteMarker        *WriteMarker `json:"write_marker"`
}

func (bc *BlobberCloseConnection) Decode(input []byte) error {
	err := json.Unmarshal(input, bc)
	if err != nil {
		return err
	}
	return nil
}

func (bc *BlobberCloseConnection) Verify() bool {
	if bc.WriteMarker == nil {
		return false
	}
	if len(bc.AllocationRoot) == 0 {
		return false
	}

	if bc.WriteMarker.AllocationRoot != bc.AllocationRoot {
		// return "", common.NewError("invalid_parameters",
		//     "Invalid Allocation root. Allocation root in write marker " +
		//     "does not match the commit")
		return false
	}

	if bc.WriteMarker.PreviousAllocationRoot != bc.PrevAllocationRoot {
		// return "", common.NewError("invalid_parameters",
		//     "Invalid Previous Allocation root. Previous Allocation root " +
		//     "in write marker does not match the commit")
		return false
	}
	return bc.WriteMarker.Verify()

}

type WriteMarker struct {
	AllocationRoot         string           `json:"allocation_root"`
	PreviousAllocationRoot string           `json:"prev_allocation_root"`
	AllocationID           string           `json:"allocation_id"`
	Size                   int64            `json:"size"`
	BlobberID              string           `json:"blobber_id"`
	Timestamp              common.Timestamp `json:"timestamp"`
	ClientID               string           `json:"client_id"`
	Signature              string           `json:"signature"`

	// file info
	LookupHash  string `json:"lookup_hash"`
	Name        string `json:"name"`
	ContentHash string `json:"content_hash"`
}

func (wm *WriteMarker) VerifySignature(
	clientPublicKey string,
	balances cstate.StateContextI,
) bool {
	hashData := wm.GetHashData()
	signatureHash := encryption.Hash(hashData)
	signatureScheme := balances.GetSignatureScheme()
	if err := signatureScheme.SetPublicKey(clientPublicKey); err != nil {
		return false
	}
	sigOK, err := signatureScheme.Verify(wm.Signature, signatureHash)
	if err != nil {
		return false
	}
	if !sigOK {
		return false
	}
	return true
}

func (wm *WriteMarker) GetHashData() string {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v", wm.AllocationRoot,
		wm.PreviousAllocationRoot, wm.AllocationID, wm.BlobberID, wm.ClientID,
		wm.Size, wm.Timestamp)
	return hashData
}

func (wm *WriteMarker) Verify() bool {
	if len(wm.AllocationID) == 0 || len(wm.AllocationRoot) == 0 ||
		len(wm.BlobberID) == 0 || len(wm.ClientID) == 0 || wm.Timestamp == 0 {
		return false
	}
	return true
}

type ReadConnection struct {
	ReadMarker *ReadMarker `json:"read_marker"`
}

func (rc *ReadConnection) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey +
		encryption.Hash(rc.ReadMarker.BlobberID+":"+rc.ReadMarker.ClientID))
}

func (rc *ReadConnection) Decode(input []byte) error {
	err := json.Unmarshal(input, rc)
	if err != nil {
		return err
	}
	return nil
}

func (rc *ReadConnection) Encode() []byte {
	buff, _ := json.Marshal(rc)
	return buff
}

func (rc *ReadConnection) GetHash() string {
	return util.ToHex(rc.GetHashBytes())
}

func (rc *ReadConnection) GetHashBytes() []byte {
	return encryption.RawHash(rc.Encode())
}

type ReadMarker struct {
	ClientID        string           `json:"client_id"`
	ClientPublicKey string           `json:"client_public_key"`
	BlobberID       string           `json:"blobber_id"`
	AllocationID    string           `json:"allocation_id"`
	OwnerID         string           `json:"owner_id"`
	Timestamp       common.Timestamp `json:"timestamp"`
	ReadCounter     int64            `json:"counter"`
	Signature       string           `json:"signature"`
	ReadSize        float64          `json:"read_size"`
}

func (rm *ReadMarker) VerifySignature(clientPublicKey string, balances cstate.StateContextI) bool {
	hashData := rm.GetHashData()
	signatureHash := encryption.Hash(hashData)
	signatureScheme := balances.GetSignatureScheme()
	if err := signatureScheme.SetPublicKey(clientPublicKey); err != nil {
		return false
	}
	sigOK, err := signatureScheme.Verify(rm.Signature, signatureHash)
	if err != nil {
		return false
	}
	if !sigOK {
		return false
	}
	return true
}

func (rm *ReadMarker) GetHashData() string {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v", rm.AllocationID,
		rm.BlobberID, rm.ClientID, rm.ClientPublicKey, rm.OwnerID,
		rm.ReadCounter, rm.Timestamp)
	return hashData
}

func (rm *ReadMarker) Verify(prevRM *ReadMarker, balances cstate.StateContextI) error {
	if rm.ReadCounter <= 0 || rm.BlobberID == "" || rm.ClientID == "" || rm.Timestamp == 0 {
		return common.NewError("invalid_read_marker", "length validations of fields failed")
	}

	if prevRM != nil {
		if rm.ClientID != prevRM.ClientID || rm.BlobberID != prevRM.BlobberID ||
			rm.Timestamp < prevRM.Timestamp ||
			rm.ReadCounter < prevRM.ReadCounter {

			return common.NewError("invalid_read_marker",
				"validations with previous marker failed.")
		}
	}

	if ok := rm.VerifySignature(rm.ClientPublicKey, balances); !ok {
		return common.NewError("invalid_read_marker", "Signature verification failed for the read marker")
	}

	return nil
}

func (rm *ReadMarker) VerifyClientID() error {
	pk := rm.ClientPublicKey

	pub := bls.PublicKey{}
	if err := pub.DeserializeHexStr(pk); err != nil {
		return err
	}

	if encryption.Hash(pub.Serialize()) != rm.ClientID {
		return common.NewError("invalid_read_marker", "Client ID verification failed")
	}

	return nil
}

type ValidationTicket struct {
	ChallengeID  string           `json:"challenge_id"`
	BlobberID    string           `json:"blobber_id"`
	ValidatorID  string           `json:"validator_id"`
	ValidatorKey string           `json:"validator_key"`
	Result       bool             `json:"success"`
	Message      string           `json:"message"`
	MessageCode  string           `json:"message_code"`
	Timestamp    common.Timestamp `json:"timestamp"`
	Signature    string           `json:"signature"`
}

func (vt *ValidationTicket) VerifySign(balances cstate.StateContextI) (bool, error) {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v", vt.ChallengeID, vt.BlobberID,
		vt.ValidatorID, vt.ValidatorKey, vt.Result, vt.Timestamp)
	hash := encryption.Hash(hashData)
	signatureScheme := balances.GetSignatureScheme()
	if err := signatureScheme.SetPublicKey(vt.ValidatorKey); err != nil {
		return false, err
	}
	verified, err := signatureScheme.Verify(vt.Signature, hash)
	return verified, err
}
