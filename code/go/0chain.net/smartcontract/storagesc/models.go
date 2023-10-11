package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/provider"

	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/threshold/bls"
	"github.com/0chain/common/core/currency"

	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

//msgp:ignore StorageAllocation AllocationChallenges
//go:generate msgp -io=false -tests=false -unexported -v

var (
	AUTHORIZERS_COUNT_KEY            = ADDRESS + encryption.Hash("all_authorizers")
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
	ID             string           `json:"id"`
	CreatedAt      common.Timestamp `json:"created_at"`
	RoundCreatedAt int64            `json:"round_created_at"`
	BlobberID      string           `json:"blobber_id"` // blobber id
}

type AllocationChallenges struct {
	AllocationID   string                         `json:"allocation_id"`
	OpenChallenges []*AllocOpenChallenge          `json:"open_challenges"`
	ChallengeMap   map[string]*AllocOpenChallenge `json:"-" msg:"-"`
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
			ID:             challenge.ID,
			BlobberID:      challenge.BlobberID,
			CreatedAt:      challenge.Created,
			RoundCreatedAt: challenge.RoundCreatedAt,
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
	Responded       int64               `json:"responded"`
	RoundCreatedAt  int64               `json:"round_created_at"`
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
	provider.Provider
	BaseURL           string             `json:"url"`
	PublicKey         string             `json:"-" msg:"-"`
	StakePoolSettings stakepool.Settings `json:"stake_pool_settings"`
	LastHealthCheck   common.Timestamp   `json:"last_health_check"`
}

func validateBaseUrl(baseUrl *string) error {
	if baseUrl != nil && strings.Contains(*baseUrl, "localhost") &&
		node.Self.Host != "localhost" {
		return errors.New("invalid validator base url")
	}

	return nil
}

func GetValidatorUrlKey(globalKey, baseUrl string) datastore.Key {
	return datastore.Key(globalKey + "validator:" + baseUrl)
}

func (sn *ValidationNode) GetKey(_ string) datastore.Key {
	return provider.GetKey(sn.ID)
}

func (sn *ValidationNode) GetUrlKey(globalKey string) datastore.Key {
	return GetValidatorUrlKey(globalKey, sn.BaseURL)
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
}

// The minLockDemand returns min lock demand value for this Terms (the
// WritePrice and the MinLockDemand must be already set). Given size in GB and
// rest of allocation duration in time units are used.
func (t *Terms) minLockDemand(gbSize, rdtu, minLockDemand float64) (currency.Coin, error) {

	var mldf = float64(t.WritePrice) * gbSize * minLockDemand //
	return currency.Float64ToCoin(mldf * rdtu)                //
}

// validate a received terms
func (t *Terms) validate(conf *Config) (err error) {
	if err = validateReadPrice(t.ReadPrice, conf); err != nil {
		return
	}

	return validateWritePrice(t.WritePrice, conf)
}

func validateReadPrice(readPrice currency.Coin, conf *Config) error {
	if readPrice > conf.MaxReadPrice {
		return errors.New("read_price is greater than max_read_price allowed")
	}

	return nil
}

