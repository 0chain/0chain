package round

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
)

type Phase int32

const (
	ShareVRF Phase = iota
	Verify
	//mb cancelled is better name
	Notarize
	Share
	Complete
)

var (
	CompleteRoundRestartError = errors.New("can't restart notarized or complete round")
)

func GetPhaseName(ph Phase) string {
	name, ok := map[Phase]string{
		ShareVRF: "ShareVRF",
		Verify:   "Verify",
		Notarize: "Notarize",
		Share:    "Share",
		Complete: "Complete",
	}[ph]

	if !ok {
		return "N/A"
	}
	return name
}

type FinalizingState int32

const (
	NotFinalized FinalizingState = iota
	RoundStateFinalizing
	RoundStateFinalized
)

type timeoutCounter struct {
	mutex sync.RWMutex // async safe

	prrs int64    // previous round random seed
	perm []string // miners of this (not previous) round sorted by the seed

	count int // current round timeout

	votes map[string]int // voted miner_id -> timeout
}

// The rankTimeoutCounters computes ranks of miners to choose timeout counter.
// Should be called under lock.
func (tc *timeoutCounter) rankTimeoutCounters(prrs int64, miners *node.Pool) {

	var nodes = miners.CopyNodes()

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})

	var (
		permi = rand.New(rand.NewSource(prrs)).Perm(len(nodes))
		perms = make([]string, 0, len(nodes))
	)

	for _, ri := range permi {
		perms = append(perms, nodes[ri].ID)
	}

	tc.prrs = prrs
	tc.perm = perms
}

func (tc *timeoutCounter) resetVotes() {
	tc.votes = make(map[string]int)
}

func (tc *timeoutCounter) AddTimeoutVote(num int, id string) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if tc.votes == nil {
		tc.resetVotes() // it creates the map
	}
	tc.votes[id] = num
}

// IncrementTimeoutCount - increments timeout count.
func (tc *timeoutCounter) IncrementTimeoutCount(prrs int64, miners *node.Pool) {
	if prrs == 0 {
		return // no PRRS, no timeout incrementation
	}

	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if tc.votes == nil {
		tc.resetVotes() // it creates the map
		tc.count++
		tc.checkCap()
		return
	}

	if len(tc.perm) == 0 {
		tc.rankTimeoutCounters(prrs, miners)
	}

	// initial count
	var (
		from = tc.count
		snk  = node.Self.Underlying().GetKey()
	)

	// from most ranked to the lowest ranked one
	for _, minerID := range tc.perm {
		if snk == minerID {
			continue
		}
		if vote, ok := tc.votes[minerID]; ok {
			if tc.count < vote {
				tc.count = vote
				break
			}
		}
	}

	tc.resetVotes()

	// increase if has not increased
	if tc.count == from {
		tc.count++
	}
	tc.checkCap()
}

func (tc *timeoutCounter) checkCap() {
	timeoutCap := viper.GetInt("server_chain.round_timeouts.timeout_cap")
	if timeoutCap > 0 && tc.count > timeoutCap {
		tc.count = timeoutCap
	}
}

// SetTimeoutCount - sets the timeout count to given number if it is greater
// than existing and returns true. Else false.
func (tc *timeoutCounter) SetTimeoutCount(count int) (set bool) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if count <= tc.count {
		return // false (not set)
	}

	tc.count = count
	return true // set
}

// GetTimeoutCount - returns the timeout count
func (tc *timeoutCounter) GetTimeoutCount() (count int) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	return tc.count
}

func (tc *timeoutCounter) GetNormalizedTimeoutCount() int {
	return tc.GetTimeoutCount()
}

/*Round - data structure for the round */
type Round struct {
	datastore.NOIDField
	Number     int64 `json:"number"`
	RandomSeed int64 `json:"round_random_seed"`

	// For generator, this is the block the miner is generating till a
	// notarization is received. For a verifier, this is the block that is
	// currently the best block received for verification. Once a round is
	// finalized, this is the finalized block of the given round.
	Block     *block.Block `json:"-" msgpack:"-"`
	BlockHash string       `json:"block_hash"`
	VRFOutput string       `json:"vrf_output"` // TODO: VRFOutput == rbooutput?

	minerPerm       []int
	phase           Phase
	finalizingState FinalizingState
	proposedBlocks  []*block.Block
	notarizedBlocks []*block.Block
	mutex           sync.RWMutex
	shares          map[string]*VRFShare

	softTimeoutCount int32
	vrfStartTime     atomic.Value

	timeoutCounter
}

