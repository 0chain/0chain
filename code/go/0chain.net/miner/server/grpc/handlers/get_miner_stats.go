package handlers

import (
	"context"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/miner"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

// GetMinerStats returns the stats of the miner.
func (m *minerGRPCService) GetMinerStats(ctx context.Context, req *minerproto.GetMinerStatsRequest) (*minerproto.GetMinerStatsResponse, error) {
	c := miner.GetMinerChain().Chain
	var total int64
	ms := node.Self.Underlying().ProtocolStats.(*chain.MinerStats)
	for i := 0; i < c.GetGeneratorsNum(); i++ {
		total += ms.FinalizationCountByRank[i]
	}
	cr := c.GetRound(c.GetCurrentRound())
	rtoc := c.GetRoundTimeoutCount()
	if cr != nil {
		rtoc = int64(cr.GetTimeoutCount())
	}
	networkTimes := make(map[string]*durationpb.Duration)
	mb := c.GetCurrentMagicBlock()
	for k, v := range mb.Miners.CopyNodesMap() {
		durationpb.New(v.Info.MinersMedianNetworkTime)
		networkTimes[k] = durationpb.New(v.Info.MinersMedianNetworkTime)
	}
	for k, v := range mb.Sharders.CopyNodesMap() {
		networkTimes[k] = durationpb.New(v.Info.MinersMedianNetworkTime)
	}

	return &minerproto.GetMinerStatsResponse{
		ExplorerStats: &minerproto.ExplorerStats{
			BlockFinality:      chain.SteadyStateFinalizationTimer.Mean() / 1000000.0,
			LastFinalizedRound: c.GetLatestFinalizedBlock().Round,
			BlocksFinalized:    total,
			StateHealth:        node.Self.Underlying().Info.StateMissingNodes,
			CurrentRound:       c.GetCurrentRound(),
			RoundTimeout:       rtoc,
			Timeouts:           c.RoundTimeoutsCount,
			AverageBlockSize:   int32(node.Self.Underlying().Info.AvgBlockTxns),
			NetworkTime:        networkTimes,
		},
	}, nil
}
