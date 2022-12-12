package node

import (
	"encoding/hex"
	"sort"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

//Score - a node with a score
type Score struct {
	Node  *Node
	Score int32
}

//PoolScorer - a node pool scorer that ranks the nodes in the pool for a given hash
type PoolScorer interface {
	ScoreHash(np *Pool, hash []byte) []*Score
	ScoreHashString(np *Pool, hash string) []*Score
}

//HashPoolScorer - a pool scorer based on hash scoring
type HashPoolScorer struct {
	HashScorer encryption.HashScorer
}

//NewHashPoolScorer - create a new hash pool scorer
func NewHashPoolScorer(hs encryption.HashScorer) *HashPoolScorer {
	return &HashPoolScorer{HashScorer: hs}
}

//ScoreHash - implement interface
func (hps *HashPoolScorer) ScoreHash(np *Pool, hash []byte) []*Score {
	npNodes := np.CopyNodes()
	nodes := make([]*Score, len(npNodes))
	for idx, nd := range npNodes {
		nodes[idx] = &Score{}
		nodes[idx].Node = nd
		nodes[idx].Score = hps.HashScorer.Score(nd.idBytes, hash)
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Score == nodes[j].Score {
			return nodes[i].Node.SetIndex > nodes[j].Node.SetIndex
		}
		return nodes[i].Score > nodes[j].Score
	})
	return nodes
}

//ScoreHashString - implement interface
func (hps *HashPoolScorer) ScoreHashString(np *Pool, hash string) []*Score {
	hBytes, err := hex.DecodeString(hash)
	if err != nil {
		logging.Logger.Info("decode failed for hash", zap.String("hash", hash), zap.Error(err))
		return nil
	}
	return hps.ScoreHash(np, hBytes)
}

//IsInTop - checks if a node is in the top N
func (n *Node) IsInTop(nodeScores []*Score, topN int) bool {
	if topN <= len(nodeScores) {
		minScore := nodeScores[topN-1].Score
		for _, ns := range nodeScores {
			if ns.Score < minScore {
				return false
			}
			if ns.Node == n {
				return true
			}
		}
	}
	return false
}

// IsInTopWithNodes gets all the nodes in topN.
func (n *Node) IsInTopWithNodes(nodeScores []*Score, topN int) (bool, []*Node) {
	nodes := make([]*Node, 0, 1)
	inTop := false
	if topN <= len(nodeScores) {
		minScore := nodeScores[topN-1].Score
		//nodeScores are in descending order
		for _, ns := range nodeScores {
			if ns.Score < minScore {
				//we've found all the nodes in topN
				break
			}
			nodes = append(nodes, ns.Node)
			if ns.Node == n {
				inTop = true
			}
		}
	}
	return inTop, nodes
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// GetTopNNodes - get the top n nodes from the sorted scores.
func GetTopNNodes(scores []*Score, topN int) (nodes []*Node) {

	var n = min(topN, len(scores))

	nodes = make([]*Node, 0, n)
	for i := 0; i < n; i++ {
		nodes = append(nodes, scores[i].Node)
	}

	return
}