// RoundFactory - a factory to create a new round object specific to miner/sharder
type RoundFactory interface {
	CreateRoundF(roundNum int64) RoundI
}

// NewRound - Create a new round object
func NewRound(round int64) *Round {
	r := datastore.GetEntityMetadata("round").Instance().(*Round)
	r.Number = round
	return r
}

var roundEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (r *Round) GetEntityMetadata() datastore.EntityMetadata {
	return roundEntityMetadata
}

/*GetKey - returns the round number as the key */
func (r *Round) GetKey() datastore.Key {
	return datastore.ToKey(fmt.Sprintf("%v", r.GetRoundNumber()))
}

// GetRoundNumber - returns the round number
func (r *Round) GetRoundNumber() int64 {
	return r.Number
}

// SetRandomSeedForNotarizedBlock - set the random seed of the round
func (r *Round) SetRandomSeedForNotarizedBlock(seed int64, minersNum int) {
	r.mutex.Lock()
	r.minerPerm = computeMinerRanks(seed, minersNum)
	r.mutex.Unlock()

	r.setRandomSeed(seed)
}

// SetRandomSeed - set the random seed of the round
func (r *Round) SetRandomSeed(seed int64, minersNum int) {
	if r.HasRandomSeed() {
		return
	}

	r.mutex.Lock()
	r.minerPerm = computeMinerRanks(seed, minersNum)
	r.mutex.Unlock()

	r.setRandomSeed(seed)
	//r.setPhase(Verify)
}

func (r *Round) setRandomSeed(seed int64) {
	atomic.StoreInt64(&r.RandomSeed, seed)
}

// GetRandomSeed - returns the random seed of the round.
func (r *Round) GetRandomSeed() int64 {
	return atomic.LoadInt64(&r.RandomSeed)
}

// SetVRFOutput --sets the VRFOutput.
func (r *Round) SetVRFOutput(rboutput string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.VRFOutput = rboutput
}

// GetVRFOutput --gets the VRFOutput.
func (r *Round) GetVRFOutput() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.VRFOutput
}

// AddNotarizedBlock - this will be concurrent as notarization is recognized by
// verifying as well as notarization message from others.
func (r *Round) AddNotarizedBlock(b *block.Block) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.addProposedBlock(b)
	found := -1

	for i, blk := range r.notarizedBlocks {
		if blk.Hash == b.Hash {
			if blk != b {
				blk.MergeVerificationTickets(b.GetVerificationTickets())
				b.MergeVerificationTickets(blk.GetVerificationTickets())
			}
			logging.Logger.Debug("add notarized block - block already exist, merge tickets",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			return
		}
		if blk.RoundRank == b.RoundRank {
			found = i
		}
	}

	if found > -1 {
		fb := r.notarizedBlocks[found]
		logging.Logger.Info("Removing the old notarized block with the same rank",
			zap.Int64("round", r.GetRoundNumber()), zap.String("hash", fb.Hash),
			zap.Int64("fb_RRS", fb.GetRoundRandomSeed()),
			zap.Int("fb_toc", fb.RoundTimeoutCount),
			zap.String("fb_Sender", fb.MinerID))
		// remove the old block with the same rank and add it below
		r.notarizedBlocks = append(r.notarizedBlocks[:found], r.notarizedBlocks[found+1:]...)
	}
	b.SetBlockNotarized()
	b.SetBlockState(block.StateNotarized)
	r.setPhase(Share)

	if r.Block == nil || (r.Block.RoundRank > b.RoundRank && b.RoundRank >= 0) {
		r.Block = b
	}

	rnb := append(r.notarizedBlocks, b)
	sort.Slice(rnb, func(i int, j int) bool {
		return rnb[i].Weight() > rnb[j].Weight()
	})
	r.notarizedBlocks = rnb
	logging.Logger.Debug("reached notarization", zap.Int64("round", b.Round))
}

// UpdateNotarizedBlock updates the notarized block in the round
func (r *Round) UpdateNotarizedBlock(b *block.Block) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	// update proposed blocks
	for i, pb := range r.proposedBlocks {
		if pb.Hash == b.Hash {
			r.proposedBlocks[i] = b
		}
	}

	// update notarized block
	for i, nb := range r.notarizedBlocks {
		if nb.Hash == b.Hash {
			r.notarizedBlocks[i] = nb
		}
	}
}