func validateWritePrice(writePrice currency.Coin, conf *Config) error {
	if writePrice < conf.MinWritePrice {
		return errors.New("write_price is less than min_write_price allowed")
	}
	if writePrice > conf.MaxWritePrice {
		return errors.New("write_price is greater than max_write_price allowed")
	}

	return nil
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

type RewardRound struct {
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
	provider.Provider
	BaseURL                 string                 `json:"url"`
	Geolocation             StorageNodeGeolocation `json:"geolocation"`
	Terms                   Terms                  `json:"terms"`     // terms
	Capacity                int64                  `json:"capacity"`  // total blobber capacity
	Allocated               int64                  `json:"allocated"` // allocated capacity
	PublicKey               string                 `json:"-"`
	SavedData               int64                  `json:"saved_data"`
	DataReadLastRewardRound float64                `json:"data_read_last_reward_round"` // in GB
	LastRewardDataReadRound int64                  `json:"last_reward_data_read_round"` // last round when data read was updated
	// StakePoolSettings used initially to create and setup stake pool.
	StakePoolSettings stakepool.Settings `json:"stake_pool_settings"`
	RewardRound       RewardRound        `json:"reward_round"`
	NotAvailable      bool               `json:"not_available"`
}

func GetUrlKey(baseUrl, globalKey string) datastore.Key {
	return datastore.Key(globalKey + baseUrl)
}

// validate the blobber configurations
func (sn *StorageNode) validate(conf *Config) (err error) {
	if err = sn.Terms.validate(conf); err != nil {
		return
	}
	if sn.Capacity <= conf.MinBlobberCapacity {
		return errors.New("insufficient blobber capacity")
	}

	if err := validateBaseUrl(&sn.BaseURL); err != nil {
		return err
	}

	if err := sn.Geolocation.validate(); err != nil {
		return err
	}

	return
}

func (sn *StorageNode) GetKey() datastore.Key {
	return provider.GetKey(sn.ID)
}

func (sn *StorageNode) GetUrlKey(globalKey string) datastore.Key {
	return GetUrlKey(sn.BaseURL, globalKey)
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
	ChallengePoolIntegralValue     currency.Coin    `json:"challenge_pool_integral_value"`
	LatestSuccessfulChallCreatedAt common.Timestamp `json:"latest_successful_chall_created_at"`
	LatestFinalizedChallCreatedAt  common.Timestamp `json:"latest_finalized_chall_created_att"`
}

func newBlobberAllocation(
	size int64,
	allocation *StorageAllocation,
	blobber *StorageNode,
	date common.Timestamp,
	timeUnit time.Duration,
) (*BlobberAllocation, error) {
	ba := &BlobberAllocation{}
	ba.Stats = &StorageAllocationStats{}
	ba.Size = size
	ba.Terms = blobber.Terms
	ba.AllocationID = allocation.ID
	ba.BlobberID = blobber.ID
	ba.LatestFinalizedChallCreatedAt = date
	ba.LatestSuccessfulChallCreatedAt = date

	rdtu, err := allocation.restDurationInTimeUnits(date, timeUnit)
	if err != nil {
		return nil, fmt.Errorf("new blobber allocation failed: %v", err)
	}

	ba.MinLockDemand, err = blobber.Terms.minLockDemand(sizeInGB(size), rdtu, allocation.MinLockDemand)
	return ba, err
}

// The upload used after commitBlobberConnection (size > 0) to calculate
// internal integral value.
func (d *BlobberAllocation) upload(size int64, now common.Timestamp,
	rdtu float64) (move currency.Coin, err error) {

	move = currency.Coin(sizeInGB(size) * float64(d.Terms.WritePrice) * rdtu)
	challengePoolIntegralValue, err := currency.AddCoin(d.ChallengePoolIntegralValue, move)
	if err != nil {
		return
	}
	d.ChallengePoolIntegralValue = challengePoolIntegralValue

	return
}

func (d *BlobberAllocation) removeBlobberPassRates(alloc *StorageAllocation, maxChallengeCompletionRounds int64, balances chainstate.StateContextI, sc *StorageSmartContract) (float64, error) {

	if alloc.Stats == nil {
		alloc.Stats = &StorageAllocationStats{}
	}

	passRate := 0.0

	allocChallenges, err := sc.getAllocationChallenges(alloc.ID, balances)
	if err != nil {
		if err == util.ErrValueNotPresent {
			return 1, nil
		} else {
			return 0, common.NewError("remove_blobber_pass_rates",
				"error fetching allocation challenge: "+err.Error())
		}
	}

	var nonRemovedChallenges []*AllocOpenChallenge
	var removedChallengeIds []string

	switch err {
	case util.ErrValueNotPresent:
		return 1, nil
	case nil:
		for _, oc := range allocChallenges.OpenChallenges {
			if oc.BlobberID != d.BlobberID {
				nonRemovedChallenges = append(nonRemovedChallenges, oc)
				continue
			}

			if d.Stats == nil {
				d.Stats = new(StorageAllocationStats) // make sure
			}

			var expire = oc.RoundCreatedAt + maxChallengeCompletionRounds
			currentRound := balances.GetBlock().Round

			d.Stats.OpenChallenges--
			alloc.Stats.OpenChallenges--

			if expire < currentRound {
				d.Stats.FailedChallenges++
				alloc.Stats.FailedChallenges++

				err := emitUpdateChallenge(&StorageChallenge{
					ID:           oc.ID,
					AllocationID: alloc.ID,
					BlobberID:    oc.BlobberID,
				}, false, ChallengeRespondedLate, balances, alloc.Stats)
				if err != nil {
					return 0.0, err
				}

			} else {
				d.Stats.SuccessChallenges++
				alloc.Stats.SuccessChallenges++

				err := emitUpdateChallenge(&StorageChallenge{
					ID:           oc.ID,
					AllocationID: alloc.ID,
					BlobberID:    oc.BlobberID,
				}, true, ChallengeResponded, balances, alloc.Stats)
				if err != nil {
					return 0.0, err
				}
			}

			removedChallengeIds = append(removedChallengeIds, oc.ID)
		}

	default:
		return 0.0, fmt.Errorf("getting allocation challenge: %v", err)
	}

	allocChallenges.OpenChallenges = nonRemovedChallenges

	// Save the allocation challenges to MPT
	if err := allocChallenges.Save(balances, sc.ID); err != nil {
		return 0, common.NewErrorf("remove_blobber_failed",
			"error storing alloc challenge: %v", err)
	}

	for _, challengeID := range removedChallengeIds {
		_, err := balances.DeleteTrieNode(storageChallengeKey(sc.ID, challengeID))
		if err != nil {
			return 0, common.NewErrorf("remove_blobber_failed", "could not delete challenge node: %v", err)
		}
	}

	blobbersSettledChallengesCount := d.Stats.OpenChallenges

	if d.Stats.OpenChallenges > 0 {
		logging.Logger.Warn("not all challenges canceled", zap.Int64("remaining", d.Stats.OpenChallenges))

		d.Stats.SuccessChallenges += d.Stats.OpenChallenges
		alloc.Stats.SuccessChallenges += d.Stats.OpenChallenges
		alloc.Stats.OpenChallenges -= d.Stats.OpenChallenges

		d.Stats.OpenChallenges = 0
	}

	if d.Stats.TotalChallenges == 0 {
		passRate = 1
	} else {
		passRate = float64(d.Stats.SuccessChallenges) / float64(d.Stats.TotalChallenges)
	}

	emitUpdateAllocationAndBlobberStatsOnBlobberRemoval(alloc, d.BlobberID, blobbersSettledChallengesCount, balances)

	return passRate, nil
}

func (d *BlobberAllocation) payMinLockDemand(alloc *StorageAllocation, sp *stakePool, balances chainstate.StateContextI) (currency.Coin, error) {

	if d.MinLockDemand > d.Spent {
		delta, err := currency.MinusCoin(d.MinLockDemand, d.Spent)
		if err != nil {
			return 0, err
		}
		if alloc.WritePool < delta {
			return 0, fmt.Errorf("paying min_lock for blobber %v"+
				"ammount was short by %v", d.BlobberID, delta)
		}
		alloc.WritePool, err = currency.MinusCoin(alloc.WritePool, delta)
		if err != nil {
			return 0, err
		}

		err = sp.DistributeRewards(delta, d.BlobberID, spenum.Blobber, spenum.MinLockDemandReward, balances, alloc.ID)
		if err != nil {
			return 0, fmt.Errorf("distribute rewards failed, paying min_lock %v for blobber "+
				"%v from write pool %v, minlock demand %v spent %v error %v",
				delta, d.BlobberID, alloc.WritePool, d.MinLockDemand, d.Spent, err.Error())
		}
		d.Spent, err = currency.AddCoin(d.Spent, delta)
		if err != nil {
			return 0, err
		}

		return delta, nil
	}

	return currency.Coin(0), nil
}

func (d *BlobberAllocation) payChallengePoolPassPayments(alloc *StorageAllocation, sp *stakePool, cp *challengePool, passRate float64, balances chainstate.StateContextI, conf *Config, now common.Timestamp, sc *StorageSmartContract) (currency.Coin, currency.Coin, error) {
	if d.LatestFinalizedChallCreatedAt == 0 {
		return 0, 0, nil
	}

	challengePenaltyPaid, err := d.challengePenaltyOnFinalization(conf, alloc, balances, sp)
	if err != nil {
		return 0, 0, common.NewError("challenge_penalty_on_finalization_error", err.Error())
	}

	challengeRewardPaid, err := d.challengeRewardOnFinalization(conf.TimeUnit, now, sp, cp, passRate, balances, alloc)
	if err != nil {
		return 0, 0, common.NewError("challenge_reward_on_finalization_error", err.Error())
	}

	return challengeRewardPaid, challengePenaltyPaid, nil
}

func (d *BlobberAllocation) challengeRewardOnFinalization(timeUnit time.Duration, now common.Timestamp, sp *stakePool, cp *challengePool, passRate float64, balances chainstate.StateContextI, alloc *StorageAllocation) (currency.Coin, error) {
	if now <= d.LatestSuccessfulChallCreatedAt {
		logging.Logger.Info("challenge reward on finalization: no reward", zap.Any("now", now), zap.Any("latest successful challenge created at", d.LatestSuccessfulChallCreatedAt))
		return 0, nil
	}

	payment := currency.Coin(0)

	rdtu, err := alloc.restDurationInTimeUnits(d.LatestFinalizedChallCreatedAt, timeUnit)
	if err != nil {
		return 0, fmt.Errorf("blobber reward failed: %v", err)
	}

	dtu, err := alloc.durationInTimeUnits(now-d.LatestFinalizedChallCreatedAt, timeUnit)
	if err != nil {
		return 0, fmt.Errorf("blobber reward failed: %v", err)
	}

	if dtu > rdtu {
		dtu = rdtu // now can be more for finalization
	}

	move := currency.Coin((dtu / rdtu) * float64(d.ChallengePoolIntegralValue))

	if alloc.Stats.UsedSize > 0 && cp.Balance > 0 && passRate > 0 && d.Stats != nil {
		reward, err := currency.MultFloat64(move, passRate)
		if err != nil {
			return payment, err
		}

		cv, err := currency.MinusCoin(d.ChallengePoolIntegralValue, reward)
		if err != nil {
			logging.Logger.Warn("challenge minus failed",
				zap.Error(err),
				zap.Any("dtu", dtu),
				zap.Any("rdtu", rdtu),
				zap.Any("challenge value", d.ChallengePoolIntegralValue),
				zap.Any("move", move))
			err = fmt.Errorf("minus challenge pool value failed: %v", err)
			return 0, err
		}
		d.ChallengePoolIntegralValue = cv

		err = sp.DistributeRewards(reward, d.BlobberID, spenum.Blobber, spenum.ChallengePassReward, balances, alloc.ID)
		if err != nil {
			return payment, fmt.Errorf("failed to distribute rewards blobber: %s, err: %v", d.BlobberID, err)
		}

		d.Spent, err = currency.AddCoin(d.Spent, reward)
		if err != nil {
			return payment, fmt.Errorf("blobber alloc spent: %v", err)
		}
		payment, err = currency.AddCoin(payment, reward)
		if err != nil {
			return payment, fmt.Errorf("pass payments: %v", err)
		}

	}

	return payment, nil
}

func (d *BlobberAllocation) challengePenaltyOnFinalization(conf *Config, alloc *StorageAllocation, balances chainstate.StateContextI, sp *stakePool) (currency.Coin, error) {
	if d.LatestSuccessfulChallCreatedAt >= d.LatestFinalizedChallCreatedAt {
		return 0, nil
	}

	rdtu, err := alloc.restDurationInTimeUnits(d.LatestSuccessfulChallCreatedAt, conf.TimeUnit)
	if err != nil {
		return 0, fmt.Errorf("blobber penalty failed: %v", err)
	}

	dtu, err := alloc.durationInTimeUnits(d.LatestFinalizedChallCreatedAt-d.LatestSuccessfulChallCreatedAt, conf.TimeUnit)
	if err != nil {
		return 0, fmt.Errorf("blobber penalty failed: %v", err)
	}

	if dtu > rdtu {
		dtu = rdtu // now can be more for finalization
	}

	move, err := d.challenge(dtu, rdtu)
	if err != nil {
		return 0, err
	}

	blobReturned, err := currency.AddCoin(d.Returned, move)
	if err != nil {
		return 0, err
	}
	d.Returned = blobReturned

	slash, err := currency.MultFloat64(move, conf.BlobberSlash)
	if err != nil {
		return 0, err
	}

	// blobber stake penalty
	if conf.BlobberSlash > 0 && move > 0 &&
		slash > 0 {

		dpMove, err := sp.slash(d.BlobberID, d.Offer(), slash, balances, alloc.ID)
		if err != nil {
			return 0, fmt.Errorf("can't slash tokens: %v", err)
		}

		penalty, err := currency.AddCoin(d.Penalty, dpMove) // penalty statistic
		if err != nil {
			return 0, err
		}
		d.Penalty = penalty

		logging.Logger.Info("Paying blobber penalty", zap.Any("penalty", dpMove), zap.Any("slash", slash), zap.Any("move", move), zap.Any("blobber", d.BlobberID))
	}

	return move, nil
}

func (d *BlobberAllocation) payCancellationCharge(alloc *StorageAllocation, sp *stakePool, balances chainstate.StateContextI, sc *StorageSmartContract, passRate float64, totalWritePrice, cancellationCharge currency.Coin) (currency.Coin, error) {
	blobberWritePriceWeight := float64(d.Terms.WritePrice) / float64(totalWritePrice)
	reward, err := currency.Float64ToCoin(float64(cancellationCharge) * blobberWritePriceWeight * passRate)

	err = sp.DistributeRewards(reward, d.BlobberID, spenum.Blobber, spenum.CancellationChargeReward, balances, alloc.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to distribute rewards, blobber: %s, err: %v", d.BlobberID, err)
	}

	blobber, err := sc.getBlobber(d.BlobberID, balances)
	if err != nil {
		return reward, common.NewError("fini_alloc_failed",
			"can't get blobber "+d.BlobberID+": "+err.Error())
	}
	blobber.SavedData += -d.Stats.UsedSize
	blobber.Allocated += -d.Size
	_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
	if err != nil {
		return reward, common.NewError("fini_alloc_failed",
			"saving blobber "+d.BlobberID+": "+err.Error())
	}

	// Update saved data on events_db
	emitUpdateBlobberAllocatedSavedHealth(blobber, balances)

	return reward, nil
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
func (d *BlobberAllocation) challenge(dtu, rdtu float64) (move currency.Coin, err error) {
	move = currency.Coin((dtu / rdtu) * float64(d.ChallengePoolIntegralValue))
	cv, err := currency.MinusCoin(d.ChallengePoolIntegralValue, move)
	if err != nil {
		logging.Logger.Warn("challenge minus failed",
			zap.Error(err),
			zap.Any("dtu", dtu),
			zap.Any("rdtu", rdtu),
			zap.Any("challenge value", d.ChallengePoolIntegralValue),
			zap.Any("move", move))
		err = fmt.Errorf("minus challenge pool value failed: %v", err)
		return
	}
	d.ChallengePoolIntegralValue = cv
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
	// Tx keeps hash with which the allocation has created or updated. todo do we need this field?
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
	FileOptions uint16 `json:"file_options"`

	WritePool currency.Coin `json:"write_pool"`

	// Requested ranges.
	ReadPriceRange  PriceRange `json:"read_price_range"`
	WritePriceRange PriceRange `json:"write_price_range"`

	// StartTime is time when the allocation has been created. We will
	// use it to check blobber's MaxOfferTime extending the allocation.
	StartTime common.Timestamp `json:"start_time"`
	// Finalized is true where allocation has been finalized.
	Finalized bool `json:"finalized,omitempty"`
	// Canceled set to true where allocation finalized by cancel_allocation
	// transaction.
	Canceled bool `json:"canceled,omitempty"`

	// MinLockDemand in number in [0; 1] range. It represents part of
	// allocation should be locked for the blobber rewards even if
	// user never write something to the blobber.
	MinLockDemand float64 `json:"min_lock_demand"`

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
}

type WithOption func(balances cstate.StateContextI) (currency.Coin, error)

func WithTokenMint(coin currency.Coin) WithOption {
	return func(balances cstate.StateContextI) (currency.Coin, error) {
		if err := balances.AddMint(&state.Mint{
			Minter:     ADDRESS,
			ToClientID: ADDRESS,
			Amount:     coin,
		}); err != nil {
			return 0, fmt.Errorf("minting tokens for write pool: %v", err)
		}
		return coin, nil
	}
}

func WithTokenTransfer(value currency.Coin, clientId, toClientId string) WithOption {
	return func(balances cstate.StateContextI) (currency.Coin, error) {
		if err := stakepool.CheckClientBalance(clientId, value, balances); err != nil {
			return 0, err
		}
		transfer := state.NewTransfer(clientId, toClientId, value)
		if err := balances.AddTransfer(transfer); err != nil {
			return 0, fmt.Errorf("adding transfer to allocation pool: %v", err)
		}

		return value, nil
	}
}

func (sa *StorageAllocation) addToWritePool(
	txn *transaction.Transaction,
	balances cstate.StateContextI,
	opts ...WithOption,
) error {
	// default behaviour
	if len(opts) == 0 {
		value, err := WithTokenTransfer(txn.Value, txn.ClientID, txn.ToClientID)(balances)
		if err != nil {
			return err
		}
		if writePool, err := currency.AddCoin(sa.WritePool, value); err != nil {
			return err
		} else {
			sa.WritePool = writePool
		}
	} else {
		for _, opt := range opts {
			value, err := opt(balances)
			if err != nil {
				return err
			}
			if writePool, err := currency.AddCoin(sa.WritePool, value); err != nil {
				return err
			} else {
				sa.WritePool = writePool
			}
		}

	}

	i, err := txn.Value.Int64()
	if err != nil {
		return err
	}
	balances.EmitEvent(event.TypeStats, event.TagLockWritePool, sa.ID, event.WritePoolLock{
		Client:       txn.ClientID,
		AllocationId: sa.ID,
		Amount:       i,
	})
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
		return fmt.Errorf("insufficient funds %v in write pool to pay %v", sa.WritePool, value)
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

func (sa *StorageAllocation) payMinLockDemand(sps []*stakePool, balances chainstate.StateContextI, t *transaction.Transaction) error {
	deductionFromWritePool := currency.Coin(0)

	for i, d := range sa.BlobberAllocs {
		// min lock demand rest
		minLockDeductionFromWritePool, err := d.payMinLockDemand(sa, sps[i], balances)
		if err != nil {
			return fmt.Errorf("error paying min lock demand: %v", err)
		}

		deductionFromWritePool, err = currency.AddCoin(deductionFromWritePool, minLockDeductionFromWritePool)
		if err != nil {
			return fmt.Errorf("error adding min lock demand to deductionFromWritePool: %v", err)
		}
	}

	i, err := deductionFromWritePool.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert deduction from write pool to int64: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUnlockWritePool, sa.ID, event.WritePoolLock{
		Client:       t.ClientID,
		AllocationId: sa.ID,
		Amount:       i,
	})

	return nil
}

func (sa *StorageAllocation) payMinLockDemandToRemoveBlobber(sp *stakePool, balances chainstate.StateContextI, clientID string, ba *BlobberAllocation) error {
	// min lock demand rest
	minLockDeductionFromWritePool, err := ba.payMinLockDemand(sa, sp, balances)
	if err != nil {
		return fmt.Errorf("error paying min lock demand: %v", err)
	}

	i, err := minLockDeductionFromWritePool.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert deduction from write pool to int64: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUnlockWritePool, sa.ID, event.WritePoolLock{
		Client:       clientID,
		AllocationId: sa.ID,
		Amount:       i,
	})

	return nil
}

func (sa *StorageAllocation) payChallengePoolPassPayments(sps []*stakePool, balances chainstate.StateContextI, cp *challengePool, passRates []float64, conf *Config, sc *StorageSmartContract, now common.Timestamp) error {
	var passPayments currency.Coin

	for i, d := range sa.BlobberAllocs {
		blobberPassPayment, _, err := d.payChallengePoolPassPayments(sa, sps[i], cp, passRates[i], balances, conf, now, sc)
		if err != nil {
			return fmt.Errorf("error paying challenge pool pass payments: %v", err)
		}

		passPayments, err = currency.AddCoin(passPayments, blobberPassPayment)
		if err != nil {
			return fmt.Errorf("error adding blobber pass payment to passPayments: %v", err)
		}
	}

	var err error
	prevBal := cp.Balance
	cp.Balance, err = currency.MinusCoin(cp.Balance, passPayments)
	if err != nil {
		return err
	}

	sa.MovedBack, err = currency.AddCoin(sa.MovedBack, cp.Balance)
	if err != nil {
		return err
	}

	err = sa.moveFromChallengePool(cp, cp.Balance)
	if err != nil {
		return fmt.Errorf("failed to move challenge pool back to write pool: %v", err)
	}

	if err = cp.save(sc.ID, sa, balances); err != nil {
		return fmt.Errorf("failed to save challenge pool: %v", err)
	}

	i, err := prevBal.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert balance: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, cp.ID, event.ChallengePoolLock{
		Client:       sa.Owner,
		AllocationId: sa.ID,
		Amount:       i,
	})

	return nil
}

