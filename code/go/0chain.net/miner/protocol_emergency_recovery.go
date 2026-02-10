package miner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// scheduleEmergencyRecovery checks if emergency recovery is enabled in config
// and launches the recovery process as a background goroutine after a delay
// to allow HTTP servers to start.
//
// Called at the end of LoadMagicBlocksAndDKG.
func (mc *Chain) scheduleEmergencyRecovery(_ context.Context, currentMB *block.MagicBlock) {
	if currentMB == nil {
		return
	}

	enabled := viper.GetBool("server_chain.emergency_recovery.enabled")
	if !enabled {
		return
	}

	excludeMiners := viper.GetStringSlice("server_chain.emergency_recovery.exclude_miners")
	if len(excludeMiners) == 0 {
		logging.Logger.Warn("[emergency_recovery] enabled but no miners to exclude")
		return
	}

	logging.Logger.Info("[emergency_recovery] scheduled — will start after HTTP server delay",
		zap.Int64("current_mb", currentMB.MagicBlockNumber),
		zap.Int("exclude_count", len(excludeMiners)))

	go func() {
		// Wait for peer miners' HTTP servers to be reachable.
		waitForPeerMiners(currentMB, 3*time.Minute)

		ctx := context.Background()
		// Retry for up to 10 minutes total. The LFB age guard in
		// attemptEmergencyRecovery requires the chain to be stuck > 5 min,
		// so we need to keep checking well beyond that threshold.
		const (
			maxRetries    = 20
			retryInterval = 30 * time.Second
		)
		for attempt := 1; attempt <= maxRetries; attempt++ {
			logging.Logger.Info("[emergency_recovery] starting attempt",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Int64("current_mb", currentMB.MagicBlockNumber))

			if err := mc.attemptEmergencyRecovery(ctx, currentMB, excludeMiners); err != nil {
				logging.Logger.Error("[emergency_recovery] attempt failed",
					zap.Int("attempt", attempt),
					zap.Error(err))
				if attempt < maxRetries {
					time.Sleep(retryInterval)
					continue
				}
			} else {
				logging.Logger.Info("[emergency_recovery] succeeded",
					zap.Int64("new_mb", currentMB.MagicBlockNumber+1))
			}
			break
		}
	}()
}

