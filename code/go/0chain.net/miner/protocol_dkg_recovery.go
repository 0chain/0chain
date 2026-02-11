package miner

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// waitForPeerMiners polls peer miner HTTP endpoints until at least one responds.
// Returns true if a peer was reached, false if we timed out.
func waitForPeerMiners(mb *block.MagicBlock, maxWait time.Duration) bool {
	if mb == nil || mb.Miners == nil {
		return false
	}

	selfKey := node.Self.Underlying().GetKey()
	miners := mb.Miners.CopyNodesMap()

	const pollInterval = 3 * time.Second
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		for id, n := range miners {
			if id == selfKey {
				continue
			}
			statusURL := n.GetStatusURL()
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Get(statusURL)
			if err == nil {
				resp.Body.Close()
				logging.Logger.Info("waitForPeerMiners - peer miner is reachable",
					zap.String("miner", id[:8]),
					zap.String("url", statusURL))
				return true
			}
		}
	}

	logging.Logger.Warn("waitForPeerMiners - timed out waiting for peer miners",
		zap.Duration("max_wait", maxWait))
	return false
}

// scheduleVRFRecovery launches RecoverDKG as a background goroutine.
// This is necessary because RecoverDKG loops indefinitely waiting for peers,
// but during startup the HTTP server (which serves recovery shares) hasn't
// started yet. Running synchronously would deadlock all miners.
// The goroutine waits for peer miners to be reachable, then begins recovery.
// Uses context.Background() — the recovery must survive the caller's context lifecycle.
func (mc *Chain) scheduleVRFRecovery(_ context.Context, mb *block.MagicBlock) {
	go func() {
		// Wait for peer miners' HTTP servers to be reachable before attempting recovery.
		waitForPeerMiners(mb, 3*time.Minute)

		// Use background context — the caller's context may be canceled before
		// the 30s delay completes (e.g., startup context canceled on restart).
		ctx := context.Background()

		// 20 retries * 30s = 10 min window. Must exceed the 3-minute LFB age
		// guard in RecoverDKG so that force recovery can trigger even if the
		// chain was recently active when the goroutine first runs.
		const maxRetries = 20
		const retryDelay = 30 * time.Second
		for attempt := 1; attempt <= maxRetries; attempt++ {
			logging.Logger.Info("[dkg_recovery] starting scheduled VRF recovery",
				zap.Int64("mb_num", mb.MagicBlockNumber),
				zap.Int64("mb_sr", mb.StartingRound),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries))

			if err := mc.RecoverDKG(ctx, mb); err != nil {
				logging.Logger.Error("[dkg_recovery] scheduled VRF recovery failed",
					zap.Int64("mb_num", mb.MagicBlockNumber),
					zap.Int("attempt", attempt),
					zap.Error(err))
				if attempt < maxRetries {
					logging.Logger.Info("[dkg_recovery] retrying after delay",
						zap.Int64("mb_num", mb.MagicBlockNumber),
						zap.Int("next_attempt", attempt+1))
					time.Sleep(retryDelay)
					continue
				}
			} else {
				logging.Logger.Info("[dkg_recovery] scheduled VRF recovery succeeded",
					zap.Int64("mb_num", mb.MagicBlockNumber))
			}
			break
		}
	}()
}

// computeDKGSeed computes a deterministic VRF seed for DKG polynomial generation.
// seed = Sign(node_private_key, Hash("DKG-SEED:<mb_number>"))
// Ed25519 signatures are deterministic: same key + same message = same output.
func computeDKGSeed(mbNumber int64) ([]byte, error) {
	message := fmt.Sprintf("DKG-SEED:%d", mbNumber)
	hash := encryption.Hash(message)
	sig, err := node.Self.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to compute DKG seed: %v", err)
	}
	return []byte(sig), nil
}