func (sa *StorageAllocation) payChallengePoolPassPaymentsToRemoveBlobber(sp *stakePool, balances chainstate.StateContextI, cp *challengePool, passRate float64, conf *Config, sc *StorageSmartContract, ba *BlobberAllocation, now common.Timestamp) error {
	passPayment, penaltyPayment, err := ba.payChallengePoolPassPayments(sa, sp, cp, passRate, balances, conf, now, sc)
	if err != nil {
		return fmt.Errorf("error paying challenge pool pass payments: %v", err)
	}

	balance, err := currency.MinusCoin(cp.Balance, passPayment)
	if err != nil {
		return err
	}
	cp.Balance = balance

	sa.MovedBack, err = currency.AddCoin(sa.MovedBack, ba.ChallengePoolIntegralValue+penaltyPayment)
	if err != nil {
		return err
	}

	err = sa.moveFromChallengePool(cp, ba.ChallengePoolIntegralValue+penaltyPayment)
	if err != nil {
		return fmt.Errorf("failed to move challenge pool back to write pool: %v", err)
	}

	if err = cp.save(sc.ID, sa, balances); err != nil {
		return fmt.Errorf("failed to save challenge pool: %v", err)
	}

	fromChallengePool := ba.ChallengePoolIntegralValue + passPayment + penaltyPayment
	i, err := fromChallengePool.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert balance: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, cp.ID, event.ChallengePoolLock{
		Client:       sa.Owner,
		AllocationId: sa.ID,
		Amount:       i,
	})

	return nil
}

