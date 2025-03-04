package minersc

import (
	"math"
)

//go:generate msgp -io=false -tests=false -v

// LightNode represents a lightweight node with only key and publicKey
type LightNode struct {
	Key       string `json:"key" msg:"k"`
	PublicKey string `json:"public_key" msg:"p"`
}

// DKGMinerNodesV2 is a lighter version of DKGMinerNodes that stores detailed node info in state
type DKGMinerNodesV2 struct {
	MinN     int     `json:"min_n" msg:"mn"`
	MaxN     int     `json:"max_n" msg:"mx"`
	TPercent float64 `json:"t_percent" msg:"tp"`
	KPercent float64 `json:"k_percent" msg:"kp"`

	Nodes          []LightNode     `json:"nodes" msg:"nd"`
	T              int             `json:"t" msg:"t"`
	K              int             `json:"k" msg:"k"`
	N              int             `json:"n" msg:"n"`
	XPercent       float64         `json:"x_percent" msg:"xp"`
	RevealedShares map[string]int  `json:"revealed_shares" msg:"rs"`
	Waited         map[string]bool `json:"waited" msg:"w"`

	// StartRound used to filter responses from old MB where sharders comes up.
	StartRound int64 `json:"start_round" msg:"sr"`
}

// NewDKGMinerNodesV2 creates a new instance of DKGMinerNodesV2
func NewDKGMinerNodesV2() *DKGMinerNodesV2 {
	return &DKGMinerNodesV2{
		Nodes:          make([]LightNode, 0),
		RevealedShares: make(map[string]int),
		Waited:         make(map[string]bool),
	}
}

// setConfigs sets the configuration parameters from GlobalNode
func (dkgmn *DKGMinerNodesV2) setConfigs(gn *GlobalNode) {
	gnb := gn.MustBase()
	dkgmn.MinN = gnb.MinN
	dkgmn.MaxN = gnb.MaxN
	dkgmn.TPercent = gnb.TPercent
	dkgmn.KPercent = gnb.KPercent
	dkgmn.XPercent = gnb.XPercent
}

// calculateTKN calculates T, K, and N values based on the number of nodes
func (dkgmn *DKGMinerNodesV2) calculateTKN(gn *GlobalNode, n int) {
	dkgmn.setConfigs(gn)
	var m = min(dkgmn.MaxN, n)
	dkgmn.N = m
	dkgmn.K = int(math.Ceil(dkgmn.KPercent * float64(m)))
	dkgmn.T = int(math.Ceil(dkgmn.TPercent * float64(m)))
}

func (dkgmn *DKGMinerNodesV2) HasNode(key string) bool {
	for _, nd := range dkgmn.Nodes {
		if nd.Key == key {
			return true
		}
	}
	return false
}

// DeleteNode deletes a node from the DKGMinerNodesV2
func (dkgmn *DKGMinerNodesV2) DeleteNode(key string) {
	for i, nd := range dkgmn.Nodes {
		if nd.Key == key {
			dkgmn.Nodes = append(dkgmn.Nodes[:i], dkgmn.Nodes[i+1:]...)
			return
		}
	}
}
