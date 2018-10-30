package node

import (
	"encoding/hex"
	"sort"

	"0chain.net/encryption"
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
	nodes := make([]*Score, np.Size())
	for idx, nd := range np.Nodes {
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
	return true
}
