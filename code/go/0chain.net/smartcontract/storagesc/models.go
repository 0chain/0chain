package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var (
	ALL_BLOBBERS_KEY    = datastore.Key(ADDRESS + encryption.Hash("all_blobbers"))
	ALL_VALIDATORS_KEY  = datastore.Key(ADDRESS + encryption.Hash("all_validators"))
	ALL_ALLOCATIONS_KEY = datastore.Key(ADDRESS + encryption.Hash("all_allocations"))
	STORAGE_STATS_KEY   = datastore.Key(ADDRESS + encryption.Hash("all_storage"))
)

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
	List sortedList
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
	BlobberID                string                       `json:"blobber_id"`
	Challenges               []*StorageChallenge          `json:"challenges"`
	ChallengeMap             map[string]*StorageChallenge `json:"-"`
	LatestCompletedChallenge *StorageChallenge            `json:"lastest_completed_challenge"`
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
	sn.ChallengeMap = make(map[string]*StorageChallenge)
	for _, challenge := range sn.Challenges {
		sn.ChallengeMap[challenge.ID] = challenge
	}
	return nil
}

func (sn *BlobberChallenge) addChallenge(challenge *StorageChallenge) bool {
	if sn.Challenges == nil {
		sn.Challenges = make([]*StorageChallenge, 0)
		sn.ChallengeMap = make(map[string]*StorageChallenge)
	}
	if _, ok := sn.ChallengeMap[challenge.ID]; !ok {
		if len(sn.Challenges) > 0 {
			lastChallenge := sn.Challenges[len(sn.Challenges)-1]
			challenge.PrevID = lastChallenge.ID
		} else if sn.LatestCompletedChallenge != nil {
			challenge.PrevID = sn.LatestCompletedChallenge.ID
		}
		sn.Challenges = append(sn.Challenges, challenge)
		sn.ChallengeMap[challenge.ID] = challenge
		return true
	}
	return false
}

type StorageChallenge struct {
	Created        common.Timestamp   `json:"created"`
	ID             string             `json:"id"`
	PrevID         string             `json:"prev_id"`
	Validators     []*ValidationNode  `json:"validators"`
	RandomNumber   int64              `json:"seed"`
	AllocationID   string             `json:"allocation_id"`
	Blobber        *StorageNode       `json:"blobber"`
	AllocationRoot string             `json:"allocation_root"`
	Response       *ChallengeResponse `json:"challenge_response,omitempty"`
}