func (sa *StorageAllocation) payCancellationCharge(sps []*stakePool, balances chainstate.StateContextI, passRates []float64, conf *Config, sc *StorageSmartContract, t *transaction.Transaction) error {
	cancellationCharge, err := sa.cancellationCharge(conf.CancellationCharge)
	if err != nil {
		return fmt.Errorf("failed to get cancellation charge: %v", err)
	}

	if sa.WritePool < cancellationCharge {
		cancellationCharge = sa.WritePool
		logging.Logger.Error("insufficient funds, %v, for cancellation charge, %v. distributing the remaining write pool.")
	}

	totalWritePrice := currency.Coin(0)
	for _, ba := range sa.BlobberAllocs {
		totalWritePrice, err = currency.AddCoin(totalWritePrice, ba.Terms.WritePrice)
		if err != nil {
			return fmt.Errorf("failed to add write price: %v", err)
		}
	}

	totalCancellationChargePaid := currency.Coin(0)

	for i, ba := range sa.BlobberAllocs {
		blobberCancellationChargePaid, err := ba.payCancellationCharge(sa, sps[i], balances, sc, passRates[i], totalWritePrice, cancellationCharge)
		if err != nil {
			return fmt.Errorf("1 error paying cancellation charge: %v", err)
		}

		totalCancellationChargePaid, err = currency.AddCoin(totalCancellationChargePaid, blobberCancellationChargePaid)
		if err != nil {
			return fmt.Errorf("error adding blobber cancellation charge paid to totalCancellationChargePaid: %v", err)
		}
	}

	sa.WritePool, err = currency.MinusCoin(sa.WritePool, totalCancellationChargePaid)
	if err != nil {
		return fmt.Errorf("failed to deduct cancellation charges from write pool: %v", err)
	}

	i, err := totalCancellationChargePaid.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert deduction from write pool to int64: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUnlockWritePool, sa.ID, event.WritePoolLock{
		Client:       t.ClientID,
		AllocationId: sa.ID,
		Amount:       i,
	})

	return nil
}

