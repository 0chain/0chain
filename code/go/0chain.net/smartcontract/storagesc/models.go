package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"strings"
	"time"

	"0chain.net/smartcontract/partitions"

	"0chain.net/smartcontract/stakepool"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

//msgp:ignore StorageAllocation BlobberChallenge
//go:generate msgp -io=false -tests=false -v

var (
	ALL_BLOBBERS_KEY           = ADDRESS + encryption.Hash("all_blobbers")
	ALL_VALIDATORS_KEY         = ADDRESS + encryption.Hash("all_validators")
	ALL_BLOBBERS_CHALLENGE_KEY = ADDRESS + encryption.Hash("all_blobbers_challenge")
	BLOBBER_REWARD_KEY         = ADDRESS + encryption.Hash("blobber_rewards")
)

func getBlobberChallengeAllocationKey(blobberID string) string {
	return ADDRESS + encryption.Hash("blobber_challenge_allocation"+blobberID)
}

type ClientAllocation struct {
	ClientID    string       `json:"client_id"`
	Allocations *Allocations `json:"allocations"`
}

func (sn *ClientAllocation) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + sn.ClientID)
}

func (sn *ClientAllocation) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *ClientAllocation) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func (sn *ClientAllocation) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *ClientAllocation) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
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

type BlobberChallenge struct {
	BlobberID                string              `json:"blobber_id"`
	LatestCompletedChallenge *StorageChallenge   `json:"lastest_completed_challenge"`
	ChallengeIDs             []string            `json:"challenge_ids"`
	ChallengeIDMap           map[string]struct{} `json:"-" msg:"-"`
}

func (sn *BlobberChallenge) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + ":blobberchallenge:" + sn.BlobberID)
}

func (sn *BlobberChallenge) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *BlobberChallenge) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *BlobberChallenge) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
}

func (sn *BlobberChallenge) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	sn.ChallengeIDMap = make(map[string]struct{})
	for _, challengeID := range sn.ChallengeIDs {
		sn.ChallengeIDMap[challengeID] = struct{}{}
	}
	return nil
}

type BlobberChallengeDecode BlobberChallenge

func (sn *BlobberChallenge) MarshalMsg(o []byte) ([]byte, error) {
	d := BlobberChallengeDecode(*sn)
	return d.MarshalMsg(o)
}

func (sn *BlobberChallenge) UnmarshalMsg(data []byte) ([]byte, error) {
	d := &BlobberChallengeDecode{}
	o, err := d.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}

	*sn = BlobberChallenge(*d)

	sn.ChallengeIDMap = make(map[string]struct{})
	for _, challenge := range sn.ChallengeIDs {
		sn.ChallengeIDMap[challenge] = struct{}{}
	}
	return o, nil
}

func (sn *BlobberChallenge) addChallenge(challenge *StorageChallenge) bool {

	if sn.ChallengeIDs == nil {
		sn.ChallengeIDMap = make(map[string]struct{})
	}
	if _, ok := sn.ChallengeIDMap[challenge.ID]; !ok {
		sn.ChallengeIDs = append(sn.ChallengeIDs, challenge.ID)
		sn.ChallengeIDMap[challenge.ID] = struct{}{}
		return true
	}
	return false
}

type AllocationChallenge struct {
	AllocationID             string                       `json:"allocation_id"`
	Challenges               []*StorageChallenge          `json:"challenges"`
	ChallengeMap             map[string]*StorageChallenge `json:"-" msg:"-"`
	LatestCompletedChallenge *StorageChallenge            `json:"lastest_completed_challenge"`
}

func (sn *AllocationChallenge) GetKey(globalKey string) datastore.Key {
	return globalKey + ":allocationchallenge:" + sn.AllocationID
}

func (sn *AllocationChallenge) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *AllocationChallenge) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *AllocationChallenge) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
}

func (sn *AllocationChallenge) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	sn.ChallengeMap = make(map[string]*StorageChallenge)
	for _, challenge := range sn.Challenges {
		sn.ChallengeMap[challenge.ID] = challenge
	}
	return nil
}

func (sn *AllocationChallenge) addChallenge(challenge *StorageChallenge) bool {

	if sn.Challenges == nil {
		sn.Challenges = make([]*StorageChallenge, 0)
	}
	if sn.ChallengeMap == nil {
		sn.ChallengeMap = make(map[string]*StorageChallenge)
	}

	if _, ok := sn.ChallengeMap[challenge.ID]; !ok {
		sn.Challenges = append(sn.Challenges, challenge)
		sn.ChallengeMap[challenge.ID] = challenge
		return true
	}

	return false
}