// RecoverShareRequestHandler handles DKG recovery share requests from peer miners.
// A recovering miner asks each peer: "give me S_you_me for MB#N."
// The peer regenerates its VRF-seeded polynomial and computes the share on the fly.
//
// Endpoint: /v1/_m2m/dkg/recover_share
// Params:
//
//	mb_num (required): magic block number
//	requester_id (required): requesting miner's node ID
//	new_t, new_n, exclude_hash (optional): emergency recovery mode params
//
// In emergency mode, the handler validates the exclude_hash against its own config,
// builds a reduced miner set, and computes shares with the new T/N values.
func RecoverShareRequestHandler(ctx context.Context, r *http.Request) (
	resp interface{}, err error) {

	// Recovery handler is always active — VRF-seeded recovery is a startup mechanism
	// with no consensus impact, safe to serve regardless of hardfork status.

	mbNumStr := r.FormValue("mb_num")
	requesterID := r.FormValue("requester_id")

	if mbNumStr == "" || requesterID == "" {
		return nil, common.NewError("recover_share", "mb_num and requester_id are required")
	}

	mbNum, err := strconv.ParseInt(mbNumStr, 10, 64)
	if err != nil {
		return nil, common.NewErrorf("recover_share", "invalid mb_num: %v", err)
	}

	// Check for emergency recovery mode params
	newTStr := r.FormValue("new_t")
	newNStr := r.FormValue("new_n")
	peerExcludeHash := r.FormValue("exclude_hash")
	isEmergencyMode := newTStr != "" && newNStr != "" && peerExcludeHash != ""

	if isEmergencyMode {
		return handleEmergencyRecoveryShare(ctx, mbNum, mbNumStr, requesterID, newTStr, newNStr, peerExcludeHash)
	}

	// Normal recovery mode — load the existing MB and compute share for it
	mb, loadErr := LoadMagicBlock(ctx, mbNumStr)
	if loadErr != nil {
		return nil, common.NewErrorf("recover_share", "magic block %d not found: %v", mbNum, loadErr)
	}

	if mb.Miners == nil {
		return nil, common.NewErrorf("recover_share", "MB %d has nil miners pool", mbNum)
	}

	// Verify requester is a miner in this MB
	miners := mb.Miners.CopyNodesMap()
	if _, ok := miners[requesterID]; !ok {
		return nil, common.NewErrorf("recover_share",
			"requester %s is not in MB %d", requesterID[:8], mbNum)
	}

	// Verify we are a miner in this MB
	selfKey := node.Self.Underlying().GetKey()
	if _, ok := miners[selfKey]; !ok {
		return nil, common.NewErrorf("recover_share",
			"self not in MB %d", mbNum)
	}

	// Regenerate our VRF-seeded polynomial for this MB
	seed, err := computeDKGSeed(mbNum)
	if err != nil {
		return nil, common.NewErrorf("recover_share", "seed computation failed: %v", err)
	}

	tempDKG := bls.MakeDKGSeeded(mb.T, mb.N, selfKey, seed)

	// Compute the share for the requester: S_self_requester
	requesterPartyID := bls.ComputeIDdkg(requesterID)
	share, err := tempDKG.ComputeDKGKeyShare(requesterPartyID)
	if err != nil {
		return nil, common.NewErrorf("recover_share",
			"failed to compute share for %s: %v", requesterID[:8], err)
	}

	// Return signed response
	shareHex := share.GetHexString()
	message := encryption.Hash(shareHex + mbNumStr)
	sign, err := node.Self.Sign(message)
	if err != nil {
		return nil, common.NewErrorf("recover_share", "failed to sign: %v", err)
	}

	// Include our VRF-seeded MPKs for force-recovery (when MB has CSPRNG MPKs)
	mpks := tempDKG.GetMPKs()
	mpksHex := make([]string, len(mpks))
	for i, pk := range mpks {
		mpksHex[i] = pk.GetHexString()
	}

	result := datastore.GetEntityMetadata("dkg_share").Instance().(*bls.DKGKeyShare)
	result.Message = message
	result.Sign = sign
	result.Share = shareHex
	result.MpksHex = mpksHex

	logging.Logger.Info("[dkg_recovery] sent share to recovering miner",
		zap.Int64("mb_num", mbNum),
		zap.String("requester", requesterID[:8]))

	return result, nil
}