func (sa *StorageAllocation) payCancellationChargeToRemoveBlobber(sp *stakePool, balances chainstate.StateContextI, passRate float64, conf *Config, sc *StorageSmartContract, clientID string, ba *BlobberAllocation) error {
	cancellationCharge, err := sa.cancellationCharge(conf.CancellationCharge)
	if err != nil {
		return fmt.Errorf("failed to get cancellation charge: %v", err)
	}

	if sa.WritePool < cancellationCharge {
		cancellationCharge = sa.WritePool
		logging.Logger.Error("insufficient funds, %v, for cancellation charge, %v. distributing the remaining write pool.")
	}

	totalWritePrice := currency.Coin(0)
	for _, ba := range sa.BlobberAllocs {
		totalWritePrice, err = currency.AddCoin(totalWritePrice, ba.Terms.WritePrice)
		if err != nil {
			return fmt.Errorf("failed to add write price: %v", err)
		}
	}

	totalCancellationChargePaid, err := ba.payCancellationCharge(sa, sp, balances, sc, passRate, totalWritePrice, cancellationCharge)
	if err != nil {
		return fmt.Errorf("2 error paying cancellation charge: %v", err)
	}

	sa.WritePool, err = currency.MinusCoin(sa.WritePool, totalCancellationChargePaid)
	if err != nil {
		return fmt.Errorf("failed to deduct cancellation charges from write pool: %v", err)
	}

	i, err := totalCancellationChargePaid.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert deduction from write pool to int64: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagUnlockWritePool, sa.ID, event.WritePoolLock{
		Client:       clientID,
		AllocationId: sa.ID,
		Amount:       i,
	})

	return nil
}