type StorageChallenge struct {
	Created         common.Timestamp `json:"created"`
	ID              string           `json:"id"`
	TotalValidators int              `json:"total_validators"`
	AllocationID    string           `json:"allocation_id"`
	BlobberID       string           `json:"blobber_id"`
	Responded       bool             `json:"responded"`
}

func (sc *StorageChallenge) GetKey(globalKey string) datastore.Key {
	return globalKey + "storagechallenge:" + sc.ID
}

func (sc *StorageChallenge) Decode(input []byte) error {
	err := json.Unmarshal(input, sc)
	if err != nil {
		return err
	}
	return nil
}

func (sc *StorageChallenge) Encode() []byte {
	buff, _ := json.Marshal(sc)
	return buff
}

func (sc *StorageChallenge) GetHash() string {
	return util.ToHex(sc.GetHashBytes())
}

func (sc *StorageChallenge) GetHashBytes() []byte {
	return encryption.RawHash(sc.Encode())
}

type ValidationNode struct {
	ID                string                      `json:"id"`
	BaseURL           string                      `json:"url"`
	PublicKey         string                      `json:"-" msg:"-"`
	StakePoolSettings stakepool.StakePoolSettings `json:"stake_pool_settings"`
}

func (sn *ValidationNode) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + "validator:" + sn.ID)
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

func (sn *ValidatorNodes) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *ValidatorNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func (sn *ValidatorNodes) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *ValidatorNodes) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
}