// handleEmergencyRecoveryShare handles the emergency recovery mode of share requests.
// The requester is asking for a share for a NEW MB (mbNum) with reduced T/N.
// We validate the exclude_hash against our own config, then compute the share
// using the new T/N and the VRF seed for the new MB number.
func handleEmergencyRecoveryShare(ctx context.Context, mbNum int64, mbNumStr, requesterID,
	newTStr, newNStr, peerExcludeHash string) (interface{}, error) {

	// Validate our own emergency recovery config
	excludeMiners := viper.GetStringSlice("server_chain.emergency_recovery.exclude_miners")
	if len(excludeMiners) == 0 {
		return nil, common.NewError("recover_share",
			"emergency mode requested but no exclude_miners in local config")
	}

	localExcludeHash := computeExcludeHash(excludeMiners)
	if localExcludeHash != peerExcludeHash {
		return nil, common.NewErrorf("recover_share",
			"exclude_hash mismatch: local=%s, peer=%s", localExcludeHash[:16], peerExcludeHash[:16])
	}

	newT, err := strconv.Atoi(newTStr)
	if err != nil {
		return nil, common.NewErrorf("recover_share", "invalid new_t: %v", err)
	}
	newN, err := strconv.Atoi(newNStr)
	if err != nil {
		return nil, common.NewErrorf("recover_share", "invalid new_n: %v", err)
	}

	selfKey := node.Self.Underlying().GetKey()

	// Verify self and requester are not excluded
	excludeSet := make(map[string]bool, len(excludeMiners))
	for _, id := range excludeMiners {
		excludeSet[id] = true
	}
	if excludeSet[selfKey] {
		return nil, common.NewError("recover_share", "self is in exclude list")
	}
	if excludeSet[requesterID] {
		return nil, common.NewErrorf("recover_share",
			"requester %s is in exclude list", requesterID[:8])
	}

	// Compute VRF-seeded polynomial for the NEW MB number with new T/N
	seed, err := computeDKGSeed(mbNum)
	if err != nil {
		return nil, common.NewErrorf("recover_share", "seed computation failed: %v", err)
	}

	tempDKG := bls.MakeDKGSeeded(newT, newN, selfKey, seed)

	// Compute share for the requester
	requesterPartyID := bls.ComputeIDdkg(requesterID)
	share, err := tempDKG.ComputeDKGKeyShare(requesterPartyID)
	if err != nil {
		return nil, common.NewErrorf("recover_share",
			"failed to compute emergency share for %s: %v", requesterID[:8], err)
	}

	// Return signed response
	shareHex := share.GetHexString()
	message := encryption.Hash(shareHex + mbNumStr)
	sign, err := node.Self.Sign(message)
	if err != nil {
		return nil, common.NewErrorf("recover_share", "failed to sign: %v", err)
	}

	// Include VRF-seeded MPKs
	mpks := tempDKG.GetMPKs()
	mpksHex := make([]string, len(mpks))
	for i, pk := range mpks {
		mpksHex[i] = pk.GetHexString()
	}

	result := datastore.GetEntityMetadata("dkg_share").Instance().(*bls.DKGKeyShare)
	result.Message = message
	result.Sign = sign
	result.Share = shareHex
	result.MpksHex = mpksHex

	logging.Logger.Info("[dkg_recovery] sent emergency share to recovering miner",
		zap.Int64("mb_num", mbNum),
		zap.String("requester", requesterID[:8]),
		zap.Int("new_t", newT),
		zap.Int("new_n", newN),
		zap.String("exclude_hash", peerExcludeHash[:16]))

	return result, nil
}

