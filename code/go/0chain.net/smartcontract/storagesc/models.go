package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/0chain/common/core/statecache"

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
	"0chain.net/core/util/entitywrapper"
)

//msgp:ignore StorageNode StorageAllocation AllocationChallenges
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

//nolint:unused
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

func (vn *ValidationNode) GetKey() datastore.Key {
	return provider.GetKey(vn.ID)
}

func (vn *ValidationNode) GetUrlKey(globalKey string) datastore.Key {
	return GetValidatorUrlKey(globalKey, vn.BaseURL)
}

func (vn *ValidationNode) Encode() []byte {
	buff, _ := json.Marshal(vn)
	return buff
}

func (vn *ValidationNode) Decode(input []byte) error {
	err := json.Unmarshal(input, vn)
	if err != nil {
		return err
	}
	return nil
}

func (vn *ValidationNode) GetHash() string {
	return util.ToHex(vn.GetHashBytes())
}

func (vn *ValidationNode) GetHashBytes() []byte {
	return encryption.RawHash(vn.Encode())
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
	WritePrice currency.Coin `json:"write_price"`
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

func GetUrlKey(baseUrl, globalKey string) datastore.Key {
	return globalKey + baseUrl
}

func GetKilledIdKey(id, providerType string) datastore.Key {
	return "killed:" + providerType + ":" + id
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
	// for the allocation. If a user extends an allocation then
	// we calculate new weighted average terms based on previous terms,
	// size and expiration and new terms size and expiration.
	Terms Terms `json:"terms"`
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
	allocation *storageAllocationBase,
	blobber *storageNodeBase,
	conf *Config,
	date common.Timestamp,
) *BlobberAllocation {
	ba := &BlobberAllocation{}
	ba.Stats = &StorageAllocationStats{}
	ba.Size = size

	setCappedPrices(ba, blobber, conf)
	ba.AllocationID = allocation.ID
	ba.BlobberID = blobber.ID
	ba.LatestFinalizedChallCreatedAt = date
	ba.LatestSuccessfulChallCreatedAt = date

	return ba
}

func setCappedPrices(ba *BlobberAllocation, blobber *storageNodeBase, conf *Config) {
	//TODO check the maximum price of the network and choose minimum
	ba.Terms = blobber.Terms
	if blobber.Terms.WritePrice > conf.MaxWritePrice {
		ba.Terms.WritePrice = conf.MaxWritePrice
	}
	if blobber.Terms.ReadPrice > conf.MaxReadPrice {
		ba.Terms.ReadPrice = conf.MaxReadPrice
	}
}

// The upload used after commitBlobberConnection (size > 0) to calculate
// internal integral value.
func (d *BlobberAllocation) upload(size int64, rdtu float64, writePool currency.Coin) (move currency.Coin, err error) {

	move = currency.Coin(sizeInGB(size) * float64(d.Terms.WritePrice) * rdtu)

	if move > writePool {
		move = writePool
	}

	challengePoolIntegralValue, err := currency.AddCoin(d.ChallengePoolIntegralValue, move)
	if err != nil {
		return
	}
	d.ChallengePoolIntegralValue = challengePoolIntegralValue

	return
}

func (d *BlobberAllocation) removeBlobberPassRates(alloc *storageAllocationBase, maxChallengeCompletionRounds int64, balances chainstate.StateContextI, sc *StorageSmartContract) (float64, error) {

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

func (d *BlobberAllocation) payChallengePoolPassPayments(alloc *storageAllocationBase, sp *stakePool, cp *challengePool, passRate float64, balances chainstate.StateContextI, conf *Config, now common.Timestamp, sc *StorageSmartContract) (currency.Coin, currency.Coin, error) {
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

func (d *BlobberAllocation) challengeRewardOnFinalization(timeUnit time.Duration, now common.Timestamp, sp *stakePool, cp *challengePool, passRate float64, balances chainstate.StateContextI, alloc *storageAllocationBase) (currency.Coin, error) {
	if now <= d.LatestFinalizedChallCreatedAt {
		logging.Logger.Info("challenge reward on finalization", zap.Any("now", now), zap.Any("latest finalized challenge created at", d.LatestFinalizedChallCreatedAt))
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

		payment, err = currency.AddCoin(payment, reward)
		if err != nil {
			return payment, fmt.Errorf("pass payments: %v", err)
		}
	}

	return payment, nil
}

func (d *BlobberAllocation) challengePenaltyOnFinalization(conf *Config, alloc *storageAllocationBase, balances chainstate.StateContextI, sp *stakePool) (currency.Coin, error) {
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

func (d *BlobberAllocation) payCancellationCharge(alloc *storageAllocationBase, sp *stakePool, balances chainstate.StateContextI, sc *StorageSmartContract, passRate float64, totalWritePrice, cancellationCharge currency.Coin) (currency.Coin, error) {
	blobberWritePriceWeight := float64(d.Terms.WritePrice) / float64(totalWritePrice)
	reward, _ := currency.Float64ToCoin(float64(cancellationCharge) * blobberWritePriceWeight * passRate)

	err := sp.DistributeRewards(reward, d.BlobberID, spenum.Blobber, spenum.CancellationChargeReward, balances, alloc.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to distribute rewards, blobber: %s, err: %v", d.BlobberID, err)
	}

	return reward, nil
}

func (d *BlobberAllocation) Offer() currency.Coin {
	return currency.Coin(sizeInGB(d.Size) * float64(d.Terms.WritePrice))
}

// The upload used after commitBlobberConnection (size < 0) to calculate
// internal integral value. The size argument expected to be positive (not
// negative).
func (d *BlobberAllocation) delete(size int64, now common.Timestamp,
	rdtu float64) (move currency.Coin, err error) {

	move = currency.Coin(sizeInGB(size) * float64(d.Terms.WritePrice) * rdtu)

	if move > d.ChallengePoolIntegralValue {
		move = d.ChallengePoolIntegralValue
	}

	cv, err := currency.MinusCoin(d.ChallengePoolIntegralValue, move)
	if err != nil {
		logging.Logger.Warn("delete minus failed",
			zap.Error(err),
			zap.Any("dtu", rdtu),
			zap.Any("challenge value", d.ChallengePoolIntegralValue),
			zap.Any("move", move))
		err = fmt.Errorf("minus challenge pool value failed: %v", err)
		return
	}
	d.ChallengePoolIntegralValue = cv

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

type Transfer struct {
	value      currency.Coin
	clientId   string
	toClientId string
	isMint     bool
}

func (t *Transfer) transfer(balances cstate.StateContextI) (currency.Coin, error) {
	if t.value == 0 {
		return 0, nil
	}
	if err := stakepool.CheckClientBalance(t.clientId, t.value, balances); err != nil {
		return 0, err
	}
	transfer := state.NewTransfer(t.clientId, t.toClientId, t.value)
	if err := balances.AddTransfer(transfer); err != nil {
		return 0, fmt.Errorf("adding transfer to allocation pool: %v", err)
	}

	return t.value, nil
}

func NewTokenTransfer(value currency.Coin, clientId, toClientId string, isMint bool) *Transfer {
	return &Transfer{
		value:      value,
		clientId:   clientId,
		toClientId: toClientId,
		isMint:     isMint,
	}
}

func (sab *storageAllocationBase) moveToChallengePool(
	cp *challengePool,
	value currency.Coin,
) error {
	if cp == nil {
		return errors.New("invalid challenge pool")
	}
	if value > sab.WritePool {
		return fmt.Errorf("insufficient funds %v in write pool to pay %v", sab.WritePool, value)
	}

	if balance, err := currency.AddCoin(cp.Balance, value); err != nil {
		return err
	} else {
		cp.Balance = balance
	}
	if writePool, err := currency.MinusCoin(sab.WritePool, value); err != nil {
		return err
	} else {
		sab.WritePool = writePool
	}

	return nil
}

func (sab *storageAllocationBase) moveFromChallengePool(
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
	if writePool, err := currency.AddCoin(sab.WritePool, value); err != nil {
		return err
	} else {
		sab.WritePool = writePool
	}
	return nil
}

func (sab *storageAllocationBase) payChallengePoolPassPayments(sps []*stakePool, balances chainstate.StateContextI, cp *challengePool, passRates []float64, conf *Config, sc *StorageSmartContract, now common.Timestamp) error {
	var passPayments currency.Coin

	for i, d := range sab.BlobberAllocs {
		blobberPassPayment, _, err := d.payChallengePoolPassPayments(sab, sps[i], cp, passRates[i], balances, conf, now, sc)
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

	sab.MovedBack, err = currency.AddCoin(sab.MovedBack, cp.Balance)
	if err != nil {
		return err
	}

	err = sab.moveFromChallengePool(cp, cp.Balance)
	if err != nil {
		return fmt.Errorf("failed to move challenge pool back to write pool: %v", err)
	}

	if err = cp.save(sc.ID, sab, balances); err != nil {
		return fmt.Errorf("failed to save challenge pool: %v", err)
	}

	i, err := prevBal.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert balance: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, cp.ID, event.ChallengePoolLock{
		Client:       sab.Owner,
		AllocationId: sab.ID,
		Amount:       i,
	})

	return nil
}

func (sab *storageAllocationBase) payChallengePoolPassPaymentsToRemoveBlobber(sp *stakePool, balances chainstate.StateContextI, cp *challengePool, passRate float64, conf *Config, sc *StorageSmartContract, ba *BlobberAllocation, now common.Timestamp) error {
	passPayment, penaltyPayment, err := ba.payChallengePoolPassPayments(sab, sp, cp, passRate, balances, conf, now, sc)
	if err != nil {
		return fmt.Errorf("error paying challenge pool pass payments: %v", err)
	}

	balance, err := currency.MinusCoin(cp.Balance, passPayment)
	if err != nil {
		return err
	}
	cp.Balance = balance

	sab.MovedBack, err = currency.AddCoin(sab.MovedBack, ba.ChallengePoolIntegralValue+penaltyPayment)
	if err != nil {
		return err
	}

	err = sab.moveFromChallengePool(cp, ba.ChallengePoolIntegralValue+penaltyPayment)
	if err != nil {
		return fmt.Errorf("failed to move challenge pool back to write pool: %v", err)
	}

	if err = cp.save(sc.ID, sab, balances); err != nil {
		return fmt.Errorf("failed to save challenge pool: %v", err)
	}

	fromChallengePool := ba.ChallengePoolIntegralValue + passPayment + penaltyPayment
	i, err := fromChallengePool.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert balance: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagFromChallengePool, cp.ID, event.ChallengePoolLock{
		Client:       sab.Owner,
		AllocationId: sab.ID,
		Amount:       i,
	})

	return nil
}

// Cancellation charge
func (sab *storageAllocationBase) payCancellationCharge(sps []*stakePool, balances chainstate.StateContextI, passRates []float64, conf *Config, sc *StorageSmartContract, t *transaction.Transaction) error {
	cancellationCharge, err := sab.cancellationCharge(conf.CancellationCharge)
	if err != nil {
		return fmt.Errorf("failed to get cancellation charge: %v", err)
	}

	usedWritePool := sab.MovedToChallenge - sab.MovedBack

	if usedWritePool < cancellationCharge {
		cancellationCharge = cancellationCharge - usedWritePool

		if sab.WritePool < cancellationCharge {
			cancellationCharge = sab.WritePool
		}
	} else {
		return nil
	}

	totalWritePrice := currency.Coin(0)
	for _, ba := range sab.BlobberAllocs {
		totalWritePrice, err = currency.AddCoin(totalWritePrice, ba.Terms.WritePrice)
		if err != nil {
			return fmt.Errorf("failed to add write price: %v", err)
		}
	}

	totalCancellationChargePaid := currency.Coin(0)

	for i, ba := range sab.BlobberAllocs {
		blobberCancellationChargePaid, err := ba.payCancellationCharge(sab, sps[i], balances, sc, passRates[i], totalWritePrice, cancellationCharge)
		if err != nil {
			return fmt.Errorf("1 error paying cancellation charge: %v", err)
		}

		totalCancellationChargePaid, err = currency.AddCoin(totalCancellationChargePaid, blobberCancellationChargePaid)
		if err != nil {
			return fmt.Errorf("error adding blobber cancellation charge paid to totalCancellationChargePaid: %v", err)
		}
	}

	sab.WritePool, err = currency.MinusCoin(sab.WritePool, totalCancellationChargePaid)
	if err != nil {
		return fmt.Errorf("failed to deduct cancellation charges from write pool: %v", err)
	}

	i, err := totalCancellationChargePaid.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert deduction from write pool to int64: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUnlockWritePool, sab.ID, event.WritePoolLock{
		Client:       t.ClientID,
		AllocationId: sab.ID,
		Amount:       i,
	})

	return nil
}

func (sab *storageAllocationBase) payCancellationChargeToRemoveBlobber(sp *stakePool, balances chainstate.StateContextI, passRate float64, conf *Config, sc *StorageSmartContract, clientID string, ba *BlobberAllocation) error {
	cancellationCharge, err := sab.cancellationCharge(conf.CancellationCharge)
	if err != nil {
		return fmt.Errorf("failed to get cancellation charge: %v", err)
	}

	usedWritePool := sab.MovedToChallenge - sab.MovedBack

	if usedWritePool < cancellationCharge {
		cancellationCharge = cancellationCharge - usedWritePool

		if sab.WritePool < cancellationCharge {
			cancellationCharge = sab.WritePool
		}
	} else {
		return nil
	}

	totalWritePrice := currency.Coin(0)
	for _, ba := range sab.BlobberAllocs {
		totalWritePrice, err = currency.AddCoin(totalWritePrice, ba.Terms.WritePrice)
		if err != nil {
			return fmt.Errorf("failed to add write price: %v", err)
		}
	}

	totalCancellationChargePaid, err := ba.payCancellationCharge(sab, sp, balances, sc, passRate, totalWritePrice, cancellationCharge)
	if err != nil {
		return fmt.Errorf("2 error paying cancellation charge: %v", err)
	}

	sab.WritePool, err = currency.MinusCoin(sab.WritePool, totalCancellationChargePaid)
	if err != nil {
		return fmt.Errorf("failed to deduct cancellation charges from write pool: %v", err)
	}

	i, err := totalCancellationChargePaid.Int64()
	if err != nil {
		return fmt.Errorf("failed to convert deduction from write pool to int64: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagUnlockWritePool, sab.ID, event.WritePoolLock{
		Client:       clientID,
		AllocationId: sab.ID,
		Amount:       i,
	})

	return nil
}

func (sab *storageAllocationBase) isActive(
	blobber *StorageNode,
	totalStakePoolBalance, spOffersTotal currency.Coin,
	stakedCapacity int64,
	conf *Config,
	now common.Timestamp,
) error {
	bb := blobber.mustBase()
	active, reason := bb.Provider.IsActive(now, conf.HealthCheckPeriod)
	if !active {
		return fmt.Errorf("blobber %s is not active, %s", bb.ID, reason)
	}

	if bb.NotAvailable {
		return fmt.Errorf("blobber %s is not currently available for new allocations", bb.ID)
	}

	// filter by read price
	if !sab.ReadPriceRange.isMatch(bb.Terms.ReadPrice) {
		return fmt.Errorf("read price range %v does not match blobber %s read price %v",
			sab.ReadPriceRange, bb.ID, bb.Terms.ReadPrice)
	}
	// filter by write price
	if !sab.WritePriceRange.isMatch(bb.Terms.WritePrice) {
		return fmt.Errorf("write price range %v does not match blobber %s write price %v",
			sab.WritePriceRange, bb.ID, bb.Terms.WritePrice)
	}

	blobberSize := sab.bSize()
	// filter by blobber's capacity left
	if bb.Capacity-bb.Allocated < blobberSize {
		return fmt.Errorf("blobber %s free capacity %v insufficient, wanted %v",
			bb.ID, bb.Capacity-bb.Allocated, blobberSize)
	} else if stakedCapacity-bb.Allocated < blobberSize {
		return fmt.Errorf("blobber %s free staked capacity %v insufficient, wanted %v",
			bb.ID, stakedCapacity-bb.Allocated, blobberSize)
	}

	unallocCapacity, err := unallocatedCapacity(bb.Terms.WritePrice, totalStakePoolBalance, spOffersTotal)
	if err != nil {
		return fmt.Errorf("failed to get unallocated capacity: %v", err)
	}

	if bb.Terms.WritePrice > 0 && unallocCapacity < blobberSize {
		return fmt.Errorf("blobber %v staked capacity %v is insufficient, wanted %v",
			bb.ID, unallocCapacity, blobberSize)
	}

	return nil
}

//nolint:unused
func (ba *BlobberAllocation) cost() (currency.Coin, error) {
	cost, err := currency.MultFloat64(ba.Terms.WritePrice, sizeInGB(ba.Size))
	if err != nil {
		return 0, err
	}
	return cost, nil
}

func (sab *storageAllocationBase) cancellationCharge(cancellationFraction float64) (currency.Coin, error) {
	cost, err := sab.cost()
	if err != nil {
		return 0, err
	}
	return currency.MultFloat64(cost, cancellationFraction)
}

func (sa *storageAllocationBase) requiredTokensForUpdateAllocation(balances cstate.StateContextI, cpBalance currency.Coin, extend, isEnterprise bool, now common.Timestamp) (currency.Coin, error) {
	var (
		costOfAllocAfterUpdate currency.Coin
		tokensRequiredToLock   currency.Coin
		err                    error
	)

	if isEnterprise || extend {
		costOfAllocAfterUpdate, err = sa.cost()
		if err != nil {
			return 0, fmt.Errorf("failed to get allocation cost: %v", err)
		}
	} else {
		costOfAllocAfterUpdate, err = sa.costForRDTU(now)
		if err != nil {
			return 0, fmt.Errorf("failed to get allocation cost: %v", err)
		}
	}

	totalWritePool := sa.WritePool + cpBalance

	if totalWritePool < costOfAllocAfterUpdate {
		tokensRequiredToLock = costOfAllocAfterUpdate - totalWritePool
	} else {
		tokensRequiredToLock = 0
	}

	var costOfUnusedAlloc currency.Coin

	actErr := chainstate.WithActivation(balances, "hercules", func() error {
		return nil
	}, func() error {
		costOfUnusedAlloc, err = sa.unUsedAllocCost()
		if err != nil {
			return fmt.Errorf("failed to get unused allocation cost: %v", err)
		}
		if costOfUnusedAlloc > sa.WritePool && costOfUnusedAlloc-sa.WritePool > tokensRequiredToLock {
			tokensRequiredToLock = costOfUnusedAlloc - sa.WritePool
		}
		return nil
	})
	if actErr != nil {
		return 0, actErr
	}

	logging.Logger.Info("requiredTokensForUpdateAllocation",
		zap.Any("costOfAllocAfterUpdate", costOfAllocAfterUpdate),
		zap.Any("totalWritePool", totalWritePool),
		zap.Any("tokensRequiredToLock", tokensRequiredToLock),
		zap.Any("extend", extend),
		zap.Any("isEnterprise", isEnterprise),
		zap.Any("sa", sa),
		zap.Any("cpBalance", cpBalance),
		zap.Any("now", now),
		zap.Any("costOfUnusedAlloc", costOfUnusedAlloc),
	)

	return tokensRequiredToLock, nil
}

func (sab *storageAllocationBase) bSize() int64 {
	return bSize(sab.Size, sab.DataShards)
}

func bSize(size int64, dataShards int) int64 {
	return int64(math.Ceil(float64(size) / float64(dataShards)))
}

func (sab *storageAllocationBase) replaceBlobber(blobberID string, sc *StorageSmartContract, balances chainstate.StateContextI, txn *transaction.Transaction, addedBlobberAllocation *BlobberAllocation, now common.Timestamp, isEnterpriseBlobber bool) error {
	_, ok := sab.BlobberAllocsMap[blobberID]
	if !ok {
		return fmt.Errorf("cannot find blobber %s in allocation", blobberID)
	}
	delete(sab.BlobberAllocsMap, blobberID)

	conf, err := getConfig(balances)
	if err != nil {
		return common.NewError("can't get config", err.Error())
	}

	for i, d := range sab.BlobberAllocs {
		if d.BlobberID == blobberID {
			blobberIsKilled := false
			blobber, e := sc.getBlobber(d.BlobberID, balances)
			if e != nil {
				return e
			}

			bb := blobber.mustBase()

			if bb.IsKilled() || bb.IsShutDown() {
				blobberIsKilled = true

				var cp *challengePool
				cp, e = sc.getChallengePool(sab.ID, balances)
				if e != nil {
					e = fmt.Errorf("could not get challenge pool of alloc: %s, err: %v", sab.ID, e)

					if demeterActErr := cstate.WithActivation(balances, "demeter", func() (e error) { return }, func() error {
						return e
					}); demeterActErr != nil {
						return demeterActErr
					}
				}

				e = sab.moveFromChallengePool(cp, d.ChallengePoolIntegralValue)
				if e != nil {
					return fmt.Errorf("failed to move challenge pool back to write pool: %v", e)
				}
			}

			if d.Stats.UsedSize > 0 {
				if err := removeAllocationFromBlobberPartitions(balances, d.BlobberID, d.AllocationID); err != nil {
					return err
				}
			}

			if blobberIsKilled {
				sab.BlobberAllocs[i] = addedBlobberAllocation
				sab.BlobberAllocsMap[addedBlobberAllocation.BlobberID] = addedBlobberAllocation
				break
			}

			passRate, err := d.removeBlobberPassRates(sab, conf.MaxChallengeCompletionRounds, balances, sc)
			if err != nil {
				logging.Logger.Info("error removing blobber pass rates",
					zap.Any("allocation", sab.ID),
					zap.Any("blobber", d.BlobberID),
					zap.Error(err))
				return fmt.Errorf("error removing blobber pass rates: %v", err)
			}

			sp, err := sc.getStakePool(spenum.Blobber, d.BlobberID, balances)
			if err != nil {
				return common.NewError("remove_blobber_failed",
					"can't get stake pool of "+d.BlobberID+": "+err.Error())
			}
			if err := sp.reduceOffer(balances, d.Offer()); err != nil {
				return common.NewError("remove_blobber_failed",
					"error removing offer: "+err.Error())
			}

			actErr := cstate.WithActivation(balances, "demeter", func() (e error) {
				return sp.Save(spenum.Blobber, d.BlobberID, balances)
			}, func() (e error) {
				return nil
			})
			if actErr != nil {
				return actErr
			}

			if isEnterpriseBlobber {
				cost, err := sab.payCostForDtuForReplaceEnterpriseBlobber(txn, conf, sp, blobberID, balances)
				logging.Logger.Info("payCostForDtuForReplaceEnterpriseBlobber", zap.Any("cost", cost), zap.Any("err", err))
				if err != nil {
					return err
				}
			} else {
				cp, err := sc.getChallengePool(sab.ID, balances)
				if err != nil {
					return fmt.Errorf("could not get challenge pool of alloc: %s, err: %v", sab.ID, err)
				}

				if err = sab.payChallengePoolPassPaymentsToRemoveBlobber(sp, balances, cp, passRate, conf, sc, d, now); err != nil {
					return fmt.Errorf("error paying challenge pool pass payments: %v", err)
				}

				if err = sab.payCancellationChargeToRemoveBlobber(sp, balances, passRate, conf, sc, txn.ClientID, d); err != nil {
					return fmt.Errorf("3 error paying cancellation charge: %v", err)
				}
			}

			actErr = cstate.WithActivation(balances, "demeter", func() (e error) { return },
				func() (e error) {
					return sp.Save(spenum.Blobber, d.BlobberID, balances)
				})
			if actErr != nil {
				return actErr
			}

			//nolint:errcheck
			blobber.mustUpdateBase(func(b *storageNodeBase) error {
				b.SavedData += -d.Stats.UsedSize
				b.Allocated += -d.Size
				return nil
			})

			// Saving removed blobber to mpt here
			_, err = balances.InsertTrieNode(blobber.GetKey(), blobber)
			if err != nil {
				return common.NewError("fini_alloc_failed",
					"saving blobber "+d.BlobberID+": "+err.Error())
			}

			// Update saved data on events_db
			emitUpdateBlobberAllocatedSavedHealth(blobber, balances)

			sab.Stats.UsedSize += -d.Stats.UsedSize
			sab.BlobberAllocs[i] = addedBlobberAllocation
			sab.BlobberAllocsMap[addedBlobberAllocation.BlobberID] = addedBlobberAllocation
			break
		}
	}

	return nil
}

func replaceBlobber(
	sa *storageAllocationBase,
	blobbers []*StorageNode,
	blobberID string,
	balances cstate.StateContextI,
	sc *StorageSmartContract,
	txn *transaction.Transaction,
	addedBlobber *StorageNode, addedBlobberAllocation *BlobberAllocation, now common.Timestamp, isEnterpriseBlobber bool) ([]*StorageNode, error) {
	if err := sa.replaceBlobber(blobberID, sc, balances, txn, addedBlobberAllocation, now, isEnterpriseBlobber); err != nil {
		return nil, err
	}

	var found bool
	for i, d := range blobbers {
		dd := d.mustBase()
		if dd.ID == blobberID {
			blobbers[i] = addedBlobber
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("cannot find blobber %s in allocation", blobberID)
	}

	return blobbers, nil
}

func (sab *storageAllocationBase) changeBlobbers(
	conf *Config,
	blobbers []*StorageNode,
	addId, authTicket, removeId string,
	now common.Timestamp,
	balances cstate.StateContextI,
	sc *StorageSmartContract,
	txn *transaction.Transaction,
	isEnterpriseAlloc bool,
	storageVersion int,
	authRoundExpiry int64,
) ([]*StorageNode, error) {
	var err error

	_, found := sab.BlobberAllocsMap[addId]
	if found {
		return nil, fmt.Errorf("allocation already has blobber %s", addId)
	}

	addedBlobber, err := getBlobber(addId, balances)
	if err != nil {
		return nil, fmt.Errorf("can't get blobber %s to add : %v", addId, err)
	}

	bb := addedBlobber.mustBase()

	var sp *stakePool
	if sp, err = getStakePool(spenum.Blobber, bb.ID, balances); err != nil {
		return nil, fmt.Errorf("can't get blobber's stake pool: %v", err)
	}
	staked, err := sp.stake()
	if err != nil {
		return nil, err
	}

	stakedCapacity, err := sp.stakedCapacity(bb.Terms.WritePrice)
	if err != nil {
		return nil, err
	}

	if err := sab.isActive(addedBlobber, staked, sp.TotalOffers, stakedCapacity, conf, now); err != nil {
		return nil, err
	}

	if actErr := cstate.WithActivation(balances, "electra",
		func() error {
			return addedBlobber.Update(&storageNodeV2{}, func(e entitywrapper.EntityI) error {
				b := e.(*storageNodeV2)

				if b.IsRestricted != nil && *b.IsRestricted {
					success, err := verifyBlobberAuthTicket(balances, sab.Owner, authTicket, b.PublicKey, authRoundExpiry)
					if err != nil {
						return fmt.Errorf("blobber %s auth ticket verification failed: %v", b.ID, err.Error())
					} else if !success {
						return fmt.Errorf("blobber %s auth ticket verification failed", b.ID)
					}
				}

				return nil
			})
		}, func() error {
			return cstate.WithActivation(balances, "hercules",
				func() error {
					return addedBlobber.Update(&storageNodeV3{}, func(e entitywrapper.EntityI) error {
						b := e.(*storageNodeV3)

						if isEnterpriseAlloc {
							if b.IsEnterprise == nil || !*b.IsEnterprise {
								return fmt.Errorf("blobber %s is not enterprise", b.ID)
							}
						}

						if (b.IsEnterprise != nil && *b.IsEnterprise) || (b.IsRestricted != nil && *b.IsRestricted) {
							success, err := verifyBlobberAuthTicket(balances, sab.Owner, authTicket, b.PublicKey, authRoundExpiry)
							if err != nil {
								return fmt.Errorf("blobber %s auth ticket verification failed: %v", b.ID, err.Error())
							} else if !success {
								return fmt.Errorf("blobber %s auth ticket verification failed", b.ID)
							}
						}

						return nil
					})
				}, func() error {
					return addedBlobber.Update(&storageNodeV4{}, func(e entitywrapper.EntityI) error {
						b := e.(*storageNodeV4)

						if isEnterpriseAlloc {
							if b.IsEnterprise == nil || !*b.IsEnterprise {
								return fmt.Errorf("blobber %s is not enterprise", b.ID)
							}
						}

						if (b.IsEnterprise != nil && *b.IsEnterprise) || (b.IsRestricted != nil && *b.IsRestricted) {
							success, err := verifyBlobberAuthTicket(balances, sab.Owner, authTicket, b.PublicKey, authRoundExpiry)
							if err != nil {
								return fmt.Errorf("blobber %s auth ticket verification failed: %v", b.ID, err.Error())
							} else if !success {
								return fmt.Errorf("blobber %s auth ticket verification failed", b.ID)
							}
						}

						if !isEnterpriseAlloc && b.StorageVersion != nil && storageVersion == 1 && *b.StorageVersion != 1 {
							return fmt.Errorf("blobber version %s is not compatible with v2 allocation", b.ID)
						}

						return nil
					})
				})
		}); actErr != nil {
		return nil, actErr
	}

	//nolint:errcheck
	addedBlobber.mustUpdateBase(func(b *storageNodeBase) error {
		b.Allocated += sab.bSize() // Why increase allocation then check if the free capacity is enough?
		return nil
	})
	afterSize := sab.bSize()

	ba := newBlobberAllocation(afterSize, sab, addedBlobber.mustBase(), conf, now)

	if len(removeId) > 0 {
		if blobbers, err = replaceBlobber(sab, blobbers, removeId, balances, sc, txn, addedBlobber, ba, now, isEnterpriseAlloc); err != nil {
			return nil, err
		}
	} else {
		// If we are not removing a blobber, then the number of shards must increase.
		sab.ParityShards++

		blobbers = append(blobbers, addedBlobber)
		sab.BlobberAllocs = append(sab.BlobberAllocs, ba)
	}

	sab.BlobberAllocsMap[addId] = ba

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

//nolint:unused
type filterBlobberFunc func(blobber *StorageNode) (kick bool, err error)

type filterValidatorFunc func(validator *ValidationNode) (kick bool, err error)

//nolint:unused
func (sab *storageAllocationBase) filterBlobbers(list []*StorageNode,
	creationDate common.Timestamp, bsize int64, filters ...filterBlobberFunc) (
	filtered []*StorageNode, err error) {

	var (
		i int
	)

List:
	for _, b := range list {
		// filter by read price
		bb := b.mustBase()
		if !sab.ReadPriceRange.isMatch(bb.Terms.ReadPrice) {
			continue
		}
		// filter by write price
		if !sab.WritePriceRange.isMatch(bb.Terms.WritePrice) {
			continue
		}
		// filter by blobber's capacity left
		if bb.Capacity-bb.Allocated < bsize {
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
func (sab *storageAllocationBase) validateEachBlobber(
	request newAllocationRequest,
	balances cstate.StateContextI,
	blobbers []*storageNodeResponse,
	blobberAuthTickets []string,
	creationDate common.Timestamp,
	conf *Config,
) ([]*StorageNode, []string) {
	var (
		errs     = make([]string, 0, len(blobbers))
		filtered = make([]*StorageNode, 0, len(blobbers))
	)
	for i, b := range blobbers {
		sn := StorageNode{}

		beforeHardfork := func() error {
			return cstate.WithActivation(balances, "electra", func() error {
				snr := storageNodeResponseToStorageNodeV2(*b)
				sn.SetEntity(snr)
				return nil
			}, func() error {
				if request.IsEnterprise && !b.IsEnterprise {
					return fmt.Errorf("blobber %s is not enterprise", b.ID)
				} else if !request.IsEnterprise && b.IsEnterprise {
					return fmt.Errorf("blobber %s is enterprise", b.ID)
				}

				snr := storageNodeResponseToStorageNodeV3(*b)
				sn.SetEntity(snr)
				return nil
			})
		}

		herculesHardfork := func() error {
			if request.IsEnterprise && !b.IsEnterprise {
				return fmt.Errorf("blobber %s is not enterprise", b.ID)
			} else if !request.IsEnterprise && b.IsEnterprise {
				return fmt.Errorf("blobber %s is enterprise", b.ID)
			}

			if !request.IsEnterprise && request.StorageVersion == 1 && b.StorageVersion != 1 {
				return fmt.Errorf("blobber version %s is not compatible with v2 allocation", b.ID)
			}

			snr := storageNodeResponseToStorageNodeV4(*b)
			sn.SetEntity(snr)
			return nil
		}

		actErr := cstate.WithActivation(balances, "hercules", beforeHardfork, herculesHardfork)
		if actErr != nil {
			errs = append(errs, actErr.Error())
			continue
		}

		if (b.IsEnterprise) || (b.IsRestricted) {
			success, err := verifyBlobberAuthTicket(balances, sab.Owner, blobberAuthTickets[i], sn.mustBase().PublicKey, request.AuthRoundExpiry)
			if err != nil {
				errs = append(errs, fmt.Sprintf("blobber %s auth ticket verification failed: %v", b.ID, err.Error()))
				continue
			} else if !success {
				errs = append(errs, fmt.Sprintf("blobber %s auth ticket verification failed", b.ID))
				continue
			}
		}

		err := sab.isActive(&sn, b.TotalStake, b.TotalOffers, b.StakedCapacity, conf, creationDate)
		if err != nil {
			logging.Logger.Debug("error validating blobber", zap.String("id", b.ID), zap.Error(err))
			errs = append(errs, err.Error())
			continue
		}

		filtered = append(filtered, &sn)
	}
	return filtered, errs
}

func verifyBlobberAuthTicket(balances cstate.StateContextI, clientID, authTicket, publicKey string, roundExpiry int64) (bool, error) {
	if authTicket == "" {
		return false, common.NewError("invalid_auth_ticket", "empty auth ticket")
	}

	var payload string

	if actErr := cstate.WithActivation(balances, "hermes", func() error {
		payload = clientID
		return nil
	}, func() error {
		if roundExpiry < balances.GetBlock().Round {
			return common.NewError("auth_ticket_expired", "auth ticket expired, current round: "+strconv.FormatInt(balances.GetBlock().Round, 10)+", expiry round: "+strconv.FormatInt(roundExpiry, 10))
		}
		payload = encryption.Hash(fmt.Sprintf("%s_%d", clientID, roundExpiry))

		return nil
	}); actErr != nil {
		return false, actErr
	}

	logging.Logger.Info("verifyBlobberAuthTicket", zap.String("clientID", clientID), zap.String("authTicket", authTicket), zap.String("publicKey", publicKey), zap.Any("roundExpiry", roundExpiry))

	signatureScheme := balances.GetSignatureScheme()
	if err := signatureScheme.SetPublicKey(publicKey); err != nil {
		return false, err
	}
	success, err := signatureScheme.Verify(authTicket, payload)
	return success, err
}

// Until returns allocation expiration.
func (sab *storageAllocationBase) Until(duration time.Duration) common.Timestamp {
	return sab.Expiration + toSeconds(duration)
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

func (sab *storageAllocationBase) challengePoolChanges(odr, ndr common.Timestamp, timeUnit time.Duration,
	oterms []Terms) (values []ChallengePoolChanges, err error) {

	// odr -- old duration remaining
	// ndr -- new duration remaining

	// in time units, instead of common.Timestamp
	odrtu, err := sab.durationInTimeUnits(odr, timeUnit)
	if err != nil {
		return nil, fmt.Errorf("failed to get old challenge pool duration: %v", err)
	}

	ndrtu, err := sab.durationInTimeUnits(ndr, timeUnit)
	if err != nil {
		return nil, fmt.Errorf("failed to get new challenge pool duration: %v", err)
	}

	values = make([]ChallengePoolChanges, 0, len(sab.BlobberAllocs))

	for i, d := range sab.BlobberAllocs {
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

func (sab *storageAllocationBase) IsValidFinalizer(id string) bool {
	if sab.Owner == id {
		return true // finalizing by owner
	}
	for _, d := range sab.BlobberAllocs {
		if d.BlobberID == id {
			return true // one of blobbers
		}
	}
	return false // unknown
}

// removeExpiredChallenges removes all expired challenges from the allocation,
// return the expired challenge ids per blobber (maps blobber id to its expiredIDs), or error if any.
// the expired challenge ids could be used to delete the challenge node from MPT when needed
func (sab *storageAllocationBase) removeExpiredChallenges(
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

		ba, ok := sab.BlobberAllocsMap[oc.BlobberID]
		if ok {
			ba.Stats.FailedChallenges++
			ba.Stats.OpenChallenges--
			sab.Stats.FailedChallenges++
			sab.Stats.OpenChallenges--

			if ba.LatestFinalizedChallCreatedAt < oc.CreatedAt {
				ba.LatestFinalizedChallCreatedAt = oc.CreatedAt
			}

			err := emitUpdateChallenge(&StorageChallenge{
				ID:           oc.ID,
				AllocationID: sab.ID,
				BlobberID:    oc.BlobberID,
			}, false, ChallengeRespondedLate, balances, sab.Stats)

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
func (sab *storageAllocationBase) removeOldChallenges(
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

		ba, ok := sab.BlobberAllocsMap[oc.BlobberID]
		if ok {
			ba.Stats.FailedChallenges++
			ba.Stats.OpenChallenges--
			sab.Stats.FailedChallenges++
			sab.Stats.OpenChallenges--

			if ba.LatestFinalizedChallCreatedAt < oc.CreatedAt {
				ba.LatestFinalizedChallCreatedAt = oc.CreatedAt
			}

			err := emitUpdateChallenge(&StorageChallenge{
				ID:           oc.ID,
				AllocationID: sab.ID,
				BlobberID:    oc.BlobberID,
			}, false, ChallengeOldRemoved, balances, sab.Stats)

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

// Clone implements statecache.Value interface
func (sa *StorageAllocation) Clone() statecache.Value {
	// clone := *sa
	// clone.Stats = &StorageAllocationStats{}
	// *clone.Stats = *sa.Stats

	// clone.PreferredBlobbers = make([]string, len(sa.PreferredBlobbers))
	// copy(clone.PreferredBlobbers, sa.PreferredBlobbers)

	// clone.BlobberAllocs = make([]*BlobberAllocation, len(sa.BlobberAllocs))
	// clone.BlobberAllocsMap = make(map[string]*BlobberAllocation, len(sa.BlobberAllocsMap))
	// for i, sba := range sa.BlobberAllocs {
	// 	ba := &BlobberAllocation{}
	// 	*ba = *sba
	// 	if sba.LastWriteMarker != nil {
	// 		ba.LastWriteMarker = &WriteMarker{}
	// 		*ba.LastWriteMarker = *sba.LastWriteMarker
	// 	}

	// 	if sba.Stats != nil {
	// 		ba.Stats = &StorageAllocationStats{}
	// 		*ba.Stats = *sba.Stats
	// 	}

	// 	clone.BlobberAllocs[i] = ba
	// 	clone.BlobberAllocsMap[ba.BlobberID] = ba
	// }

	// return &clone
	v, err := sa.MarshalMsg(nil)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal StorageAllocation: %v", err))
	}

	na := &StorageAllocation{}
	_, err = na.UnmarshalMsg(v)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal StorageAllocation: %v", err))
	}

	return na
}

// CopyFrom implements statecache.Value interface
func (sa *StorageAllocation) CopyFrom(v interface{}) bool {
	sav, ok := v.(*StorageAllocation)
	if !ok {
		return false
	}

	clone := sav.Clone().(*StorageAllocation)

	*sa = *clone
	return true
}

func (sab *storageAllocationBase) RefreshAllocationUsedSize() {
	totalBlobberAllocationUsedSize := int64(0)
	for _, ba := range sab.BlobberAllocs {
		totalBlobberAllocationUsedSize += ba.Stats.UsedSize
	}

	sab.Stats.UsedSize = (totalBlobberAllocationUsedSize * int64(sab.DataShards)) / (int64(sab.DataShards + sab.ParityShards))
}

type BlobberCloseConnection struct {
	AllocationRoot     string       `json:"allocation_root"`
	PrevAllocationRoot string       `json:"prev_allocation_root"`
	WriteMarker        *WriteMarker `json:"write_marker"`
	ChainData          []byte       `json:"chain_data"`
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

	return bc.WriteMarker.Verify(bc.AllocationRoot, bc.PrevAllocationRoot)
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
