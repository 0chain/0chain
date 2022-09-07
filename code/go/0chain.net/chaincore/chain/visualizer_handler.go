package chain

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"

	"0chain.net/chaincore/block"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/chaincore/node"
)

type bNode struct {
	ID                 string  `json:"id"`
	PrevID             string  `json:"prev_id"`
	Round              int64   `json:"round"`
	Rank               int     `json:"rank"`
	GeneratorID        int     `json:"generator_id"`
	GeneratorName      string  `json:"generator_name"`
	Weight             float64 `json:"chain_weight"`
	Verifications      int     `json:"verifications"`
	Verified           bool    `json:"verified"`
	VerificationFailed bool    `json:"verification_failed"`
	Notarized          bool    `json:"notarized"`
	Finalized          bool    `json:"finalized"`
	X                  int     `json:"x"`
	Y                  int     `json:"y"`
	Size               int     `json:"size"`
}

//WIPBlockChainHandler - all the blocks in the memory useful to visualize and debug
func (c *Chain) WIPBlockChainHandler(w http.ResponseWriter, r *http.Request) {
	bl := c.getBlocks()
	var minr int64 = math.MaxInt64
	var maxr int64
	for _, b := range bl {
		if b.Round < minr {
			minr = b.Round
		}
		if b.Round > maxr {
			maxr = b.Round
		}
	}
	if minr < maxr-12 {
		minr = maxr - 12
	}
	if minr <= 0 {
		minr = 1
	}
	sort.SliceStable(bl, func(i, j int) bool {
		if bl[i].Round == bl[j].Round {
			return bl[i].RoundRank < bl[j].RoundRank
		}
		return bl[i].Round < bl[j].Round
	})
	finzalizedBlocks := make(map[string]bool)
	for fb := c.GetLatestFinalizedBlock(); fb != nil; fb = fb.PrevBlock {
		finzalizedBlocks[fb.Hash] = true
	}
	bNodes := make([]*bNode, 0, len(bl))
	radius := 3
	padding := 5
	numGenerators := c.GetGeneratorsNum()
	DXR := numGenerators*radius + padding
	DYR := DXR
	for _, b := range bl {
		if b.Round < minr {
			continue
		}
		miner := node.GetNode(b.MinerID)
		x := int(b.Round - minr)
		y := miner.SetIndex
		_, finalized := finzalizedBlocks[b.Hash]
		bNd := &bNode{
			ID:                 b.Hash,
			PrevID:             b.PrevHash,
			Round:              b.Round,
			Rank:               b.RoundRank,
			GeneratorID:        miner.SetIndex,
			GeneratorName:      miner.Description,
			Weight:             b.Weight(),
			Verifications:      b.VerificationTicketsSize(),
			Verified:           b.GetVerificationStatus() != block.VerificationPending,
			VerificationFailed: b.GetVerificationStatus() == block.VerificationFailed,
			Notarized:          b.IsBlockNotarized(),
			Finalized:          finalized,
			X:                  x*DXR*2 + DXR,
			Y:                  y*DYR*2 + DYR,
			Size:               6 * (numGenerators - b.RoundRank),
		}
		bNodes = append(bNodes, bNd)
	}
	//TODO: make CORS more restrictive
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bNodes); err != nil {
		logging.Logger.Error("http write json failed", zap.Error(err))
	}
}
