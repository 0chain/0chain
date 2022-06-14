package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"sort"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/build"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/smartcontract/minersc"
	"go.uber.org/zap"
)

type home struct {
	Name              string            `json:"name"`
	ChainName         string            `json:"chain_name"`
	ID                string            `json:"id"`
	PublicKey         string            `json:"public_key"`
	BuildTag          string            `json:"build_tag"`
	StartTime         time.Time         `json:"start_time"`
	NodeType          string            `json:"node_type"`
	IsDevMode         bool              `json:"is_dev_mode"`
	CurrentMagicBlock *block.MagicBlock `json:"current_magic_block"`
	Miners            []nodeInfo        `json:"miners"`
	Sharders          []nodeInfo        `json:"sharders"`
	HealthSummary     healthSummary     `json:"health_summary"`
}

func jsonHome(ctx context.Context) home {
	sc := GetServerChain()
	selfNode := node.Self.Underlying()
	mb := sc.GetCurrentMagicBlock()
	miners := sc.getNodePool(mb.Miners)
	sharders := sc.getNodePool(mb.Sharders)
	healthSummary := sc.getHealthSummary(ctx)
	return home{
		Name:              selfNode.GetPseudoName(),
		ChainName:         sc.GetKey(),
		PublicKey:         selfNode.PublicKey,
		BuildTag:          build.BuildTag,
		StartTime:         StartTime,
		NodeType:          node.Self.Underlying().Type.String(),
		IsDevMode:         config.Development(),
		CurrentMagicBlock: mb,
		Miners:            miners,
		Sharders:          sharders,
		HealthSummary:     healthSummary,
	}
}

type nodeInfo struct {
	Status                      string        `json:"status"`
	Index                       int           `json:"index"`
	Rank                        string        `json:"rank"`
	Name                        string        `json:"name"`
	Host                        string        `json:"host"`
	Path                        string        `json:"path"`
	Port                        int           `json:"port"`
	Sent                        int64         `json:"sent"`
	SendErrors                  int64         `json:"send_errors"`
	Received                    int64         `json:"received"`
	LastActiveTime              time.Time     `json:"last_active_time"`
	LargeMessageSendTimeSec     float64       `json:"large_message_send_time_sec"`
	OptimalLargeMessageSendTime float64       `json:"optimal_large_message_send_time"`
	Description                 string        `json:"description"`
	BuildTag                    string        `json:"build_tag"`
	StateMissingNodes           int64         `json:"state_missing_nodes"`
	MinersMedianNetworkTime     time.Duration `json:"miners_median_network_time"`
	AvgBlockTxns                int           `json:"avg_block_txns"`
}

func (c *Chain) getNodePool(np *node.Pool) []nodeInfo {
	r := c.GetRound(c.GetCurrentRound())
	hasRanks := r != nil && r.HasRandomSeed()
	lfb := c.GetLatestFinalizedBlock()
	nodes := np.CopyNodes()
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].SetIndex < nodes[j].SetIndex
	})
	viewNodes := make([]nodeInfo, len(nodes))
	for i, nd := range nodes {
		n := nodeInfo{
			Index:                       nd.SetIndex,
			Name:                        nd.GetPseudoName(),
			Sent:                        nd.GetSent(),
			SendErrors:                  nd.GetSendErrors(),
			Received:                    nd.GetReceived(),
			LastActiveTime:              nd.GetLastActiveTime(),
			LargeMessageSendTimeSec:     nd.GetLargeMessageSendTimeSec(),
			OptimalLargeMessageSendTime: nd.GetOptimalLargeMessageSendTime(),
			Description:                 nd.Description,
			BuildTag:                    nd.Info.BuildTag,
			StateMissingNodes:           nd.Info.StateMissingNodes,
			MinersMedianNetworkTime:     nd.Info.MinersMedianNetworkTime,
			AvgBlockTxns:                nd.Info.AvgBlockTxns,
		}

		switch {
		case nd.GetStatus() == node.NodeStatusInactive:
			n.Status = "inactive"
		case node.Self.IsEqual(nd) && c.GetCurrentRound() > lfb.Round+10:
			n.Status = "warning"
		default:
			n.Status = "normal"
		}

		switch {
		case nd.Type == node.NodeTypeMiner && hasRanks && c.IsRoundGenerator(r, nd):
			n.Rank = fmt.Sprintf("%d", r.GetMinerRank(nd))
		case nd.Type == node.NodeTypeSharder && c.IsBlockSharder(lfb, nd):
			n.Rank = "*"
		}
		if !node.Self.IsEqual(nd) {
			n.Host = nd.Host
			n.Port = nd.Port
		}
		viewNodes[i] = n
	}
	return viewNodes
}

type healthSummary struct {
	RoundHealth roundHealth `json:"round_health"`
	ChainHealth ChainHealth `json:"chain_health"`
	InfraHealth InfraHealth `json:"infra_health"`
	BlockHealth BlockHealth `json:"block_health"`
}

