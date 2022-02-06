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
	"0chain.net/core/memorystore"
	"0chain.net/smartcontract/minersc"
)

type Home struct {
	Name              string
	ChainName         string
	ID                string
	PublicKey         string
	BuildTag          string
	StartTime         time.Time
	NodeType          string
	isDevMode         bool
	CurrentMagicBlock *block.MagicBlock
	Miners            []Node
	Sharders          []Node
	HealthSummary     HealthSummary
}

func jsonHome(ctx context.Context) Home {
	sc := GetServerChain()
	selfNode := node.Self.Underlying()
	mb := sc.GetCurrentMagicBlock()
	miners := sc.getNodePool(mb.Miners)
	sharders := sc.getNodePool(mb.Sharders)
	healthSummary := sc.getHealthSummary(ctx)
	return Home{
		Name:              selfNode.GetPseudoName(),
		ChainName:         sc.GetKey(),
		PublicKey:         selfNode.PublicKey,
		BuildTag:          build.BuildTag,
		StartTime:         StartTime,
		NodeType:          node.Self.Underlying().Type.String(),
		isDevMode:         config.Development(),
		CurrentMagicBlock: mb,
		Miners:            miners,
		Sharders:          sharders,
		HealthSummary:     healthSummary,
	}
}

type Node struct {
	Status                      string
	Index                       int
	Rank                        string
	Name                        string
	Host                        string
	Path                        string
	Port                        int
	Sent                        int64
	SendErrors                  int64
	Received                    int64
	LastActiveTime              time.Time
	LargeMessageSendTimeSec     float64
	OptimalLargeMessageSendTime float64
	Description                 string
	BuildTag                    string
	StateMissingNodes           int64
	MinersMedianNetworkTime     time.Duration
	AvgBlockTxns                int
}

func (c *Chain) getNodePool(np *node.Pool) []Node {
	r := c.GetRound(c.GetCurrentRound())
	hasRanks := r != nil && r.HasRandomSeed()
	lfb := c.GetLatestFinalizedBlock()
	nodes := np.CopyNodes()
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].SetIndex < nodes[j].SetIndex
	})
	viewNodes := make([]Node, len(nodes))
	for _, nd := range nodes {
		n := Node{
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
		viewNodes = append(viewNodes, n)
	}
	return viewNodes
}

type HealthSummary struct {
	RoundHealth RoundHealth
	ChainHealth ChainHealth
	InfraHealth InfraHealth
	BlockHealth BlockHealth
}

func (c *Chain) getHealthSummary(ctx context.Context) HealthSummary {
	roundHealth := c.getRoundHealth(ctx)
	chainHealth := c.getChainHealth()
	infraHealth := c.getInfraHealth()
	return HealthSummary{
		RoundHealth: roundHealth,
		ChainHealth: chainHealth,
		InfraHealth: infraHealth,
		BlockHealth: c.getBlocksHealth(ctx),
	}
}

type RoundHealth struct {
	Round         int64
	VRFs          string
	RRS           int64
	Proposals     int
	Notarizations int
	Phase         string
	Shares        int
	VRFThreshold  int
	LFBTicket     int64
	isActive      bool
}

type ChainHealth struct {
	LatestFinalizedRound        int64
	DeterministicFinalizedRound int64
	Rollbacks                   int64
	Timeouts                    int64
	RoundTimeoutCount           int64
	RelatedMB                   int64
	FinalizedMB                 int64
}

type InfraHealth struct {
	GoRoutines            int
	HeapAlloc             uint64
	MissingNodes          *int64
	StateMissingNodes     int64
	RedisCollection       int64
	IsLFBStateComputed    bool
	IsDKGProcessDisabled  bool
	IsLFBStateInitialized bool
	DKGPhase              string
	DKGRestarts           int64
}

type BlockHealth struct {
	Blocks                 []blockName
	BlockHash              string
	NumVerificationTickets int
	Consensus              int
	CRB                    string
	LFMBStartingRound      int64
	LFMBHash               string
}

type blockName struct {
	name  string
	style string
	block *block.Block
}

func (c *Chain) getRoundHealth(ctx context.Context) RoundHealth {
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

	return RoundHealth{
		isActive:      isActive,
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
			lfb                     = c.GetLatestFinalizedBlock()
			seri, err               = c.GetBlockStateNode(lfb, minersc.PhaseKey)
			phase     minersc.Phase = minersc.Unknown
			restarts  int64         = -1
			pn        minersc.PhaseNode
		)
		if err == nil {
			if err = pn.Decode(seri.Encode()); err == nil {
				phase = pn.Phase
				restarts = pn.Restarts
			}
		}
		if !config.DevConfiguration.ViewChange {
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
		RedisCollection:       *redisCollection,
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
	w.Write(data)
}