// computeExcludeHash computes a deterministic hash of the sorted exclude list.
// All miners must compute the same hash to ensure they agree on who's excluded.
func computeExcludeHash(excludeMiners []string) string {
	sorted := make([]string, len(excludeMiners))
	copy(sorted, excludeMiners)
	sort.Strings(sorted)

	h := sha256.New()
	for _, id := range sorted {
		h.Write([]byte(id))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// attemptEmergencyRecovery creates a new magic block with excluded miners removed,
// runs VRF-seeded DKG among the remaining miners, and registers the new MB.
func (mc *Chain) attemptEmergencyRecovery(ctx context.Context, _ *block.MagicBlock,
	excludeMiners []string) error {

	// Guard: chain must be stuck (LFB age > 5 min)
	lfb := mc.GetLatestFinalizedBlock()
	if lfb == nil {
		return fmt.Errorf("no LFB available")
	}
	lfbAge := time.Since(time.Unix(int64(lfb.CreationDate), 0))
	if lfbAge < 5*time.Minute {
		return fmt.Errorf("chain is active (LFB age %v < 5min), skipping emergency recovery", lfbAge)
	}

	// Use the LATEST magic block, not the one captured at startup.
	// A VC may have happened since startup, changing the active MB.
	currentMB := mc.GetLatestMagicBlock()
	if currentMB == nil {
		return fmt.Errorf("no latest magic block available")
	}

	logging.Logger.Info("[emergency_recovery] chain appears stuck, proceeding",
		zap.Int64("lfb_round", lfb.Round),
		zap.Duration("lfb_age", lfbAge),
		zap.Int64("current_mb", currentMB.MagicBlockNumber))

	excludeHash := computeExcludeHash(excludeMiners)
	excludeSet := make(map[string]bool, len(excludeMiners))
	for _, id := range excludeMiners {
		excludeSet[id] = true
	}

	// Build reduced miner list
	selfKey := node.Self.Underlying().GetKey()
	if excludeSet[selfKey] {
		return fmt.Errorf("self (%s) is in the exclude list", selfKey[:8])
	}

	if currentMB.Miners == nil {
		return fmt.Errorf("current MB has nil miners pool")
	}

	currentMiners := currentMB.Miners.CopyNodesMap()

	// Self must be in the current MB to participate in emergency recovery.
	// A miner dropped by a VC (not in the active MB) should not create new MBs.
	if _, ok := currentMiners[selfKey]; !ok {
		return fmt.Errorf("self (%s) is not in current MB %d miners, cannot run emergency recovery",
			selfKey[:8], currentMB.MagicBlockNumber)
	}
	remainingMiners := make(map[string]*node.Node)
	for id, n := range currentMiners {
		if !excludeSet[id] {
			remainingMiners[id] = n
		}
	}

	newN := len(remainingMiners)
	if newN < 2 {
		return fmt.Errorf("too few remaining miners (%d), need at least 2", newN)
	}

	// Calculate new T and K using 66% BFT threshold for emergency recovery.
	// We don't use currentMB.T/currentMB.N ratio because rounding artifacts
	// can produce T=N (e.g., 4 miners T=3 → 3 miners T=ceil(3*0.75)=3),
	// which means every miner must agree on every round — too fragile.
	// 66% (2/3) is the standard BFT threshold ensuring 1-fault-tolerance.
	newT := int(math.Ceil(float64(newN) * 2.0 / 3.0))
	newK := int(math.Ceil(float64(newN) * 2.0 / 3.0))
	if newT < 1 {
		newT = 1
	}
	if newK < 1 {
		newK = 1
	}
	// Ensure at least 1-fault-tolerance when N >= 3
	if newT >= newN && newN >= 3 {
		newT = newN - 1
	}
	if newK >= newN && newN >= 3 {
		newK = newN - 1
	}

	logging.Logger.Info("[emergency_recovery] building new MB",
		zap.Int("old_n", currentMB.N),
		zap.Int("new_n", newN),
		zap.Int("old_t", currentMB.T),
		zap.Int("new_t", newT),
		zap.Int("old_k", currentMB.K),
		zap.Int("new_k", newK),
		zap.Int("excluded", len(excludeMiners)),
		zap.String("exclude_hash", excludeHash[:16]))

	// Create new MagicBlock
	newMB := block.NewMagicBlock()
	newMB.MagicBlockNumber = currentMB.MagicBlockNumber + 1
	// Deterministic StartingRound: doesn't depend on local LFB state
	newMB.StartingRound = currentMB.StartingRound + 1
	newMB.PreviousMagicBlockHash = currentMB.Hash
	newMB.T = newT
	newMB.K = newK
	newMB.N = newN

	// Create new miner pool with remaining miners
	minerPool := node.NewPool(node.NodeTypeMiner)
	for _, n := range remainingMiners {
		if err := minerPool.AddNode(n.Clone()); err != nil {
			logging.Logger.Warn("[emergency_recovery] failed to add miner to pool",
				zap.String("miner", n.GetKey()[:8]),
				zap.Error(err))
		}
	}
	newMB.Miners = minerPool

	// Keep sharders unchanged
	if currentMB.Sharders != nil {
		newMB.Sharders = currentMB.Sharders.Clone()
	} else {
		newMB.Sharders = node.NewPool(node.NodeTypeSharder)
	}

	// Run DKG among remaining miners
	return mc.runEmergencyDKG(ctx, newMB, currentMB, excludeHash)
}

// runEmergencyDKG performs VRF-seeded DKG among the miners in the new MB,
// collects shares from peers, validates, and persists the new MB + DKG.
func (mc *Chain) runEmergencyDKG(ctx context.Context, newMB, currentMB *block.MagicBlock,
	excludeHash string) error {

	startTime := time.Now()
	selfKey := node.Self.Underlying().GetKey()

	logging.Logger.Info("[emergency_recovery] starting DKG for new MB",
		zap.Int64("new_mb_num", newMB.MagicBlockNumber),
		zap.Int("t", newMB.T),
		zap.Int("n", newMB.N))

	// Step 1: Compute VRF seed for the new MB number
	seed, err := computeDKGSeed(newMB.MagicBlockNumber)
	if err != nil {
		return fmt.Errorf("seed computation failed: %v", err)
	}

	newDKG := bls.MakeDKGSeeded(newMB.T, newMB.N, selfKey, seed)
	newDKG.MagicBlockNumber = newMB.MagicBlockNumber
	newDKG.StartingRound = newMB.StartingRound

	// Step 2: Add own self-share
	selfPartyID := bls.ComputeIDdkg(selfKey)
	selfShare, err := newDKG.ComputeDKGKeyShare(selfPartyID)
	if err != nil {
		return fmt.Errorf("self share computation failed: %v", err)
	}
	if err := newDKG.AddSecretShare(selfPartyID, selfShare.GetHexString(), false); err != nil {
		return fmt.Errorf("self share add failed: %v", err)
	}

	// Collect own MPKs
	selfMPKs := newDKG.GetMPKs()
	selfMPKsHex := make([]string, len(selfMPKs))
	for i, pk := range selfMPKs {
		selfMPKsHex[i] = pk.GetHexString()
	}
	collectedMPKs := map[string][]string{selfKey: selfMPKsHex}

	// Step 3: Request shares from all remaining peer miners
	miners := newMB.Miners.CopyNodesMap()
	const retryDelay = 15 * time.Second

	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("context canceled: %v", ctx.Err())
			default:
			}

			logging.Logger.Info("[emergency_recovery] retrying share collection",
				zap.Int("attempt", attempt+1),
				zap.Int64("mb_num", newMB.MagicBlockNumber),
				zap.Duration("elapsed", time.Since(startTime)))
			time.Sleep(retryDelay)

			// Reset DKG for retry
			newDKG = bls.MakeDKGSeeded(newMB.T, newMB.N, selfKey, seed)
			newDKG.MagicBlockNumber = newMB.MagicBlockNumber
			newDKG.StartingRound = newMB.StartingRound
			if err := newDKG.AddSecretShare(selfPartyID, selfShare.GetHexString(), false); err != nil {
				return fmt.Errorf("self share add on retry: %v", err)
			}
			collectedMPKs = map[string][]string{selfKey: selfMPKsHex}
		}

		received := 1 // self-share already added
		var failedPeers []string

		for minerID := range miners {
			if minerID == selfKey {
				continue
			}

			n := node.GetNode(minerID)
			if n == nil {
				logging.Logger.Warn("[emergency_recovery] peer not found in node registry",
					zap.String("miner", minerID[:8]))
				failedPeers = append(failedPeers, minerID[:8])
				continue
			}

			result, reqErr := mc.requestEmergencyShare(ctx, n, newMB.MagicBlockNumber,
				selfKey, newMB.T, newMB.N, excludeHash)
			if reqErr != nil {
				logging.Logger.Warn("[emergency_recovery] peer share request failed",
					zap.String("miner", minerID[:8]),
					zap.Error(reqErr))
				failedPeers = append(failedPeers, minerID[:8])
				continue
			}

			peerPartyID := bls.ComputeIDdkg(minerID)
			if err := newDKG.AddSecretShare(peerPartyID, result.share, false); err != nil {
				logging.Logger.Warn("[emergency_recovery] failed to add peer share",
					zap.String("miner", minerID[:8]),
					zap.Error(err))
				failedPeers = append(failedPeers, minerID[:8])
				continue
			}
			if len(result.mpksHex) > 0 {
				collectedMPKs[minerID] = result.mpksHex
			}
			received++
		}

		logging.Logger.Info("[emergency_recovery] share collection status",
			zap.Int("attempt", attempt+1),
			zap.Int("received", received),
			zap.Int("needed", newMB.N),
			zap.Int("mpks_collected", len(collectedMPKs)),
			zap.Int("failed", len(failedPeers)),
			zap.Duration("elapsed", time.Since(startTime)))

		// Need ALL N miners' shares for additive DKG
		if received >= newMB.N {
			break
		}
	}

	// Step 4: Aggregate secret shares
	newDKG.AggregateSecretKeyShares()
	newDKG.Pi = newDKG.Si.GetPublicKey()

	// Step 5: Build Mpks from collected VRF-seeded MPKs
	newMpks := block.NewMpks()
	for minerID, mpkHex := range collectedMPKs {
		newMpks.Mpks[minerID] = &block.MPK{
			ID:  minerID,
			Mpk: mpkHex,
		}
	}

	// Step 6: Validate Pi against new MPKs
	newMpkMap, err := newMpks.GetMpkMap()
	if err != nil {
		return fmt.Errorf("failed to parse VRF-seeded MPKs: %v", err)
	}

	if err := newDKG.AggregatePublicKeyShares(newMpkMap); err != nil {
		return fmt.Errorf("failed to aggregate public keys: %v", err)
	}

	mpkStrings := newMpks.GetMpkMapStrings()
	newDKG.SetMpksMap(mpkStrings)

	expectedPi := newDKG.GetPublicKeyByID(selfPartyID)
	if !newDKG.Pi.IsEqual(&expectedPi) {
		return fmt.Errorf("Pi mismatch (expected %s, got %s)",
			expectedPi.GetHexString()[:16], newDKG.Pi.GetHexString()[:16])
	}

	logging.Logger.Info("[emergency_recovery] Pi validated successfully",
		zap.Int64("mb_num", newMB.MagicBlockNumber),
		zap.String("pi", newDKG.Pi.GetHexString()[:16]))

	// Step 7: Set Mpks and ShareOrSigns on the new MB, then compute hash
	newMB.Mpks = newMpks
	// ShareOrSigns is already initialized by NewMagicBlock() (empty)
	// GetHash() includes ShareOrSigns and Mpks keys in the hash
	newMB.Hash = newMB.GetHash()

	logging.Logger.Info("[emergency_recovery] new MB hash computed",
		zap.Int64("mb_num", newMB.MagicBlockNumber),
		zap.String("hash", newMB.Hash[:16]),
		zap.Int64("starting_round", newMB.StartingRound))

	// Step 8: Persist new MB to mb/ RocksDB
	if err := StoreMagicBlock(ctx, newMB); err != nil {
		return fmt.Errorf("failed to store new MB: %v", err)
	}

	// Step 9: Store DKG summary
	summary := newDKG.GetDKGSummary()
	summary.IsFinalized = true
	if err := StoreDKGSummary(ctx, summary); err != nil {
		return fmt.Errorf("failed to store DKG summary: %v", err)
	}

	// Step 10: Set DKG in round storage
	if err := mc.SetDKG(newDKG); err != nil {
		return fmt.Errorf("failed to set DKG in round storage: %v", err)
	}

	// Step 11: Register new MB in chain
	if err := mc.UpdateMagicBlock(newMB); err != nil {
		return fmt.Errorf("failed to update magic block: %v", err)
	}

	// Step 12: Register as LFMB via synthetic block (existing pattern —
	// SetLatestFinalizedMagicBlock handles synthetic blocks with empty hash/round)
	syntheticBlock := &block.Block{}
	syntheticBlock.MagicBlock = newMB
	mc.SetLatestFinalizedMagicBlock(syntheticBlock)

	logging.Logger.Info("[emergency_recovery] complete — new MB registered",
		zap.Int64("mb_num", newMB.MagicBlockNumber),
		zap.Int64("starting_round", newMB.StartingRound),
		zap.String("hash", newMB.Hash[:16]),
		zap.Int("miners", newMB.N),
		zap.Int("t", newMB.T),
		zap.String("pi", newDKG.Pi.GetHexString()[:16]),
		zap.Duration("duration", time.Since(startTime)))

	return nil
}