type ValidationNode struct {
	ID        string `json:"id"`
	BaseURL   string `json:"url"`
	PublicKey string `json:"-"`
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
	// ReadPrice is price for reading. Token / GB.
	ReadPrice state.Balance `json:"read_price"`
	// WritePrice is price for reading. Token / GB. Also,
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

// validate a received terms
func (t *Terms) validate(conf *scConfig) (err error) {
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
	return // nil
}

// StorageNode represents Blobber configurations.
type StorageNode struct {
	ID              string           `json:"id"`
	BaseURL         string           `json:"url"`
	Terms           Terms            `json:"terms"`    // terms
	Capacity        int64            `json:"capacity"` // total blobber capacity
	Used            int64            `json:"used"`     // allocated capacity
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	PublicKey       string           `json:"-"`
}

// validate the blobber configurations
func (sn *StorageNode) validate(conf *scConfig) (err error) {
	if err = sn.Terms.validate(conf); err != nil {
		return
	}
	if sn.Capacity <= conf.MinBlobberCapacity {
		return errors.New("insufficient blobber capacity")
	}
	return
}

// stake required for the blobber with current capacity and
// current write price (i.e. with current terms)
func (sn *StorageNode) stake() state.Balance {
	return state.Balance(sizeInGB(sn.Capacity) * float64(sn.Terms.WritePrice))
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

type StorageNodes struct {
	Nodes sortedBlobbers
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
	BlobberID       string                  `json:"blobber_id"`
	AllocationID    string                  `json:"allocation_id"`
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
	// ChallengeReward of the blobber.
	ChallengeReward state.Balance `json:"challenge_reward"`
	// FinalReward is number of tokens moved to the blobber on finalization.
	// It can be greater then zero, if user didn't spent the min lock demand
	// during the allocation.
	FinalReward state.Balance `json:"final_reward"`
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

	DataShards        int                           `json:"data_shards"`
	ParityShards      int                           `json:"parity_shards"`
	Size              int64                         `json:"size"`
	Expiration        common.Timestamp              `json:"expiration_date"`
	Blobbers          []*StorageNode                `json:"blobbers"`
	Owner             string                        `json:"owner_id"`
	OwnerPublicKey    string                        `json:"owner_public_key"`
	Stats             *StorageAllocationStats       `json:"stats"`
	PreferredBlobbers []string                      `json:"preferred_blobbers"`
	BlobberDetails    []*BlobberAllocation          `json:"blobber_details"`
	BlobberMap        map[string]*BlobberAllocation `json:"-"`
	ReadPriceRange    PriceRange                    `json:"read_price_range"`
	WritePriceRange   PriceRange                    `json:"write_price_range"`
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
	Cancelled bool `json:"canceled,omitempty"`
	// UsedSize used to calculate blobber reward ratio.
	UsedSize int64 `json:"-"`

	// MovedToChallenge is number of tokens moved to challenge pool.
	MovedToChallenge state.Balance `json:"moved_to_challenge"`
	// MovedToBlobers is number of tokens moved to blobbers on finalization.
	MovedToBlobers state.Balance `json:"moved_to_blobbers"`
	// MovedBack is number of tokens moved from challenge pool to
	// related write pool (the Back) if a data has deleted.
	MovedBack state.Balance `json:"moved_back"`
	// MovedToValidators is total number of tokens moved to validators
	// of the allocation.
	MovedToValidators state.Balance `json:"moved_to_validators"`
}

// minLockDemandLeft returns number of tokens required as min_lock_demand;
// if a blobber receive write marker, then some token moves to related
// challenge pool and 'Spent' of this blobber is increased; thus, the 'Spent'
// reduces the min_lock_demand left of this blobber; but, if a malfunctioning
// client doesn't send a data to a blobber (or blobbers) then this blobbers
// don't receive tokens, their spent will be zero, and the min lock demand
//
func (sa *StorageAllocation) minLockDemandLeft() (left state.Balance) {
	for _, details := range sa.BlobberDetails {
		if details.MinLockDemand > details.Spent {
			left += details.MinLockDemand - details.Spent
		}
	}
	return
}

func (sa *StorageAllocation) validate(now common.Timestamp,
	conf *scConfig) (err error) {

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
		//return "", common.NewError("invalid_parameters", "Invalid Allocation root. Allocation root in write marker does not match the commit")
		return false
	}

	if bc.WriteMarker.PreviousAllocationRoot != bc.PrevAllocationRoot {
		//return "", common.NewError("invalid_parameters", "Invalid Previous Allocation root. Previous Allocation root in write marker does not match the commit")
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
}

func (wm *WriteMarker) VerifySignature(clientPublicKey string) bool {
	hashData := wm.GetHashData()
	signatureHash := encryption.Hash(hashData)
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	signatureScheme.SetPublicKey(clientPublicKey)
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
}

func (rm *ReadMarker) VerifySignature(clientPublicKey string) bool {
	hashData := rm.GetHashData()
	signatureHash := encryption.Hash(hashData)
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	signatureScheme.SetPublicKey(clientPublicKey)
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
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v", rm.AllocationID, rm.BlobberID, rm.ClientID, rm.ClientPublicKey, rm.OwnerID, rm.ReadCounter, rm.Timestamp)
	return hashData
}

func (rm *ReadMarker) Verify(prevRM *ReadMarker) error {
	if rm.ReadCounter <= 0 || len(rm.BlobberID) == 0 || len(rm.ClientID) == 0 || rm.Timestamp == 0 {
		return common.NewError("invalid_read_marker", "length validations of fields failed")
	}
	if prevRM != nil {
		if rm.ClientID != prevRM.ClientID || rm.BlobberID != prevRM.BlobberID || rm.Timestamp < prevRM.Timestamp || rm.ReadCounter < prevRM.ReadCounter {
			return common.NewError("invalid_read_marker", "validations with previous marker failed.")
		}
	}
	ok := rm.VerifySignature(rm.ClientPublicKey)
	if ok {
		return nil
	}
	return common.NewError("invalid_read_marker", "Signature verification failed for the read marker")
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

func (vt *ValidationTicket) VerifySign() (bool, error) {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v", vt.ChallengeID, vt.BlobberID, vt.ValidatorID, vt.ValidatorKey, vt.Result, vt.Timestamp)
	hash := encryption.Hash(hashData)
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	signatureScheme.SetPublicKey(vt.ValidatorKey)
	verified, err := signatureScheme.Verify(vt.Signature, hash)
	return verified, err
}

type StorageStats struct {
	Stats              *StorageAllocationStats `json:"stats"`
	LastChallengedSize int64                   `json:"last_challenged_size"`
	LastChallengedTime common.Timestamp        `json:"last_challenged_time"`
}

func (sn *StorageStats) GetKey(globalKey string) datastore.Key {
	return STORAGE_STATS_KEY
}

func (sn *StorageStats) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func (sn *StorageStats) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}