/*GetNotarizedBlocks - return all the notarized blocks associated with this round */
func (r *Round) GetNotarizedBlocks() []*block.Block {
	return r.notarizedBlocks
}

/*AddProposedBlock - this will be concurrent as notarization is recognized by verifying as well as notarization message from others */
func (r *Round) AddProposedBlock(b *block.Block) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.addProposedBlock(b)
}

func (r *Round) addProposedBlock(b *block.Block) {
	for i, blk := range r.proposedBlocks {
		if blk.Hash == b.Hash {
			r.proposedBlocks[i] = b
			return
		}
	}
	r.proposedBlocks = append(r.proposedBlocks, b)
	sort.SliceStable(r.proposedBlocks, func(i, j int) bool {
		return r.proposedBlocks[i].RoundRank < r.proposedBlocks[j].RoundRank && r.proposedBlocks[i].RoundRank >= 0 // avoid treat -1 as the highest rank
	})
	//nolint:gosimple
	return
}

/*GetProposedBlocks - return all the blocks that have been proposed for this round */
func (r *Round) GetProposedBlocks() []*block.Block {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.proposedBlocks
}

func (r *Round) GetBestRankedProposedBlock() *block.Block {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	pbs := r.proposedBlocks
	if len(pbs) == 0 {
		return nil
	}
	if len(pbs) == 1 {
		return pbs[0]
	}
	pbs = r.GetBlocksByRank(pbs)
	return pbs[0]
}

/*GetHeaviestNotarizedBlock - get the heaviest notarized block that we have in this round */
func (r *Round) GetHeaviestNotarizedBlock() *block.Block {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	rnb := r.notarizedBlocks
	if len(rnb) == 0 {
		return nil
	}
	return rnb[0]
}

/*GetBlocksByRank - return the currently stored blocks in the order of best rank for the round */
func (r *Round) GetBlocksByRank(blocks []*block.Block) []*block.Block {
	sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].RoundRank < blocks[j].RoundRank })
	return blocks
}

/*GetBestRankedNotarizedBlock - get the best ranked notarized block for this round */
func (r *Round) GetBestRankedNotarizedBlock() *block.Block {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	rnb := r.notarizedBlocks
	if len(rnb) == 0 {
		return nil
	}
	if len(rnb) == 1 {
		return rnb[0]
	}
	rnb = r.GetBlocksByRank(rnb)
	return rnb[0]
}

/*Finalize - finalize the round */
func (r *Round) Finalize(b *block.Block) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.setFinalizingPhase(RoundStateFinalized)
	r.Block = b
	r.BlockHash = b.Hash
}

func (r *Round) GetBlockHash() (hash string) {
	r.mutex.Lock()
	hash = r.BlockHash
	r.mutex.Unlock()
	return
}

/*SetFinalizing - the round is being finalized */
func (r *Round) SetFinalizing() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isFinalized() || r.isFinalizing() {
		return false
	}
	r.setFinalizingPhase(RoundStateFinalizing)
	return true
}

func (r *Round) SetFinalized() {
	r.mutex.Lock()
	logging.Logger.Debug("Set round as finalized", zap.Int64("round", r.Number))
	r.setFinalizingPhase(RoundStateFinalized)
	r.mutex.Unlock()
}

// ResetFinalizeStateIfNotFinalized reset finalizing state if it's not finalized yet,
// otherwise do nothing. This is for protecting the finalized round get reset
func (r *Round) ResetFinalizingStateIfNotFinalized() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.isFinalized() {
		return
	}
	r.setFinalizingPhase(NotFinalized)
}

func (r *Round) ResetFinalizingState() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	logging.Logger.Debug("reset finalizing state",
		zap.Int64("round", r.Number),
		zap.String("block", r.BlockHash))
	r.setFinalizingPhase(NotFinalized)
}

/*IsFinalizing - is the round finalizing */
func (r *Round) IsFinalizing() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.isFinalizing()
}

func (r *Round) isFinalizing() bool {
	return r.getFinalizingState() == RoundStateFinalizing
}

/*IsFinalized - indicates if the round is finalized */
func (r *Round) IsFinalized() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.isFinalized()
}

func (r *Round) isFinalized() bool {
	return r.getFinalizingState() == RoundStateFinalized || r.GetRoundNumber() == 0
}

func (r *Round) FinalizeState() FinalizingState {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.finalizingState
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	r := &Round{}
	r.initialize()
	r.timeoutCounter.resetVotes() // create votes maps
	return r
}

