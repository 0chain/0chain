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

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// scheduleVRFRecovery launches RecoverDKG as a background goroutine.
// This is necessary because RecoverDKG loops indefinitely waiting for peers,
// but during startup the HTTP server (which serves recovery shares) hasn't
// started yet. Running synchronously would deadlock all miners.
// The goroutine waits briefly for the HTTP server to start, then begins recovery.
func (mc *Chain) scheduleVRFRecovery(ctx context.Context, mb *block.MagicBlock) {
	go func() {
		// Wait for HTTP server to start before attempting recovery.
		// LoadMagicBlocksAndDKG -> ... -> ListenAndServe() takes a few seconds.
		time.Sleep(30 * time.Second)

		logging.Logger.Info("[dkg_recovery] starting scheduled VRF recovery",
			zap.Int64("mb_num", mb.MagicBlockNumber),
			zap.Int64("mb_sr", mb.StartingRound))

		if err := mc.RecoverDKG(ctx, mb); err != nil {
			logging.Logger.Error("[dkg_recovery] scheduled VRF recovery failed",
				zap.Int64("mb_num", mb.MagicBlockNumber),
				zap.Error(err))
		} else {
			logging.Logger.Info("[dkg_recovery] scheduled VRF recovery succeeded",
				zap.Int64("mb_num", mb.MagicBlockNumber))
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
// Params: mb_num (magic block number), requester_id (requesting miner's node ID)
func RecoverShareRequestHandler(ctx context.Context, r *http.Request) (
	resp interface{}, err error) {

	mc := GetMinerChain()
	if !mc.isHardforkActive("Nyx", mc.GetCurrentRound()) {
		return nil, common.NewError("recover_share", "Nyx hardfork not active")
	}

	mbNumStr := r.FormValue("mb_num")
	requesterID := r.FormValue("requester_id")

	if mbNumStr == "" || requesterID == "" {
		return nil, common.NewError("recover_share", "mb_num and requester_id are required")
	}

	mbNum, err := strconv.ParseInt(mbNumStr, 10, 64)
	if err != nil {
		return nil, common.NewErrorf("recover_share", "invalid mb_num: %v", err)
	}

	// Load the magic block to get T, N, miner list
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

	logging.Logger.Info("[dkg_recovery] starting recovery",
		zap.Int64("mb_num", mb.MagicBlockNumber),
		zap.Int64("mb_sr", mb.StartingRound))

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
			zap.Int("threshold", mb.T),
			zap.Int("total_miners", mb.N),
			zap.Int("mpks_collected", len(collectedMPKs)),
			zap.Int("failed", len(failedPeers)),
			zap.Duration("elapsed", time.Since(startTime)))

		// Need ALL N miners' shares and MPKs, not just threshold T.
		// Force recovery (CSPRNG→VRF transition) requires all N MPKs to compute
		// the correct group public key. With only T shares, we'd have enough for
		// DKG reconstruction but force recovery would fail.
		if received >= mb.N {
			break // all miners' shares and MPKs collected
		}
	}

	// Step 4: Aggregate secret shares
	newDKG.AggregateSecretKeyShares()
	newDKG.Pi = newDKG.Si.GetPublicKey()

	// Step 5: Try validation against existing MB MPKs first
	forceRecovery := false
	if mb.Mpks != nil {
		mpks, mpkErr := mb.Mpks.GetMpkMap()
		if mpkErr == nil {
			if err := newDKG.AggregatePublicKeyShares(mpks); err == nil {
				newDKG.SetMpksMap(mb.Mpks.GetMpkMapStrings())
				expectedPi := newDKG.GetPublicKeyByID(selfPartyID)
				if newDKG.Pi.IsEqual(&expectedPi) {
					logging.Logger.Info("[dkg_recovery] Pi validated against existing MB MPKs",
						zap.Int64("mb_num", mb.MagicBlockNumber))
				} else {
					logging.Logger.Warn("[dkg_recovery] Pi mismatch with existing MB MPKs, attempting force recovery",
						zap.Int64("mb_num", mb.MagicBlockNumber),
						zap.String("got_pi", newDKG.Pi.GetHexString()[:16]),
						zap.String("expected_pi", expectedPi.GetHexString()[:16]))
					forceRecovery = true
				}
			} else {
				forceRecovery = true
			}
		} else {
			forceRecovery = true
		}
	} else {
		forceRecovery = true
	}

	// Step 6: Force recovery - update MB's MPKs to VRF-seeded ones
	// Force recovery requires ALL N miners' MPKs because the group public key
	// is computed from all miners' mpk[0] values. Partial MPKs produce a
	// different group key, causing VRF verification failures.
	if forceRecovery {
		if len(collectedMPKs) < mb.N {
			return fmt.Errorf("force recovery needs MPKs from all %d miners, got %d",
				mb.N, len(collectedMPKs))
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

	// Step 7: Store recovered DKG summary
	summary := newDKG.GetDKGSummary()
	summary.IsFinalized = true

	if err := StoreDKGSummary(ctx, summary); err != nil {
		return fmt.Errorf("failed to store recovered DKG: %v", err)
	}

	// Step 8: Set DKG in chain's round storage
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