func (c *Chain) getHealthSummary(ctx context.Context) healthSummary {
	roundHealth := c.getRoundHealth(ctx)
	chainHealth := c.getChainHealth()
	infraHealth := c.getInfraHealth()
	return healthSummary{
		RoundHealth: roundHealth,
		ChainHealth: chainHealth,
		InfraHealth: infraHealth,
		BlockHealth: c.getBlocksHealth(ctx),
	}
}

type roundHealth struct {
	Round         int64  `json:"round"`
	VRFs          string `json:"vrfs"`
	RRS           int64  `json:"rrs"`
	Proposals     int    `json:"proposals"`
	Notarizations int    `json:"notarizations"`
	Phase         string `json:"phase"`
	Shares        int    `json:"shares"`
	VRFThreshold  int    `json:"vrf_threshold"`
	LFBTicket     int64  `json:"lfb_ticket"`
	IsActive      bool   `json:"is_active"`
}

type ChainHealth struct {
	LatestFinalizedRound        int64 `json:"latest_finalized_round"`
	DeterministicFinalizedRound int64 `json:"determintinistic_finalized_round"`
	Rollbacks                   int64 `json:"rollbacks"`
	Timeouts                    int64 `json:"timeouts"`
	RoundTimeoutCount           int64 `json:"round_timeout_count"`
	RelatedMB                   int64 `json:"related_mb"`
	FinalizedMB                 int64 `json:"finalized_mb"`
}

type InfraHealth struct {
	GoRoutines            int    `json:"go_routines"`
	HeapAlloc             uint64 `json:"heap_alloc"`
	MissingNodes          *int64 `json:"missing_nodes"`
	StateMissingNodes     int64  `json:"state_missing_nodes"`
	RedisCollection       *int64 `json:"redis_collection"`
	IsLFBStateComputed    bool   `json:"is_lfb_state_computed"`
	IsDKGProcessDisabled  bool   `json:"is_dkg_process_disabled"`
	IsLFBStateInitialized bool   `json:"is_lfb_state_initialized"`
	DKGPhase              string `json:"dkg_phase"`
	DKGRestarts           int64  `json:"dkg_restart"`
}

type BlockHealth struct {
	Blocks                 []blockName `json:"blocks"`
	BlockHash              string      `json:"block_hash"`
	NumVerificationTickets int         `json:"num_verification_tickets"`
	Consensus              int         `json:"concensus"`
	CRB                    string      `json:"crb"`
	LFMBStartingRound      int64       `json:"lfmb_starting_round"`
	LFMBHash               string      `json:"lfmb_hash"`
}

type blockName struct {
	Name  string       `json:"name"`
	Style string       `json:"style"`
	Block *block.Block `json:"block"`
}

func (c *Chain) getRoundHealth(ctx context.Context) roundHealth {
	var rn = c.GetCurrentRound()
	cr := c.GetRound(rn)

	vrfMsg := "N/A"
	notarizations := 0
	proposals := 0
	rrs := int64(0)
	phase := "N/A"
	var mb = c.GetMagicBlock(rn)
	vrfThreshold := 0

	if node.Self.Underlying().Type == node.NodeTypeMiner {
		var shares int
		check := "✗"
		if cr != nil {
			shares = len(cr.GetVRFShares())
			notarizations = len(cr.GetNotarizedBlocks())
			proposals = len(cr.GetProposedBlocks())
			rrs = cr.GetRandomSeed()
			phase = round.GetPhaseName(cr.GetPhase())
		}

		vrfThreshold = mb.T
		if shares >= vrfThreshold {
			check = "&#x2714;"
		}
		vrfMsg = fmt.Sprintf("(%v/%v)%s", shares, vrfThreshold, check)
	}

	var (
		crn      = c.GetCurrentRound()
		ahead    = int64(config.GetLFBTicketAhead())
		tk       = c.GetLatestLFBTicket(ctx)
		tkRound  int64
		isActive = true
	)

	if tk != nil {
		tkRound = tk.Round

		if tkRound+ahead <= crn {
			isActive = false
		}
	}

	return roundHealth{
		IsActive:      isActive,
		Round:         rn,
		VRFs:          vrfMsg,
		RRS:           rrs,
		Proposals:     proposals,
		Phase:         phase,
		Notarizations: notarizations,
		VRFThreshold:  vrfThreshold,
		LFBTicket:     tkRound,
	}
}

func (c *Chain) getChainHealth() ChainHealth {
	var rn = c.GetCurrentRound()
	cr := c.GetRound(rn)
	rtoc := c.GetRoundTimeoutCount()
	if cr != nil {
		rtoc = int64(cr.GetTimeoutCount())
	}

	var (
		mb            = c.GetMagicBlock(rn)
		fmb           = c.GetLatestFinalizedMagicBlockRound(rn)
		startingRound int64
	)
	if fmb != nil {
		startingRound = fmb.StartingRound
	}

	return ChainHealth{
		LatestFinalizedRound:        c.GetLatestFinalizedBlock().Round,
		DeterministicFinalizedRound: c.LatestDeterministicBlock.Round,
		Rollbacks:                   c.RollbackCount,
		Timeouts:                    c.RoundTimeoutsCount,
		RoundTimeoutCount:           rtoc,
		RelatedMB:                   mb.StartingRound,
		FinalizedMB:                 startingRound,
	}
}