func (sa *StorageAllocation) isActive(
	blobber *StorageNode,
	total, offers currency.Coin,
	conf *Config,
	now common.Timestamp,
) error {
	active, reason := blobber.Provider.IsActive(now, conf.HealthCheckPeriod)
	if !active {
		return fmt.Errorf("blobber %s is not active, %s", blobber.ID, reason)
	}

	if blobber.NotAvailable {
		return fmt.Errorf("blobber %s is not currently available for new allocations", blobber.ID)
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

	bSize := sa.bSize()
	// filter by blobber's capacity left
	if blobber.Capacity-blobber.Allocated < bSize {
		return fmt.Errorf("blobber %s free capacity %v insufficient, wanted %v",
			blobber.ID, blobber.Capacity-blobber.Allocated, bSize)
	}

	unallocCapacity, err := unallocatedCapacity(blobber.Terms.WritePrice, total, offers)
	if err != nil {
		return fmt.Errorf("failed to get unallocated capacity: %v", err)
	}

	if blobber.Terms.WritePrice > 0 && unallocCapacity < bSize {
		return fmt.Errorf("blobber %v staked capacity %v is insufficient, wanted %v",
			blobber.ID, unallocCapacity, bSize)
	}

	return nil
}

func (sa *StorageAllocation) cost() (currency.Coin, error) {
	var cost currency.Coin
	for _, ba := range sa.BlobberAllocs {
		c, err := currency.MultFloat64(ba.Terms.WritePrice, sizeInGB(ba.Size))
		if err != nil {
			return 0, err
		}
		cost, err = currency.AddCoin(cost, c)
		if err != nil {
			return 0, err
		}
	}
	return cost, nil
}

func (sa *StorageAllocation) cancellationCharge(cancellationFraction float64) (currency.Coin, error) {
	cost, err := sa.cost()
	if err != nil {
		return 0, err
	}
	return currency.MultFloat64(cost, cancellationFraction)
}

func (sa *StorageAllocation) checkFunding(cancellationFraction float64) error {
	cancellationCharge, err := sa.cancellationCharge(cancellationFraction)
	if err != nil {
		return err
	}
	mld, err := sa.restMinLockDemand()
	if err != nil {
		return err
	}
	if sa.WritePool < cancellationCharge+mld {
		return fmt.Errorf("not enough tokens to honor the cancellation charge plus min lock demand"+" (%d < %d + %d)",
			sa.WritePool, cancellationCharge, mld)
	}

	return nil
}

func (sa *StorageAllocation) bSize() int64 {
	return bSize(sa.Size, sa.DataShards)
}

func bSize(size int64, dataShards int) int64 {
	return int64(math.Ceil(float64(size) / float64(dataShards)))
}

func (sa *StorageAllocation) replaceBlobber(blobberID string, sc *StorageSmartContract, balances chainstate.StateContextI, clientID string, addedBlobberAllocation *BlobberAllocation, now common.Timestamp) error {
	_, ok := sa.BlobberAllocsMap[blobberID]
	if !ok {
		return fmt.Errorf("cannot find blobber %s in allocation", blobberID)
	}
	delete(sa.BlobberAllocsMap, blobberID)

	conf, err := getConfig(balances)
	if err != nil {
		return common.NewError("can't get config", err.Error())
	}

	for i, d := range sa.BlobberAllocs {
		if d.BlobberID == blobberID {
			if d.Stats.UsedSize > 0 {
				if err := removeAllocationFromBlobberPartitions(balances, d.BlobberID, d.AllocationID); err != nil {
					return err
				}
			}

			passRate, err := d.removeBlobberPassRates(sa, conf.MaxChallengeCompletionRounds, balances, sc)
			if err != nil {
				logging.Logger.Info("error removing blobber pass rates",
					zap.Any("allocation", sa.ID),
					zap.Any("blobber", d.BlobberID),
					zap.Error(err))
				return fmt.Errorf("error removing blobber pass rates: %v", err)
			}

			sp, err := sc.getStakePool(spenum.Blobber, d.BlobberID, balances)
			if err != nil {
				return common.NewError("remove_blobber_failed",
					"can't get stake pool of "+d.BlobberID+": "+err.Error())
			}
			if err := sp.reduceOffer(d.Offer()); err != nil {
				return common.NewError("remove_blobber_failed",
					"error removing offer: "+err.Error())
			}

			if err := sa.payMinLockDemandToRemoveBlobber(sp, balances, clientID, d); err != nil {
				return fmt.Errorf("error paying min lock demand: %v", err)
			}

			cp, err := sc.getChallengePool(sa.ID, balances)
			if err != nil {
				return fmt.Errorf("could not get challenge pool of alloc: %s, err: %v", sa.ID, err)
			}

			if err = sa.payChallengePoolPassPaymentsToRemoveBlobber(sp, balances, cp, passRate, conf, sc, d, now); err != nil {
				return fmt.Errorf("error paying challenge pool pass payments: %v", err)
			}

			if err = sa.payCancellationChargeToRemoveBlobber(sp, balances, passRate, conf, sc, clientID, d); err != nil {
				return fmt.Errorf("3 error paying cancellation charge: %v", err)
			}

			sa.BlobberAllocs[i] = addedBlobberAllocation
			break
		}
	}

	return nil
}

func replaceBlobber(
	sa *StorageAllocation,
	blobbers []*StorageNode,
	blobberID string,
	balances cstate.StateContextI,
	sc *StorageSmartContract,
	clientID string,
	addedBlobber *StorageNode, addedBlobberAllocation *BlobberAllocation, now common.Timestamp) ([]*StorageNode, error) {

	if err := sa.replaceBlobber(blobberID, sc, balances, clientID, addedBlobberAllocation, now); err != nil {
		return nil, err
	}

	var removedBlobber *StorageNode
	var found bool
	for i, d := range blobbers {
		if d.ID == blobberID {
			removedBlobber = blobbers[i]
			blobbers[i] = addedBlobber
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("cannot find blobber %s in allocation", blobberID)
	}

	if _, err := balances.InsertTrieNode(removedBlobber.GetKey(), removedBlobber); err != nil {
		return nil, fmt.Errorf("saving blobber %v, error: %v", removedBlobber.ID, err)
	}
	return blobbers, nil
}

func (sa *StorageAllocation) changeBlobbers(
	conf *Config,
	blobbers []*StorageNode,
	addId, removeId string,
	now common.Timestamp,
	balances cstate.StateContextI,
	sc *StorageSmartContract,
	clientID string,
) ([]*StorageNode, error) {
	var err error

	_, found := sa.BlobberAllocsMap[addId]
	if found {
		return nil, fmt.Errorf("allocation already has blobber %s", addId)
	}

	addedBlobber, err := getBlobber(addId, balances)
	if err != nil {
		return nil, fmt.Errorf("can't get blobber %s to add : %v", addId, err)
	}

	var sp *stakePool
	if sp, err = getStakePool(spenum.Blobber, addedBlobber.ID, balances); err != nil {
		return nil, fmt.Errorf("can't get blobber's stake pool: %v", err)
	}
	staked, err := sp.stake()
	if err != nil {
		return nil, err
	}

	if err := sa.isActive(addedBlobber, staked, sp.TotalOffers, conf, now); err != nil {
		return nil, err
	}

	addedBlobber.Allocated += sa.bSize() // Why increase allocation then check if the free capacity is enough?
	afterSize := sa.bSize()

	ba, err := newBlobberAllocation(afterSize, sa, addedBlobber, now, conf.TimeUnit)
	if err != nil {
		return nil, fmt.Errorf("can't allocate blobber: %v", err)
	}

	if len(removeId) > 0 {
		if blobbers, err = replaceBlobber(sa, blobbers, removeId, balances, sc, clientID, addedBlobber, ba, now); err != nil {
			return nil, err
		}
	} else {
		// If we are not removing a blobber, then the number of shards must increase.
		sa.ParityShards++

		blobbers = append(blobbers, addedBlobber)
		sa.BlobberAllocs = append(sa.BlobberAllocs, ba)
	}

	sa.BlobberAllocsMap[addId] = ba

	if err := sp.addOffer(ba.Offer()); err != nil {
		return nil, fmt.Errorf("failed to add offter: %v", err)
	}

	if err := sp.Save(spenum.Blobber, addId, balances); err != nil {
		return nil, err
	}

	return blobbers, nil
}

func (sa *StorageAllocation) save(state cstate.StateContextI, scAddress string) error {
	_, err := state.InsertTrieNode(sa.GetKey(scAddress), sa)
	return err
}

type StorageAllocationDecode StorageAllocation

// The restMinLockDemand returns number of tokens required as min_lock_demand;
// if a blobber receive write marker, then some token moves to related
// challenge pool and 'Spent' of this blobber is increased; thus, the 'Spent'
// reduces the rest of min_lock_demand of this blobber; but, if a malfunctioning
// client doesn't send a data to a blobber (or blobbers) then this blobbers
// don't receive tokens, their spent will be zero, and the min lock demand
// will be blobber reward anyway.
func (sa *StorageAllocation) restMinLockDemand() (rest currency.Coin, err error) {
	for _, details := range sa.BlobberAllocs {
		if details.MinLockDemand > details.Spent {
			rest, err = currency.AddCoin(rest, details.MinLockDemand-details.Spent)
			if err != nil {
				return
			}
		}
	}
	return
}

type filterBlobberFunc func(blobber *StorageNode) (kick bool, err error)

type filterValidatorFunc func(validator *ValidationNode) (kick bool, err error)

func (sa *StorageAllocation) filterBlobbers(list []*StorageNode,
	creationDate common.Timestamp, bsize int64, filters ...filterBlobberFunc) (
	filtered []*StorageNode, err error) {

	var (
		i int
	)

List:
	for _, b := range list {
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
			kick, err := filter(b)
			if err != nil {
				return nil, err
			}

			if kick {
				continue List
			}
		}
		list[i] = b
		i++
	}

	return list[:i], nil
}

// validateEachBlobber (this is a copy paste version of filterBlobbers with minute modification for verifications)
func (sa *StorageAllocation) validateEachBlobber(
	blobbers []*storageNodeResponse,
	creationDate common.Timestamp,
	conf *Config,
) ([]*StorageNode, []string) {
	var (
		errs     = make([]string, 0, len(blobbers))
		filtered = make([]*StorageNode, 0, len(blobbers))
	)
	for _, b := range blobbers {
		sn := StoragNodeResponseToStorageNode(*b)
		err := sa.isActive(&sn, b.TotalStake, b.TotalOffers, conf, creationDate)
		if err != nil {
			logging.Logger.Debug("error validating blobber", zap.String("id", b.ID), zap.Error(err))
			errs = append(errs, err.Error())
			continue
		}
		filtered = append(filtered, &sn)
	}
	return filtered, errs
}

// Until returns allocation expiration.
func (sa *StorageAllocation) Until(duration time.Duration) common.Timestamp {
	return sa.Expiration + toSeconds(duration)
}

// The durationInTimeUnits returns given duration (represented as
// common.Timestamp) as duration in time units (float point value) for
// this allocation (time units for the moment of the allocation creation).
func (sa *StorageAllocation) durationInTimeUnits(dur common.Timestamp, timeUnit time.Duration) (float64, error) {
	if dur < 0 {
		return 0, errors.New("negative duration")
	}
	return float64(dur.Duration()) / float64(timeUnit), nil
}

// The restDurationInTimeUnits return rest duration of the allocation in time
// units as a float64 value.
func (sa *StorageAllocation) restDurationInTimeUnits(now common.Timestamp, timeUnit time.Duration) (float64, error) {
	if sa.Expiration < now {
		logging.Logger.Error("rest duration time overflow, timestamp is beyond alloc expiration",
			zap.Int64("now", int64(now)),
			zap.Int64("alloc expiration", int64(sa.Expiration)))
		return 0, errors.New("rest duration time overflow, timestamp is beyond alloc expiration")
	}
	logging.Logger.Info("rest_duration", zap.Int64("expiration", int64(sa.Expiration)), zap.Int64("now", int64(now)), zap.Float64("timeUnit", float64(timeUnit)), zap.Int64("rest", int64(sa.Expiration-now)))
	return sa.durationInTimeUnits(sa.Expiration-now, timeUnit)
}

// For a stored files (size). Changing an allocation duration and terms
// (weighted average). We need to move more tokens to related challenge pool.
// Or move some tokens from the challenge pool back.
//
// For example, we have allocation for 1 time unit (let it be mouth), with
// 1 GB of stored files. For the 1GB related challenge pool originally filled
// up with
//
//	(integral): write_price * size * duration
//	e.g.: (integral) write_price * 1 GB * 1 month
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
//	   a = old_write_price * size * old_duration_remaining (old expiration)
//	   b = new_write_price * size * new_duration_remaining (new expiration)
//
//	And the difference is
//
//	   b - a (move to challenge pool, or move back from challenge pool)
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

type ChallengePoolChanges struct {
	Value      currency.Coin
	isNegative bool
}

func (sa *StorageAllocation) challengePoolChanges(odr, ndr common.Timestamp, timeUnit time.Duration,
	oterms []Terms) (values []ChallengePoolChanges, err error) {

	// odr -- old duration remaining
	// ndr -- new duration remaining

	// in time units, instead of common.Timestamp
	odrtu, err := sa.durationInTimeUnits(odr, timeUnit)
	if err != nil {
		return nil, fmt.Errorf("failed to get old challenge pool duration: %v", err)
	}

	ndrtu, err := sa.durationInTimeUnits(ndr, timeUnit)
	if err != nil {
		return nil, fmt.Errorf("failed to get new challenge pool duration: %v", err)
	}

	values = make([]ChallengePoolChanges, 0, len(sa.BlobberAllocs))

	for i, d := range sa.BlobberAllocs {
		if d.Stats == nil || d.Stats.UsedSize == 0 {
			values = append(values, ChallengePoolChanges{Value: 0, isNegative: false}) // no data, no changes
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

		if diff < 0 {
			diff = -diff
			values = append(values, ChallengePoolChanges{Value: currency.Coin(diff), isNegative: true})
		} else {
			values = append(values, ChallengePoolChanges{Value: currency.Coin(diff), isNegative: false})
		}
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
		sn.BlobberAllocsMap[blobberAllocation.BlobberID] = blobberAllocation
	}
	return o, nil
}

// removeExpiredChallenges removes all expired challenges from the allocation,
// return the expired challenge ids per blobber (maps blobber id to its expiredIDs), or error if any.
// the expired challenge ids could be used to delete the challenge node from MPT when needed
func (sa *StorageAllocation) removeExpiredChallenges(
	allocChallenges *AllocationChallenges,
	cct int64,
	balances cstate.StateContextI,
	sc *StorageSmartContract,
) (int, error) {

	var expiredChallengeBlobberMap = make(map[string]string)
	var nonExpiredChallenges []*AllocOpenChallenge

	for _, oc := range allocChallenges.OpenChallenges {
		if !isChallengeExpired(balances.GetBlock().Round, oc.RoundCreatedAt, cct) {
			nonExpiredChallenges = append(nonExpiredChallenges, oc)
			continue
		}

		// expired
		expiredChallengeBlobberMap[oc.ID] = oc.BlobberID

		ba, ok := sa.BlobberAllocsMap[oc.BlobberID]
		if ok {
			ba.Stats.FailedChallenges++
			ba.Stats.OpenChallenges--
			sa.Stats.FailedChallenges++
			sa.Stats.OpenChallenges--

			if ba.LatestFinalizedChallCreatedAt < oc.CreatedAt {
				ba.LatestFinalizedChallCreatedAt = oc.CreatedAt
			}

			err := emitUpdateChallenge(&StorageChallenge{
				ID:           oc.ID,
				AllocationID: sa.ID,
				BlobberID:    oc.BlobberID,
			}, false, ChallengeRespondedLate, balances, sa.Stats)

			if err != nil {
				return 0, err
			}
		}
	}

	allocChallenges.OpenChallenges = nonExpiredChallenges

	var expChalIDs []string
	for challengeID := range expiredChallengeBlobberMap {
		expChalIDs = append(expChalIDs, challengeID)
	}

	// maps blobberID to count of its expiredIDs.
	for _, challengeID := range expChalIDs {
		_, err := balances.DeleteTrieNode(storageChallengeKey(sc.ID, challengeID))
		if err != nil {
			return 0, common.NewErrorf("remove_expired_challenges", "could not delete challenge node: %v", err)
		}
	}

	return len(expChalIDs), nil
}

// removeOldChallenges removes all open challenges from the allocation that are old
func (sa *StorageAllocation) removeOldChallenges(
	allocChallenges *AllocationChallenges,
	balances cstate.StateContextI,
	currentChallenge *StorageChallenge,
	sc *StorageSmartContract,
) error {
	var nonRemovedChallenges []*AllocOpenChallenge
	var expChalIDs []string

	for _, oc := range allocChallenges.OpenChallenges {
		if oc.RoundCreatedAt >= currentChallenge.RoundCreatedAt || oc.BlobberID != currentChallenge.BlobberID {
			nonRemovedChallenges = append(nonRemovedChallenges, oc)
			continue
		}

		logging.Logger.Info("removeOldChallenges",
			zap.String("challenge_id", oc.ID),
			zap.String("blobber_id", oc.BlobberID),
			zap.Int64("round_created_at", oc.RoundCreatedAt),
			zap.Int64("current_round_created_at", currentChallenge.RoundCreatedAt),
		)

		expChalIDs = append(expChalIDs, oc.ID)

		ba, ok := sa.BlobberAllocsMap[oc.BlobberID]
		if ok {
			ba.Stats.FailedChallenges++
			ba.Stats.OpenChallenges--
			sa.Stats.FailedChallenges++
			sa.Stats.OpenChallenges--

			if ba.LatestFinalizedChallCreatedAt < oc.CreatedAt {
				ba.LatestFinalizedChallCreatedAt = oc.CreatedAt
			}

			err := emitUpdateChallenge(&StorageChallenge{
				ID:           oc.ID,
				AllocationID: sa.ID,
				BlobberID:    oc.BlobberID,
			}, false, ChallengeOldRemoved, balances, sa.Stats)

			if err != nil {
				return err
			}
		}
	}

	allocChallenges.OpenChallenges = nonRemovedChallenges

	// maps blobberID to count of its expiredIDs.

	for _, challengeID := range expChalIDs {
		_, err := balances.DeleteTrieNode(storageChallengeKey(sc.ID, challengeID))
		if err != nil {
			return common.NewErrorf("remove_old_challenges", "could not delete challenge node: %v", err)
		}
	}

	return nil
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
	FileMetaRoot           string           `json:"file_meta_root"`
	AllocationID           string           `json:"allocation_id"`
	Size                   int64            `json:"size"`
	BlobberID              string           `json:"blobber_id"`
	Timestamp              common.Timestamp `json:"timestamp"`
	ClientID               string           `json:"client_id"`
	Signature              string           `json:"signature"`
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
	hashData := fmt.Sprintf(
		"%s:%s:%s:%s:%s:%s:%d:%d",
		wm.AllocationRoot, wm.PreviousAllocationRoot,
		wm.FileMetaRoot, wm.AllocationID,
		wm.BlobberID, wm.ClientID, wm.Size, wm.Timestamp)
	return hashData
}

func (wm *WriteMarker) Verify() bool {
	if len(wm.AllocationID) == 0 || len(wm.BlobberID) == 0 ||
		len(wm.ClientID) == 0 || wm.Timestamp == 0 {
		return false
	}
	return true
}

type ReadConnection struct {
	ReadMarker *ReadMarker `json:"read_marker"`
}

func (rc *ReadConnection) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey +
		encryption.Hash(rc.ReadMarker.BlobberID+rc.ReadMarker.ClientID+rc.ReadMarker.AllocationID))
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

func (vt *ValidationTicket) Validate(challengeID, blobberID string) error {
	if err := encryption.VerifyPublicKeyClientID(vt.ValidatorKey, vt.ValidatorID); err != nil {
		return fmt.Errorf("invalid validator tickets: %v", err)
	}

	if vt.ChallengeID != challengeID {
		return errors.New("challenge id does not match")
	}

	if vt.BlobberID != blobberID {
		return errors.New("challenge blobber id does not match")
	}

	return nil
}
