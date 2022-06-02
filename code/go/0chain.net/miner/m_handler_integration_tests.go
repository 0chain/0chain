//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/conductor/config/cases"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/gosdk/core/util"
)

// SetupX2MResponders - setup responders.
func SetupX2MResponders() {
	handlers := x2mRespondersMap()
	handlers[getNotarizedBlockX2MV1Pattern] = chain.BlockStats(
		handlers[getNotarizedBlockX2MV1Pattern],
		chain.BlockStatsConfigurator{
			HashKey:      "block",
			SenderHeader: node.HeaderNodeID,
		},
	)
	setupHandlers(handlers)
}

// SetupM2MReceivers - setup receivers for miner to miner communication.
func SetupM2MReceivers(c node.Chainer) {
	handlers := x2mReceiversMap(c)
	handlers[vrfsShareRoundM2MV1Pattern] = common.N2NRateLimit(
		node.ToN2NReceiveEntityHandler(
			VRFSStats(VRFShareHandler),
			nil,
		),
	)
	setupHandlers(handlers)
}

// VRFSStats represents middleware for datastore.JSONEntityReqResponderF handlers.
// Collects vrfs requests stats.
func VRFSStats(handler datastore.JSONEntityReqResponderF) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		if !crpc.Client().State().ServerStatsCollectorEnabled {
			return handler(ctx, entity)
		}

		vrfs, ok := entity.(*round.VRFShare)
		if !ok {
			log.Panicf("Conductor: unexpected entity type is provided")
		}

		ss := &stats.VRFSRequest{
			NodeID:   node.Self.ID,
			Round:    vrfs.Round,
			SenderID: node.GetSender(ctx).GetKey(),
		}
		if err := crpc.Client().AddVRFSServerStats(ss); err != nil {
			log.Panicf("Conductor: error while adding server stats: %v", err)
		}

		return handler(ctx, entity)
	}
}

var (
	waitForSpammingVRF      chan bool
	waitForSpammingVRFCount int32
)

// NotarizationReceiptHandler - handles the receipt of a notarization
// for a block.
func NotarizationReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	not, ok := entity.(*Notarization)
	if !ok {
		log.Panicf("unexpected type")
	}

	if isDelayingBlock(not.Round) {
		go func() {
			<-delayedBlock
		}()
	}

	state := crpc.Client().State()

	if state.RoundHasFinalized != nil && state.RoundHasFinalized.Round == int(not.Round) && chain.IsSpamReceiver(state, not.Round) {
		m := GetMinerChain()
		mr := m.getOrCreateRound(ctx, (int64)(state.RoundHasFinalized.Round+1))
		// if already received VRF share of next round the miner does not need to wait for the spamming VRF share
		if len(mr.vrfSharesCache.getAll()) == 0 {
			logging.Logger.Sugar().Debugf("Waiting for spamming VRF")
			if waitForSpammingVRF == nil {
				waitForSpammingVRF = make(chan bool)
			}
			atomic.AddInt32(&waitForSpammingVRFCount, 1)
			<-waitForSpammingVRF
		}
	}

	return notarizationReceiptHandler(ctx, entity)
}

// NotarizedBlockSendHandler - handles a request for a notarized block.
func NotarizedBlockSendHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	var (
		minerInformer = createMinerInformer(r)
		requestorID   = r.Header.Get(node.HeaderNodeID)
	)

	if isPreparingStateForFBRequestorTestCase(minerInformer, requestorID) {
		return nil, errors.New("conductor expected error")
	}

	cfg := crpc.Client().State().MinerNotarisedBlockRequestor
	if cfg == nil {
		return notarizedBlockSendHandler(ctx, r)
	}

	selfInfo := cases.SelfInfo{
		IsSharder: node.Self.Type == node.NodeTypeSharder,
		ID:        node.Self.ID,
		SetIndex:  node.Self.SetIndex,
	}

	cfg.Lock()
	defer cfg.Unlock()

	switch {
	case cfg.IgnoringRequestsBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo) && cfg.Ignored < 1:
		cfg.Ignored++
		return nil, fmt.Errorf("%w: conductor expected error", common.ErrInternal)

	case cfg.ValidBlockWithChangedHashBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return validBlockWithChangedHash(r)

	case cfg.InvalidBlockWithChangedHashBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return invalidBlockWithChangedHash(r)

	case cfg.BlockWithoutVerTicketsBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		return blockWithoutVerTickets(r)

	case cfg.CorrectResponseBy.IsActingOnTestRequestor(minerInformer, requestorID, cfg.OnRound, selfInfo):
		fallthrough

	default:
		return notarizedBlockSendHandler(ctx, r)
	}
}

func isPreparingStateForFBRequestorTestCase(mi *cases.MinerInformer, requestorID string) bool {
	cfg := crpc.Client().State().FBRequestor
	if cfg == nil || mi == nil || mi.GetRoundNumber() != cfg.OnRound {
		return false
	}
	return !mi.IsGenerator(requestorID) && mi.GetTypeRank(requestorID) == 0 // Replica0
}

func createMinerInformer(r *http.Request) *cases.MinerInformer {
	mChain := GetMinerChain()
	bl, err := getNotarizedBlock(context.Background(), r)
	if err != nil {
		return nil
	}
	miners := mChain.GetMiners(bl.Round)

	roundI := round.NewRound(bl.Round)
	roundI.SetRandomSeed(bl.RoundRandomSeed, len(miners.Nodes))

	return cases.NewMinerInformer(
		chain.NewRanker(roundI, miners),
		mChain.GetGeneratorsNum(),
	)
}

func validBlockWithChangedHash(r *http.Request) (*block.Block, error) {
	bl, err := getNotarizedBlock(context.Background(), r)
	if err != nil {
		log.Panicf("Conductor: error while getting notarised block: %v", err)
	}

	bl.CreationDate++
	bl.HashBlock()
	if bl.MinerID != node.Self.ID {
		log.Printf("miner id is unexpected, block miner %s, self %s", bl.MinerID, node.Self.ID)
	}
	if bl.Signature, err = node.Self.Sign(bl.Hash); err != nil {
		log.Panicf("Conductor: error while signing block: %v", err)
	}
	return bl, nil
}

func invalidBlockWithChangedHash(r *http.Request) (*block.Block, error) {
	bl, err := getNotarizedBlock(context.TODO(), r)
	if err != nil {
		log.Panicf("Conductor: error while getting notarised block: %v", err)
	}

	bl.Hash = util.Hash("invalid hash")
	return bl, nil
}

func blockWithoutVerTickets(r *http.Request) (*block.Block, error) {
	bl, err := getNotarizedBlock(context.TODO(), r)
	if err != nil {
		log.Panicf("Conductor: error while getting notarised block: %v", err)
	}

	bl.VerificationTickets = nil

	return bl, nil
}

// VRFShareHandler - handle the vrf share.
func VRFShareHandler(ctx context.Context, entity datastore.Entity) (
	interface{}, error) {
	return vrfShareHandler(ctx, entity)
}