// requestEmergencyShare sends an emergency recovery share request to a peer.
// Includes new_t, new_n, exclude_hash so the peer can validate agreement.
func (mc *Chain) requestEmergencyShare(ctx context.Context, peer *node.Node,
	newMBNum int64, selfKey string, newT, newN int, excludeHash string) (*recoveryShareResult, error) {

	params := &url.Values{}
	params.Add("mb_num", strconv.FormatInt(newMBNum, 10))
	params.Add("requester_id", selfKey)
	// Emergency mode params
	params.Add("new_t", strconv.Itoa(newT))
	params.Add("new_n", strconv.Itoa(newN))
	params.Add("exclude_hash", excludeHash)

	var result *recoveryShareResult

	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		share, ok := entity.(*bls.DKGKeyShare)
		if !ok {
			return nil, fmt.Errorf("invalid response type")
		}

		if share.Share == "" {
			return nil, fmt.Errorf("empty share")
		}

		// Verify signature from peer
		expectedMessage := encryption.Hash(share.Share + strconv.FormatInt(newMBNum, 10))
		if share.Message != expectedMessage {
			return nil, fmt.Errorf("message mismatch")
		}

		signatureScheme := chain.GetServerChain().GetSignatureScheme()
		if err := signatureScheme.SetPublicKey(peer.PublicKey); err != nil {
			return nil, fmt.Errorf("set public key failed: %v", err)
		}

		ok, err := signatureScheme.Verify(share.Sign, share.Message)
		if !ok || err != nil {
			return nil, fmt.Errorf("signature verification failed: %v", err)
		}

		result = &recoveryShareResult{
			share:   share.Share,
			mpksHex: share.MpksHex,
		}
		return nil, nil
	}

	if !peer.RequestEntityFromNode(ctx, DKGShareRecoverySender, params, handler) {
		return nil, fmt.Errorf("request to %s failed", peer.ID[:8])
	}

	if result == nil || result.share == "" {
		return nil, fmt.Errorf("no share from %s", peer.ID[:8])
	}

	return result, nil
}