// Terms represents Blobber terms. A Blobber can update its terms,
// but any existing offer will use terms of offer signing time.
type Terms struct {
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

// The minLockDemand returns min lock demand value for this Terms (the
// WritePrice and the MinLockDemand must be already set). Given size in GB and
// rest of allocation duration in time units are used.
func (t *Terms) minLockDemand(gbSize, rdtu float64) (mdl state.Balance) {

	var mldf = float64(t.WritePrice) * gbSize * t.MinLockDemand //
	return state.Balance(mldf * rdtu)                           //
}

// validate a received terms
func (t *Terms) validate(conf *Config) (err error) {
	if t.ReadPrice < 0 {
		return errors.New("negative read_price")
	}
	if t.WritePrice < 0 {
		return errors.New("negative write_price")
	}
	if t.MinLockDemand < 0.0 || t.MinLockDemand > 1.0 {
		return errors.New("invalid min_lock_demand")
	}
	if t.MaxOfferDuration < conf.MinOfferDuration {
		return errors.New("insufficient max_offer_duration")
	}
	if t.ChallengeCompletionTime < 0 {
		return errors.New("negative challenge_completion_time")
	}
	if t.ChallengeCompletionTime > conf.MaxChallengeCompletionTime {
		return errors.New("challenge_completion_time is greater than max " +
			"allowed by SC")
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
	Terms                   Terms                  `json:"terms"`         // terms
	Capacity                int64                  `json:"capacity"`      // total blobber capacity
	Used                    int64                  `json:"used"`          // allocated capacity
	BytesWritten            int64                  `json:"bytes_written"` // in bytes
	DataRead                float64                `json:"data_read"`     // in GB
	LastHealthCheck         common.Timestamp       `json:"last_health_check"`
	PublicKey               string                 `json:"-"`
	SavedData               int64                  `json:"saved_data"`
	DataReadLastRewardRound float64                `json:"data_read_last_reward_round"` // in GB
	LastRewardDataReadRound int64                  `json:"last_reward_data_read_round"` // last round when data read was updated
	// StakePoolSettings used initially to create and setup stake pool.
	StakePoolSettings stakepool.StakePoolSettings `json:"stake_pool_settings"`
	// ChallengeLocation to be replaced for BlobberChallengePartitionLocation once StorageNode is normalised
	//ChallengeLocation *partitions.PartitionLocation `json:"challenge_location"`
	RewardPartition RewardPartitionLocation `json:"reward_partition"`
	Information     Info                    `json:"info"`
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

// BlobberChallengePartitionLocation is a temporary object. should be removed once StorageNode is normalised
type BlobberChallengePartitionLocation struct {
	ID                string                        `json:"id"`
	PartitionLocation *partitions.PartitionLocation `json:"challenge_location"`
}

func (bcpl *BlobberChallengePartitionLocation) GetKey(globalKey string) datastore.Key {
	return globalKey + bcpl.ID + "blobber_challenge_partition"
}

func (bcpl *BlobberChallengePartitionLocation) Encode() []byte {
	buff, _ := json.Marshal(bcpl)
	return buff
}

func (bcpl *BlobberChallengePartitionLocation) Decode(input []byte) error {
	err := json.Unmarshal(input, bcpl)
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
	MinLockDemand state.Balance `json:"min_lock_demand"`
	// Spent is number of tokens sent from write pool to challenge pool
	// for this blobber. It's used to calculate min lock demand left
	// for this blobber. For a case, where a client uses > 1 parity shards
	// and don't sends a data to one of blobbers, the blobber should
	// receive its min_lock_demand tokens. Thus, we can't use shared
	// (for allocation) min_lock_demand and spent.
	Spent state.Balance `json:"spent"`
	// Penalty o the blobber for the allocation in tokens.
	Penalty state.Balance `json:"penalty"`
	// ReadReward of the blobber.
	ReadReward state.Balance `json:"read_reward"`
	// Returned back to write pool on challenge failed.
	Returned state.Balance `json:"returned"`
	// ChallengeReward of the blobber.
	ChallengeReward state.Balance `json:"challenge_reward"`
	// FinalReward is number of tokens moved to the blobber on finalization.
	// It can be greater than zero, if user didn't spent the min lock demand
	// during the allocation.
	FinalReward state.Balance `json:"final_reward"`

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
	ChallengePoolIntegralValue state.Balance `json:"challenge_pool_integral_value"`
	// ChallengePartitionLoc is the location of blobber partition(if exists) in BlobberChallengePartition
	ChallengePartitionLoc *partitions.PartitionLocation `json:"challenge_partition_loc"`
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
	rdtu float64) (move state.Balance) {

	move = state.Balance(sizeInGB(size) * float64(d.Terms.WritePrice) * rdtu)
	d.ChallengePoolIntegralValue += move
	return
}

func (d *BlobberAllocation) Offer() state.Balance {
	return state.Balance(sizeInGB(d.Size) * float64(d.Terms.WritePrice))
}

// The upload used after commitBlobberConnection (size < 0) to calculate
// internal integral value. The size argument expected to be positive (not
// negative).
func (d *BlobberAllocation) delete(size int64, now common.Timestamp,
	rdtu float64) (move state.Balance) {

	move = state.Balance(sizeInGB(size) * float64(d.Terms.WritePrice) * rdtu)
	d.ChallengePoolIntegralValue -= move
	return
}

// The upload used after commitBlobberConnection (size < 0) to calculate
// internal integral value. It returns tokens should be moved for the blobber
// challenge (doesn't matter rewards or penalty). The RDTU should be based on
// previous challenge time. And the DTU should be based on previous - current
// challenge time.
func (d *BlobberAllocation) challenge(dtu, rdtu float64) (move state.Balance) {
	move = state.Balance((dtu / rdtu) * float64(d.ChallengePoolIntegralValue))
	d.ChallengePoolIntegralValue -= move
	return
}

// PriceRange represents a price range allowed by user to filter blobbers.
type PriceRange struct {
	Min state.Balance `json:"min"`
	Max state.Balance `json:"max"`
}

// isValid price range.
func (pr *PriceRange) isValid() bool {
	return 0 <= pr.Min && pr.Min <= pr.Max
}

// isMatch given price
func (pr *PriceRange) isMatch(price state.Balance) bool {
	return pr.Min <= price && price <= pr.Max
}

// StorageAllocation request and entity.
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
	Blobbers       []*StorageNode                `json:"blobbers"`
	BlobberDetails []*BlobberAllocation          `json:"blobber_details"`
	BlobberMap     map[string]*BlobberAllocation `json:"-" msg:"-"`
	IsImmutable    bool                          `json:"is_immutable"`

	// Requested ranges.
	ReadPriceRange             PriceRange    `json:"read_price_range"`
	WritePriceRange            PriceRange    `json:"write_price_range"`
	MaxChallengeCompletionTime time.Duration `json:"max_challenge_completion_time"`

	//AllocationPools allocationPools `json:"allocation_pools"`
	WritePoolOwners []string `json:"write_pool_owners"`

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
	MovedToChallenge state.Balance `json:"moved_to_challenge,omitempty"`
	// MovedBack is number of tokens moved from challenge pool to
	// related write pool (the Back) if a data has deleted.
	MovedBack state.Balance `json:"moved_back,omitempty"`
	// MovedToValidators is total number of tokens moved to validators
	// of the allocation.
	MovedToValidators state.Balance `json:"moved_to_validators,omitempty"`

	// TimeUnit configured in Storage SC when the allocation created. It can't
	// be changed for this allocation anymore. Even using expire allocation.
	TimeUnit time.Duration `json:"time_unit"`

	Curators []string `json:"curators"`
	// Name is the name of an allocation
	Name string `json:"name"`
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
	if blobber.Capacity-blobber.Used < bSize {
		return fmt.Errorf("blobber %s free capacity %v insufficent, wanted %v",
			blobber.ID, blobber.Capacity-blobber.Used, bSize)
	}
	// filter by max challenge completion time
	if blobber.Terms.ChallengeCompletionTime > sa.MaxChallengeCompletionTime {
		return fmt.Errorf("blobber %s challenge compledtion time %v exceeds maximum challenge completeion time %v",
			blobber.ID, blobber.Terms.ChallengeCompletionTime, sa.MaxChallengeCompletionTime)
	}

	if blobber.LastHealthCheck <= (now - blobberHealthTime) {
		return fmt.Errorf("blobber %s failed health check", blobber.ID)
	}

	if blobber.Terms.WritePrice > 0 && sp.cleanCapacity(now, blobber.Terms.WritePrice) < bSize {
		return fmt.Errorf("blobber %v staked capacity %v is insufficent, wanted %v",
			blobber.ID, sp.cleanCapacity(now, blobber.Terms.WritePrice), bSize)
	}

	return nil
}

func (sa *StorageAllocation) bSize() int64 {
	var size = sa.DataShards + sa.ParityShards
	return (sa.Size + int64(size-1)) / int64(size)
}

func (sa *StorageAllocation) removeBlobber(
	blobbers []*StorageNode,
	removeId string,
	ssc *StorageSmartContract,
	balances chainstate.StateContextI,
) ([]*StorageNode, error) {
	remove, found := sa.BlobberMap[removeId]
	if !found {
		return nil, fmt.Errorf("cannot find blobber %s in allocation", remove.BlobberID)
	}
	delete(sa.BlobberMap, removeId)

	var removedBlobber *StorageNode
	found = false
	for i, d := range blobbers {
		if d.ID == removeId {
			removedBlobber = blobbers[i]
			blobbers[i] = blobbers[len(blobbers)-1]
			blobbers = blobbers[:len(blobbers)-1]
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("cannot find blobber %s in allocation", remove.BlobberID)
	}

	found = false
	for i, d := range sa.BlobberDetails {
		if d.BlobberID == removeId {
			sa.BlobberDetails[i] = sa.BlobberDetails[len(sa.BlobberDetails)-1]
			sa.BlobberDetails = sa.BlobberDetails[:len(sa.BlobberDetails)-1]
			removedBlobber.Used -= d.Size

			if d.ChallengePartitionLoc != nil {
				if err := removeBlobberAllocation(removeId, sa.ID,
					d.ChallengePartitionLoc.Location, ssc, balances); err != nil {
					return nil, err
				}
			}
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("cannot find blobber %s in allocation", remove.BlobberID)
	}

	if _, err := balances.InsertTrieNode(removedBlobber.GetKey(ADDRESS), removedBlobber); err != nil {
		return nil, fmt.Errorf("saving blobber %v, error: %v", removedBlobber.ID, err)
	}
	if err := emitUpdateBlobber(removedBlobber, balances); err != nil {
		return nil, fmt.Errorf("emitting blobber %s, error: %v", removedBlobber.ID, err)
	}

	blobber, err := ssc.getBlobber(removeId, balances)
	if err != nil {
		return nil, err
	}
	blobber.Used -= sa.bSize()
	_, err = balances.InsertTrieNode(blobber.GetKey(ssc.ID), blobber)
	if err != nil {
		return nil, err
	}

	return blobbers, nil
}

func (sa *StorageAllocation) changeBlobbers(
	blobbers []*StorageNode,
	addId, removeId string,
	ssc *StorageSmartContract,
	now common.Timestamp,
	balances chainstate.StateContextI,
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

	_, found := sa.BlobberMap[addId]
	if found {
		return nil, fmt.Errorf("allocatino already has blobber %s", addId)
	}

	addedBlobber, err := ssc.getBlobber(addId, balances)
	if err != nil {
		return nil, err
	}
	addedBlobber.Used += sa.bSize()
	afterSize := sa.bSize()

	blobbers = append(blobbers, addedBlobber)
	ba := newBlobberAllocation(afterSize, sa, addedBlobber, now)
	sa.BlobberMap[addId] = ba
	sa.BlobberDetails = append(sa.BlobberDetails, ba)

	var sp *stakePool
	if sp, err = ssc.getStakePool(addedBlobber.ID, balances); err != nil {
		return nil, fmt.Errorf("can't get blobber's stake pool: %v", err)
	}
	if err := sa.validateAllocationBlobber(addedBlobber, sp, now); err != nil {
		return nil, err
	}

	return blobbers, nil
}

func removeBlobberAllocation(
	removeId string,
	allocID string,
	allocPartitionIndex int,
	ssc *StorageSmartContract,
	balances chainstate.StateContextI) error {
	blobberAllocChallPartition, err := getBlobbersChallengeAllocationList(removeId, balances)
	if err != nil {
		return fmt.Errorf("cannot fetch blobber allocation partition: %v", err)
	}
	err = blobberAllocChallPartition.RemoveItem(balances, allocPartitionIndex, allocID)
	if err != nil {
		return fmt.Errorf("error removing allocation from challenge partition: %v", err)
	}

	blobberAllocChallSize, err := blobberAllocChallPartition.Size(balances)
	if err != nil {
		return fmt.Errorf("error getting size of challenge partition: %v", err)
	}
	if blobberAllocChallSize == 0 {
		bcPartitionLoc, err := ssc.getBlobberChallengePartitionLocation(removeId, balances)
		if err != nil {
			return fmt.Errorf("error retrieving blobber challenge partition location: %v", err)
		}

		bcPartition, err := getBlobbersChallengeList(balances)
		if err != nil {
			return fmt.Errorf("error retrieving blobber challenge partition: %v", err)
		}

		err = bcPartition.RemoveItem(balances, bcPartitionLoc.PartitionLocation.Location, removeId)
		if err != nil {
			return fmt.Errorf("error removing blobber from challenge partition: %v", err)
		}

		err = bcPartition.Save(balances)
		if err != nil {
			return fmt.Errorf("error saving blobber challenge partition: %v", err)
		}

		_, err = balances.DeleteTrieNode(bcPartitionLoc.GetKey(ssc.ID))
		if err != nil {
			return fmt.Errorf("error deleting blobber challenge partition location: %v", err)
		}
	}

	if err = blobberAllocChallPartition.Save(balances); err != nil {
		return fmt.Errorf("error saving allocation challenge partition: %v", err)
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
func (sa *StorageAllocation) restMinLockDemand() (rest state.Balance) {
	for _, details := range sa.BlobberDetails {
		if details.MinLockDemand > details.Spent {
			rest += details.MinLockDemand - details.Spent
		}
	}
	return
}

func (sa *StorageAllocation) getBlobbers(balances chainstate.StateContextI) error {

	for _, ba := range sa.BlobberDetails {
		blobber, err := balances.GetEventDB().GetBlobber(ba.BlobberID)
		if err != nil {
			return err
		}
		sn, err := blobberTableToStorageNode(*blobber)
		if err != nil {
			return err
		}
		sa.Blobbers = append(sa.Blobbers, &sn.StorageNode)
	}
	return nil
}

func (sa *StorageAllocation) addWritePoolOwner(userId string) {
	for _, id := range sa.WritePoolOwners {
		if userId == id {
			return
		}
	}
	sa.WritePoolOwners = append(sa.WritePoolOwners, userId)
}

func (sa *StorageAllocation) getAllocationPools(
	ssc *StorageSmartContract,
	balances chainstate.StateContextI,
) (*allocationWritePools, error) {
	var awp = allocationWritePools{
		ownerId: -1,
	}

	for i, wpOwner := range sa.WritePoolOwners {
		wp, err := ssc.getWritePool(wpOwner, balances)
		if err != nil {
			return nil, err
		}
		awp.writePools = append(awp.writePools, wp)
		cut := wp.Pools.allocationCut(sa.ID)
		for _, ap := range cut {
			awp.allocationPools.add(ap)
		}
		if wpOwner == sa.Owner {
			awp.ownerId = i
		}
	}

	if awp.ownerId < 0 {
		wp, err := ssc.getWritePool(sa.Owner, balances)
		if err != nil {
			return nil, err
		}
		awp.writePools = append(awp.writePools, wp)
		cut := wp.Pools.allocationCut(sa.ID)
		for _, ap := range cut {
			awp.allocationPools.add(ap)
		}
		awp.ownerId = len(awp.writePools) - 1
		sa.WritePoolOwners = append(sa.WritePoolOwners, sa.Owner)
	}
	awp.ids = sa.WritePoolOwners

	return &awp, nil
}

func (sa *StorageAllocation) validate(now common.Timestamp,
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
	var dur = common.ToTime(sa.Expiration).Sub(common.ToTime(now))
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
		if b.Capacity-b.Used < bsize {
			continue
		}
		// filter by max challenge completion time
		if b.Terms.ChallengeCompletionTime > sa.MaxChallengeCompletionTime {
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

func (sa *StorageAllocation) diversifyBlobbers(list []*StorageNode, size int) (diversified []*StorageNode) {
	if !sa.DiverseBlobbers {
		return list
	}

	if len(list) <= size {
		return list
	}

	// thanks to @shenwei356
	combinations := func(set []int, n int) (subsets [][]int) {
		length := uint(len(set))

		if n > len(set) {
			n = len(set)
		}

		for subsetBits := 1; subsetBits < (1 << length); subsetBits++ {
			if n > 0 && bits.OnesCount(uint(subsetBits)) != n {
				continue
			}

			var subset []int

			for object := uint(0); object < length; object++ {
				if (subsetBits>>object)&1 == 1 {
					subset = append(subset, set[object])
				}
			}
			subsets = append(subsets, subset)
		}
		return
	}

	// thanks to @cdipaolo
	distance := func(geoloc1, geoloc2 StorageNodeGeolocation) float64 {
		hsin := func(theta float64) float64 {
			return math.Pow(math.Sin(theta/2), 2)
		}

		var la1, lo1, la2, lo2 float64
		la1 = geoloc1.Latitude * math.Pi / 180
		lo1 = geoloc1.Longitude * math.Pi / 180
		la2 = geoloc2.Latitude * math.Pi / 180
		lo2 = geoloc2.Longitude * math.Pi / 180

		h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

		return math.Asin(math.Sqrt(h))
	}

	var maxD float64 // distance
	var maxDIndex int

	// create [1, ..., N] slice
	n := make([]int, len(list))
	for i := range n {
		n[i] = i
	}

	// get all combinations of s "size" elements from n "nodes"
	combs := combinations(n, size)

	// find out the max distance among combs of nodes
	for i, comb := range combs {
		var d float64 // distance

		// calculate distance for the combination
		combPairs := combinations(comb, 2)
		for _, combPair := range combPairs {
			d += distance(list[combPair[0]].Geolocation, list[combPair[1]].Geolocation)
		}

		// update the max distance value
		if d > maxD {
			maxD = d
			maxDIndex = i
		}
	}

	for _, v := range combs[maxDIndex] {
		diversified = append(diversified, list[v])
	}

	return
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
// In result, the changes is ordered as BlobberDetails field is ordered.
//
// For a case of allocation reducing, where no expiration, nor size changed
// we are using the same terms. And for this method, the oterms argument is
// nil for this case (meaning, terms hasn't changed).
func (sa *StorageAllocation) challengePoolChanges(odr, ndr common.Timestamp,
	oterms []Terms) (values []state.Balance) {

	// odr -- old duration remaining
	// ndr -- new duration remaining

	// in time units, instead of common.Timestamp
	var (
		odrtu = sa.durationInTimeUnits(odr)
		ndrtu = sa.durationInTimeUnits(ndr)
	)

	values = make([]state.Balance, 0, len(sa.BlobberDetails))

	for i, d := range sa.BlobberDetails {

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

		values = append(values, state.Balance(diff))
	}

	return
}

func (sa *StorageAllocation) IsValidFinalizer(id string) bool {
	if sa.Owner == id {
		return true // finalizing by owner
	}
	for _, d := range sa.BlobberDetails {
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
	sn.BlobberMap = make(map[string]*BlobberAllocation)
	for _, blobberAllocation := range sn.BlobberDetails {
		if blobberAllocation.Stats != nil {
			sn.UsedSize += blobberAllocation.Stats.UsedSize // total used
		}
		sn.BlobberMap[blobberAllocation.BlobberID] = blobberAllocation
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

	sn.BlobberMap = make(map[string]*BlobberAllocation)
	for _, blobberAllocation := range sn.BlobberDetails {
		if blobberAllocation.Stats != nil {
			sn.UsedSize += blobberAllocation.Stats.UsedSize // total used
		}
		sn.BlobberMap[blobberAllocation.BlobberID] = blobberAllocation
	}
	return o, nil
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
	balances chainstate.StateContextI,
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

type AuthTicket struct {
	ClientID        string           `json:"client_id"`
	OwnerID         string           `json:"owner_id"`
	AllocationID    string           `json:"allocation_id"`
	FilePathHash    string           `json:"file_path_hash"`
	ActualFileHash  string           `json:"actual_file_hash"`
	FileName        string           `json:"file_name"`
	RefType         string           `json:"reference_type"`
	Expiration      common.Timestamp `json:"expiration"`
	Timestamp       common.Timestamp `json:"timestamp"`
	ReEncryptionKey string           `json:"re_encryption_key"`
	Signature       string           `json:"signature"`
	Encrypted       bool             `json:"encrypted"`
}

func (at *AuthTicket) getHashData() string {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v:%v:%v:%v:%v",
		at.AllocationID, at.ClientID, at.OwnerID, at.FilePathHash,
		at.FileName, at.RefType, at.ReEncryptionKey, at.Expiration, at.Timestamp,
		at.ActualFileHash, at.Encrypted)
	return hashData
}

func (at *AuthTicket) verify(
	alloc *StorageAllocation,
	now common.Timestamp,
	clientID string,
	balances chainstate.StateContextI,
) (err error) {

	if at.AllocationID != alloc.ID {
		return common.NewError("invalid_read_marker",
			"Invalid auth ticket. Allocation ID mismatch")
	}

	if at.ClientID != clientID && len(at.ClientID) > 0 {
		return common.NewError("invalid_read_marker",
			"Invalid auth ticket. Client ID mismatch")
	}

	if at.Expiration > 0 && (at.Expiration < at.Timestamp || at.Expiration < now) {
		return common.NewError("invalid_read_marker",
			"Invalid auth ticket. Expired ticket")
	}

	if at.OwnerID != alloc.Owner {
		return common.NewError("invalid_read_marker",
			"Invalid auth ticket. Owner ID mismatch")
	}

	if at.Timestamp > now+2 {
		return common.NewError("invalid_read_marker",
			"Invalid auth ticket. Timestamp in future")
	}

	var ss = balances.GetSignatureScheme()

	if err = ss.SetPublicKey(alloc.OwnerPublicKey); err != nil {
		return common.NewErrorf("invalid_read_marker",
			"setting owner public key: %v", err)
	}

	var (
		sighash = encryption.Hash(at.getHashData())
		ok      bool
	)
	if ok, err = ss.Verify(at.Signature, sighash); err != nil || !ok {
		return common.NewError("invalid_read_marker",
			"Invalid auth ticket. Signature verification failed")
	}

	return
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
	PayerID         string           `json:"payer_id"`
	AuthTicket      *AuthTicket      `json:"auth_ticket"`
	ReadSize        float64          `json:"read_size"`
}

func (rm *ReadMarker) VerifySignature(clientPublicKey string, balances chainstate.StateContextI) bool {
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

func (rm *ReadMarker) verifyAuthTicket(alloc *StorageAllocation, now common.Timestamp, balances chainstate.StateContextI) (err error) {
	// owner downloads, pays itself, no ticket needed
	if rm.PayerID == alloc.Owner {
		return
	}
	// 3rd party payment
	if rm.AuthTicket == nil {
		return common.NewError("invalid_read_marker", "missing auth. ticket")
	}
	return rm.AuthTicket.verify(alloc, now, rm.PayerID, balances)
}

func (rm *ReadMarker) GetHashData() string {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v", rm.AllocationID,
		rm.BlobberID, rm.ClientID, rm.ClientPublicKey, rm.OwnerID,
		rm.ReadCounter, rm.Timestamp)
	return hashData
}

func (rm *ReadMarker) Verify(prevRM *ReadMarker, balances chainstate.StateContextI) error {
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

func (vt *ValidationTicket) VerifySign(balances chainstate.StateContextI) (bool, error) {
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