func (c *Chain) getInfraHealth() InfraHealth {
	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)
	ps := c.GetPruneStats()
	var (
		missingNodes          *int64
		redisCollection       *int64
		isLFBStateComputed    bool
		isLFBStateInitialized bool
		isDKGProcessDisabled  bool
		dkgPhase              string
		dkgRestarts           int64
	)
	if ps != nil {
		missingNodes = &ps.MissingNodes
	}
	snt := node.Self.Underlying().Type
	switch snt {
	case node.NodeTypeMiner:
		txn, ok := transaction.Provider().(*transaction.Transaction)
		if ok {
			transactionEntityMetadata := txn.GetEntityMetadata()
			collectionName := txn.GetCollectionName()
			ctx := common.GetRootContext()
			cctx := memorystore.WithEntityConnection(ctx, transactionEntityMetadata)
			defer memorystore.Close(cctx)
			mstore, ok := transactionEntityMetadata.GetStore().(*memorystore.Store)
			if ok {
				temp := mstore.GetCollectionSize(cctx, transactionEntityMetadata, collectionName)
				redisCollection = &temp
			}
		}
		var lfb = c.GetLatestFinalizedBlock()
		isLFBStateComputed = lfb.IsStateComputed()
		isLFBStateInitialized = lfb.ClientState != nil
	case node.NodeTypeSharder:
		var (
			lfb      = c.GetLatestFinalizedBlock()
			pn       minersc.PhaseNode
			phase    minersc.Phase = minersc.Unknown
			restarts int64         = -1
		)
		err := c.GetBlockStateNode(lfb, minersc.PhaseKey, &pn)
		switch err {
		case nil:
			phase = pn.Phase
			restarts = pn.Restarts
		default:
			logging.Logger.Warn("get block state node failed", zap.Error(err))
		}

		if !c.ChainConfig.IsViewChangeEnabled() {
			isDKGProcessDisabled = true
		} else {
			dkgPhase = phase.String()
			dkgRestarts = restarts
		}
	}

	return InfraHealth{
		GoRoutines:            runtime.NumGoroutine(),
		HeapAlloc:             mstats.HeapAlloc,
		MissingNodes:          missingNodes,
		RedisCollection:       redisCollection,
		IsLFBStateComputed:    isLFBStateComputed,
		IsLFBStateInitialized: isLFBStateInitialized,
		IsDKGProcessDisabled:  isDKGProcessDisabled,
		DKGPhase:              dkgPhase,
		DKGRestarts:           dkgRestarts,
	}
}

func (c *Chain) getBlocksHealth(ctx context.Context) BlockHealth {
	var (
		blocks                 = make([]blockName, 0)
		rn                     = c.GetCurrentRound()
		cr                     = c.GetRound(rn)
		lfb                    = c.GetLatestFinalizedBlock()
		plfb                   = c.GetLocalPreviousBlock(ctx, lfb)
		lfmb                   = c.GetLatestMagicBlock()
		next                   [4]*block.Block // blocks after LFB
		blockHash              string
		numVerificationTickets int
		consensus              int
		bvts                   string
	)

	for i := range next {
		var r = c.GetRound(lfb.Round + 1 + int64(i))
		if r == nil {
			continue // no round, no block
		}
		var hnb = r.GetHeaviestNotarizedBlock()
		if hnb != nil {
			next[i] = hnb // keep the block
			continue
		}
		var pbs = r.GetProposedBlocks()
		if len(pbs) == 0 {
			continue
		}
		next[i] = pbs[0] // use first one
	}
	for i, bn := range []blockName{
		{itoa(lfb.Round - 1), " class='green'", plfb},
		{"LFB", " class='green'", lfb},
		{itoa(lfb.Round + 1), "", next[0]},
		{itoa(lfb.Round + 2), "", next[1]},
		{itoa(lfb.Round + 3), "", next[2]},
		{itoa(lfb.Round + 4), "", next[3]},
	} {
		if i == 5 && node.Self.Underlying().Type == node.NodeTypeMiner {
			continue
		}
		blocks = append(blocks, bn)
	}

	if node.Self.Underlying().Type == node.NodeTypeMiner {
		if cr != nil {
			b := cr.GetBestRankedProposedBlock()
			if b != nil {
				blockHash = b.Hash
				numVerificationTickets = len(b.GetVerificationTickets())
			}
		}
		consensus = int(math.Ceil((float64(config.GetThresholdCount()) / 100) * float64(lfmb.Miners.Size())))
	}

	return BlockHealth{
		CRB:                    bvts,
		Blocks:                 blocks,
		BlockHash:              blockHash,
		NumVerificationTickets: numVerificationTickets,
		Consensus:              consensus,
		LFMBStartingRound:      lfmb.StartingRound,
		LFMBHash:               lfmb.Hash,
	}
}

func JSONHandler(w http.ResponseWriter, r *http.Request) {
	stat := jsonHome(r.Context())
	data, err := json.Marshal(&stat)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if _, err = w.Write(data); err != nil {
		logging.Logger.Error("http write failed",
			zap.String("url", r.URL.String()),
			zap.Error(err))
	}
}