// RecoverDKG attempts to recover the DKG for a given magic block by:
// 1. Regenerating own polynomial from VRF seed
// 2. Requesting S_j_self AND peer MPKs from each peer miner j
// 3. Aggregating shares, validating Pi against MB MPKs
// 4. If Pi mismatch (CSPRNG-based MB), force-updates MB's MPKs to VRF-seeded ones
// 5. Storing the recovered DKG (and updated MB if force recovery)
//
// Triggered when SetDKGSFromStore detects nil DKG or Pi mismatch.
func (mc *Chain) RecoverDKG(ctx context.Context, mb *block.MagicBlock) error {
	startTime := time.Now()
	selfKey := node.Self.Underlying().GetKey()

	// Read excluded miners from emergency recovery config.
	// These miners are known to be absent and should not be waited for.
	excludeMiners := viper.GetStringSlice("server_chain.emergency_recovery.exclude_miners")
	excludeSet := make(map[string]bool, len(excludeMiners))
	for _, id := range excludeMiners {
		excludeSet[id] = true
	}
	requiredN := mb.N - len(excludeMiners)
	if requiredN < mb.T {
		requiredN = mb.T // never go below threshold
	}

	logging.Logger.Info("[dkg_recovery] starting recovery",
		zap.Int64("mb_num", mb.MagicBlockNumber),
		zap.Int64("mb_sr", mb.StartingRound),
		zap.Int("total_n", mb.N),
		zap.Int("excluded", len(excludeMiners)),
		zap.Int("required_n", requiredN))

	// Step 1: Regenerate own polynomial from VRF seed
	seed, err := computeDKGSeed(mb.MagicBlockNumber)
	if err != nil {
		return fmt.Errorf("seed computation failed: %v", err)
	}

	newDKG := bls.MakeDKGSeeded(mb.T, mb.N, selfKey, seed)
	newDKG.MagicBlockNumber = mb.MagicBlockNumber
	newDKG.StartingRound = mb.StartingRound

	// Step 2: Add own self-share (S_self_self) and own MPKs
	selfPartyID := bls.ComputeIDdkg(selfKey)
	selfShare, err := newDKG.ComputeDKGKeyShare(selfPartyID)
	if err != nil {
		return fmt.Errorf("self share computation failed: %v", err)
	}
	if err := newDKG.AddSecretShare(selfPartyID, selfShare.GetHexString(), false); err != nil {
		return fmt.Errorf("self share add failed: %v", err)
	}

	// Collect own MPKs for force recovery
	selfMPKs := newDKG.GetMPKs()
	selfMPKsHex := make([]string, len(selfMPKs))
	for i, pk := range selfMPKs {
		selfMPKsHex[i] = pk.GetHexString()
	}
	// Map minerID -> VRF-seeded MPKs (hex strings)
	collectedMPKs := map[string][]string{selfKey: selfMPKsHex}

	// Step 3: Request shares and MPKs from peer miners.
	// Loop indefinitely — without DKG the chain cannot progress, so giving up is pointless.
	// Exit conditions: enough shares collected, or context canceled (chain moved forward).
	miners := mb.Miners.CopyNodesMap()
	const retryDelay = 15 * time.Second

	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			// Check if context was canceled (chain progressed or shutting down)
			select {
			case <-ctx.Done():
				return fmt.Errorf("context canceled during recovery: %v", ctx.Err())
			default:
			}

			logging.Logger.Info("[dkg_recovery] retrying share collection",
				zap.Int("attempt", attempt+1),
				zap.Int64("mb_num", mb.MagicBlockNumber),
				zap.Duration("elapsed", time.Since(startTime)))
			time.Sleep(retryDelay)
			// Reset DKG for retry — need fresh state for AddSecretShare
			newDKG = bls.MakeDKGSeeded(mb.T, mb.N, selfKey, seed)
			newDKG.MagicBlockNumber = mb.MagicBlockNumber
			newDKG.StartingRound = mb.StartingRound
			if err := newDKG.AddSecretShare(selfPartyID, selfShare.GetHexString(), false); err != nil {
				return fmt.Errorf("self share add failed on retry: %v", err)
			}
			collectedMPKs = map[string][]string{selfKey: selfMPKsHex}
		}

		var (
			received    int = 1 // self-share already added
			failedPeers []string
		)

		for minerID := range miners {
			if minerID == selfKey {
				continue
			}
			// Skip excluded miners — they are known to be absent
			if excludeSet[minerID] {
				continue
			}

			n := node.GetNode(minerID)
			if n == nil {
				logging.Logger.Warn("[dkg_recovery] peer not found",
					zap.String("miner", minerID[:8]))
				failedPeers = append(failedPeers, minerID[:8])
				continue
			}

			result, reqErr := mc.requestRecoveryShare(ctx, n, mb.MagicBlockNumber, selfKey)
			if reqErr != nil {
				logging.Logger.Warn("[dkg_recovery] peer share request failed",
					zap.String("miner", minerID[:8]),
					zap.Error(reqErr))
				failedPeers = append(failedPeers, minerID[:8])
				continue
			}

			peerPartyID := bls.ComputeIDdkg(minerID)
			if err := newDKG.AddSecretShare(peerPartyID, result.share, false); err != nil {
				logging.Logger.Warn("[dkg_recovery] failed to add peer share",
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

		logging.Logger.Info("[dkg_recovery] share collection status",
			zap.Int("attempt", attempt+1),
			zap.Int("received", received),
			zap.Int("required_n", requiredN),
			zap.Int("total_miners", mb.N),
			zap.Int("excluded", len(excludeMiners)),
			zap.Int("mpks_collected", len(collectedMPKs)),
			zap.Int("failed", len(failedPeers)),
			zap.Duration("elapsed", time.Since(startTime)))

		// Need all N miners' shares minus excluded emergency miners.
		if received >= requiredN {
			break
		}
	}

	// Step 4: Aggregate secret shares
	newDKG.AggregateSecretKeyShares()
	newDKG.Pi = newDKG.Si.GetPublicKey()

	// Step 5: Check if MB has VRF-seeded or CSPRNG MPKs by comparing
	// the collected VRF-seeded MPKs against the MB's existing MPKs.
	// Both are keyed by miner node ID strings.
	mbIsVRFSeeded := false
	if mb.Mpks != nil {
		mbIsVRFSeeded = mpkMapsMatch(mb.Mpks.Mpks, collectedMPKs)
	}

	logging.Logger.Info("[dkg_recovery] MB MPK comparison",
		zap.Int64("mb_num", mb.MagicBlockNumber),
		zap.Bool("mb_is_vrf_seeded", mbIsVRFSeeded))

	// Step 6: Check if chain is active or stuck.
	lfb := mc.GetLatestFinalizedBlock()
	chainStuck := false
	if lfb != nil {
		lfbAge := time.Since(time.Unix(int64(lfb.CreationDate), 0))
		chainStuck = lfbAge >= 3*time.Minute
		if chainStuck {
			logging.Logger.Info("[dkg_recovery] chain appears stuck",
				zap.Int64("mb_num", mb.MagicBlockNumber),
				zap.Int64("lfb_round", lfb.Round),
				zap.Duration("lfb_age", lfbAge))
		}
	} else {
		chainStuck = true // no LFB = definitely stuck
	}

	// Step 7: Decide action based on VRF-seeded status and chain state.
	//
	// VRF-seeded MB + chain active  → validate Pi against MB MPKs → done
	// VRF-seeded MB + chain stuck   → validate Pi against MB MPKs → done (already VRF)
	// CSPRNG MB + chain active       → return error (keep retrying until chain stuck)
	// CSPRNG MB + chain stuck        → force-recover to VRF-seeded MPKs
	forceRecovery := false
	if mbIsVRFSeeded {
		// MB already has VRF-seeded MPKs — validate Pi against them.
		mpks, mpkErr := mb.Mpks.GetMpkMap()
		if mpkErr != nil {
			return fmt.Errorf("failed to parse MB MPKs: %v", mpkErr)
		}
		if err := newDKG.AggregatePublicKeyShares(mpks); err != nil {
			return fmt.Errorf("failed to aggregate MB public keys: %v", err)
		}
		newDKG.SetMpksMap(mb.Mpks.GetMpkMapStrings())
		expectedPi := newDKG.GetPublicKeyByID(selfPartyID)
		if !newDKG.Pi.IsEqual(&expectedPi) {
			return fmt.Errorf("Pi mismatch against VRF-seeded MB MPKs (expected %s, got %s)",
				expectedPi.GetHexString()[:16], newDKG.Pi.GetHexString()[:16])
		}
		logging.Logger.Info("[dkg_recovery] Pi validated against VRF-seeded MB MPKs",
			zap.Int64("mb_num", mb.MagicBlockNumber))
	} else if !chainStuck {
		// CSPRNG MB but chain is active — can't force-recover without all miners restarting.
		// Return error to keep retrying via scheduleVRFRecovery.
		return fmt.Errorf("MB#%d has CSPRNG MPKs but chain is active — retrying",
			mb.MagicBlockNumber)
	} else {
		// CSPRNG MB + chain stuck — force-recover all miners to VRF-seeded MPKs.
		forceRecovery = true
		if len(collectedMPKs) < requiredN {
			return fmt.Errorf("force recovery needs MPKs from %d miners, got %d",
				requiredN, len(collectedMPKs))
		}

		// Build new Mpks from collected VRF-seeded MPKs
		newMpks := block.NewMpks()
		for minerID, mpkHex := range collectedMPKs {
			newMpks.Mpks[minerID] = &block.MPK{
				ID:  minerID,
				Mpk: mpkHex,
			}
		}

		// Validate Pi against the new VRF-seeded MPKs
		newMpkMap, err := newMpks.GetMpkMap()
		if err != nil {
			return fmt.Errorf("failed to parse new VRF-seeded MPKs: %v", err)
		}

		if err := newDKG.AggregatePublicKeyShares(newMpkMap); err != nil {
			return fmt.Errorf("failed to aggregate VRF-seeded public keys: %v", err)
		}

		mpkStrings := newMpks.GetMpkMapStrings()
		newDKG.SetMpksMap(mpkStrings)

		expectedPi := newDKG.GetPublicKeyByID(selfPartyID)
		if !newDKG.Pi.IsEqual(&expectedPi) {
			return fmt.Errorf("Pi mismatch even with VRF-seeded MPKs (expected %s, got %s)",
				expectedPi.GetHexString()[:16], newDKG.Pi.GetHexString()[:16])
		}

		// Update MB's MPKs to VRF-seeded ones
		mb.Mpks = newMpks
		logging.Logger.Info("[dkg_recovery] force recovery - updated MB MPKs to VRF-seeded",
			zap.Int64("mb_num", mb.MagicBlockNumber),
			zap.Int("mpks_count", len(newMpks.Mpks)))

		// Persist the updated MB
		if err := StoreMagicBlock(ctx, mb); err != nil {
			logging.Logger.Error("[dkg_recovery] failed to persist updated MB",
				zap.Int64("mb_num", mb.MagicBlockNumber),
				zap.Error(err))
			// Continue — DKG is still valid even if MB persistence fails
		}
	}

	// Step 8: Store recovered DKG summary
	summary := newDKG.GetDKGSummary()
	summary.IsFinalized = true

	if err := StoreDKGSummary(ctx, summary); err != nil {
		return fmt.Errorf("failed to store recovered DKG: %v", err)
	}

	// Step 9: Set DKG in chain's round storage
	if err := mc.SetDKG(newDKG); err != nil {
		return fmt.Errorf("failed to set recovered DKG: %v", err)
	}

	logging.Logger.Info("[dkg_recovery] recovery complete",
		zap.Int64("mb_num", mb.MagicBlockNumber),
		zap.Int64("mb_sr", mb.StartingRound),
		zap.String("pi", newDKG.Pi.GetHexString()[:16]),
		zap.Bool("force_recovery", forceRecovery),
		zap.Duration("duration", time.Since(startTime)))

	return nil
}

// mpkMapsMatch compares an MB's Mpks map (minerID -> *MPK) against
// a collected VRF-seeded MPKs map (minerID -> []string hex).
// Returns true if both maps have the same keys with the same MPK values.
func mpkMapsMatch(mbMpks map[string]*block.MPK, collected map[string][]string) bool {
	if len(mbMpks) != len(collected) {
		return false
	}
	for k, mpk := range mbMpks {
		coll, ok := collected[k]
		if !ok || len(mpk.Mpk) != len(coll) {
			return false
		}
		for i := range mpk.Mpk {
			if mpk.Mpk[i] != coll[i] {
				return false
			}
		}
	}
	return true
}

// recoveryShareResult holds the share and MPKs from a peer recovery response.
type recoveryShareResult struct {
	share   string
	mpksHex []string
}

// requestRecoveryShare sends a recovery share request to a peer miner.
// Returns the share hex and the peer's VRF-seeded MPKs (for force recovery).
func (mc *Chain) requestRecoveryShare(ctx context.Context, peer *node.Node,
	mbNum int64, selfKey string) (*recoveryShareResult, error) {

	params := &url.Values{}
	params.Add("mb_num", strconv.FormatInt(mbNum, 10))
	params.Add("requester_id", selfKey)

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
		expectedMessage := encryption.Hash(share.Share + strconv.FormatInt(mbNum, 10))
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