func (r *Round) initialize() {
	r.notarizedBlocks = make([]*block.Block, 0, 1)
	r.proposedBlocks = make([]*block.Block, 0, 3)
	r.shares = make(map[string]*VRFShare)
	// when we restart a round we call this. So, explicitly, set them to default
	r.setRandomSeed(0)
}

/*Read - read round entity from store */
func (r *Round) Read(ctx context.Context, key datastore.Key) error {
	return r.GetEntityMetadata().GetStore().Read(ctx, key, r)
}

/*Write - write round entity to store */
func (r *Round) Write(ctx context.Context) error {
	return r.GetEntityMetadata().GetStore().Write(ctx, r)
}

/*Delete - delete round entity from store */
func (r *Round) Delete(ctx context.Context) error {
	return r.GetEntityMetadata().GetStore().Delete(ctx, r)
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	roundEntityMetadata = datastore.MetadataProvider()
	roundEntityMetadata.Name = "round"
	roundEntityMetadata.DB = "roundsummarydb"
	roundEntityMetadata.Provider = Provider
	roundEntityMetadata.Store = store
	roundEntityMetadata.IDColumnName = "number"
	datastore.RegisterEntityMetadata("round", roundEntityMetadata)
}

// SetupRoundSummaryDB - setup the round summary db
func SetupRoundSummaryDB(workdir string) {
	datadir := filepath.Join(workdir, "data/rocksdb/roundsummary")

	db, err := ememorystore.CreateDB(datadir)
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("roundsummarydb", db)
}

/*ComputeMinerRanks - Compute random order of n elements given the random seed of the round */
func computeMinerRanks(seed int64, minersNum int) []int {
	return rand.New(rand.NewSource(seed)).Perm(minersNum)
}

func (r *Round) IsRanksComputed() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.minerPerm != nil
}

/*GetMinerRank - get the rank of element at the elementIdx position based on the permutation of the round */
func (r *Round) GetMinerRank(miner *node.Node) int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if r.minerPerm == nil {
		_ = pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		logging.Logger.DPanic(fmt.Sprintf("miner ranks not computed yet: %v, random seed: %v, round: %v",
			r.GetPhase(), r.GetRandomSeed(), r.GetRoundNumber()))
	}
	if miner.SetIndex >= len(r.minerPerm) {
		logging.Logger.Warn("get miner rank -- the node index in the permutation is missing. Returns: -1.",
			zap.Ints("r.minerPerm", r.minerPerm), zap.Int("set_index", miner.SetIndex),
			zap.String("node", miner.ID))
		return -1
	}
	return r.minerPerm[miner.SetIndex]
}

/*GetMinersByRank - get the rnaks of the miners */
func (r *Round) GetMinersByRank(nodes []*node.Node) []*node.Node {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	logging.Logger.Info("get miners by rank", zap.Int("num_miners", len(nodes)),
		zap.Int64("round", r.Number), zap.Ints("r.minerPerm", r.minerPerm))
	sort.Slice(nodes, func(i, j int) bool {
		idxi, idxj := 0, 0
		if nodes[i].SetIndex < len(r.minerPerm) {
			idxi = r.minerPerm[nodes[i].SetIndex]
		} else {
			logging.Logger.Warn("get miner by rank -- the node index in the permutation is missing",
				zap.Ints("r.minerPerm", r.minerPerm), zap.Int("set_index", nodes[i].SetIndex),
				zap.String("node", nodes[i].ID))
		}
		if nodes[j].SetIndex < len(r.minerPerm) {
			idxj = r.minerPerm[nodes[j].SetIndex]
		} else {
			logging.Logger.Warn("get miner by rank -- the node index in the permutation is missing",
				zap.Ints("r.minerPerm", r.minerPerm), zap.Int("set_index", nodes[j].SetIndex),
				zap.String("node", nodes[j].ID))
		}
		// return idxi > idxj
		return idxi < idxj
	})
	return nodes
}

// Clear - implement interface
func (r *Round) Clear() {
}

// Restart - restart the round
func (r *Round) Restart() error {
	r.mutex.Lock()
	if r.getState() >= Share {
		return CompleteRoundRestartError
	}
	r.initialize()
	r.Block = nil
	r.resetSoftTimeoutCount()
	r.ResetPhase(ShareVRF)

	r.mutex.Unlock()
	return nil
}

// VRFShareExist checks if the VRF share already exist
func (r *Round) VRFShareExist(share *VRFShare) (exist bool) {
	r.mutex.Lock()
	_, exist = r.shares[share.party.GetKey()]
	r.mutex.Unlock()
	return
}

// AddVRFShare - implement interface
func (r *Round) AddVRFShare(share *VRFShare, threshold int) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if len(r.getVRFShares()) >= threshold {
		//if we already have enough shares, do not add.
		logging.Logger.Info("add_vrf_share already at threshold. Returning false.")
		return false
	}
	if _, ok := r.shares[share.party.GetKey()]; ok {
		logging.Logger.Info("add_vrf_share share is already there. Returning false.")
		return false
	}
	r.setPhase(ShareVRF)
	r.shares[share.party.GetKey()] = share
	logging.Logger.Debug("add_vrf_share",
		zap.Int64("round", r.GetRoundNumber()),
		zap.Int("round_vrf_num", len(r.getVRFShares())),
		zap.Int("threshold", threshold))
	return true
}

// GetVRFShares - implement interface
func (r *Round) GetVRFShares() map[string]*VRFShare {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.getVRFShares()
}

func (r *Round) getVRFShares() map[string]*VRFShare {
	result := make(map[string]*VRFShare, len(r.shares))
	for k, v := range r.shares {
		result[k] = v
	}
	return result
}

// GetPhase - get the phase of the round
func (r *Round) GetPhase() Phase {
	return r.getState()
}

// SetPhase - set the phase of the round in a progressive order
func (r *Round) SetPhase(state Phase) {
	r.setPhase(state)
}

// ResetPhase resets the phase to any desired phase
func (r *Round) ResetPhase(state Phase) {
	atomic.StoreInt32((*int32)(&r.phase), int32(state))
}

func (r *Round) getState() Phase {
	return Phase(atomic.LoadInt32((*int32)(&r.phase)))
}

func (r *Round) setPhase(state Phase) {
	if state > r.getState() {
		atomic.StoreInt32((*int32)(&r.phase), int32(state))
	}
}

// HasRandomSeed - implement interface
func (r *Round) HasRandomSeed() bool {
	return atomic.LoadInt64(&r.RandomSeed) != 0
}

func (r *Round) GetSoftTimeoutCount() int {
	return int(atomic.LoadInt32(&r.softTimeoutCount))
}

func (r *Round) IncSoftTimeoutCount() {
	atomic.AddInt32(&r.softTimeoutCount, 1)
}

func (r *Round) resetSoftTimeoutCount() {
	atomic.StoreInt32(&r.softTimeoutCount, 0)
}

func (r *Round) SetVrfStartTime(t time.Time) {
	r.vrfStartTime.Store(t)
}

func (r *Round) GetVrfStartTime() time.Time {
	value := r.vrfStartTime.Load()
	if value == nil {
		return time.Time{}
	}
	return value.(time.Time)
}

func (r *Round) setFinalizingPhase(finalized FinalizingState) {
	r.finalizingState = finalized
}

func (r *Round) getFinalizingState() FinalizingState {
	return r.finalizingState
}

// Clone do light copy of round
func (r *Round) Clone() RoundI {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var (
		mp      = make([]int, len(r.minerPerm))
		pblocks = make([]*block.Block, len(r.proposedBlocks))
		nblocks = make([]*block.Block, len(r.notarizedBlocks))
		shares  = make(map[string]*VRFShare, len(r.shares))
	)

	copy(mp, r.minerPerm)

	for i, b := range r.proposedBlocks {
		pblocks[i] = b.Clone()
	}

	for i, b := range r.notarizedBlocks {
		nblocks[i] = b.Clone()
	}

	for k, s := range r.shares {
		shares[k] = s.Clone()
	}

	return &Round{
		Number:           r.Number,
		RandomSeed:       r.RandomSeed,
		Block:            r.Block.Clone(),
		BlockHash:        r.BlockHash,
		VRFOutput:        r.VRFOutput,
		minerPerm:        mp,
		phase:            r.phase,
		finalizingState:  r.finalizingState,
		proposedBlocks:   pblocks,
		notarizedBlocks:  nblocks,
		shares:           shares,
		softTimeoutCount: r.softTimeoutCount,
		vrfStartTime:     r.vrfStartTime,
		timeoutCounter: timeoutCounter{
			prrs:  r.timeoutCounter.prrs,
			perm:  r.timeoutCounter.perm,
			count: r.timeoutCounter.count,
			votes: r.timeoutCounter.votes,
		},
	}
}
