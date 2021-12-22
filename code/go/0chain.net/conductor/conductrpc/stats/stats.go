package stats

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
)

type (
	// NodesServerStats represents struct with maps containing
	// needed nodes server stats.
	NodesServerStats struct {
		blockMu sync.Mutex

		// Block represents map or storing fetching block stats.
		// minerID -> BlockInfos
		Block map[string]BlockInfos
	}

	// BlockInfos represents a map:
	// 	handler -> BlockInfo -> counter (number of requests with BlockInfo parameters)
	BlockInfos map[string]map[BlockInfo]int

	// BlockInfo contains individual parameters of block's requests.
	BlockInfo struct {
		Hash  string `json:"hash"`
		Round int    `json:"round"`
	}

	// BlockReport represents struct for collecting reports from the nodes
	// about handled block's requests.
	BlockReport struct {
		NodeID    string    `json:"miner_id"`
		BlockInfo BlockInfo `json:"block_info"`
		Handler   string    `json:"path"`
	}
)

// NewNodeServerStats creates initialized NodesServerStats.
func NewNodeServerStats() *NodesServerStats {
	return &NodesServerStats{
		Block: make(map[string]BlockInfos),
	}
}

// AddBlockStats takes needed info from the BlockReport and inserts it to the NodesServerStats.Block map.
func (nss *NodesServerStats) AddBlockStats(rep *BlockReport) {
	nss.blockMu.Lock()
	defer nss.blockMu.Unlock()

	_, ok := nss.Block[rep.NodeID]
	if !ok {
		nss.Block[rep.NodeID] = make(BlockInfos)
	}
	_, ok = nss.Block[rep.NodeID][rep.Handler]
	if !ok {
		nss.Block[rep.NodeID][rep.Handler] = make(map[BlockInfo]int)
	}
	nss.Block[rep.NodeID][rep.Handler][rep.BlockInfo]++
}

// ContainsHashOrRound looks for BlockInfo with provided hash and round or individually hash or round.
// Return BlockReport with found hash, round and handler.
func (bi BlockInfos) ContainsHashOrRound(hash string, round int) (bool, BlockReport) {
	for path, stats := range bi {
		var (
			hashAndRoundBI = BlockInfo{
				Hash:  hash,
				Round: round,
			}
			hashBI = BlockInfo{
				Hash: hash,
			}
			roundBI = BlockInfo{
				Round: round,
			}
		)
		_, containsHR := stats[hashAndRoundBI]
		_, containsH := stats[hashBI]
		_, containsR := stats[roundBI]

		switch {
		case containsHR:
			return true, BlockReport{BlockInfo: hashAndRoundBI, Handler: path}

		case containsH:
			return true, BlockReport{BlockInfo: hashBI, Handler: path}

		case containsR:
			return true, BlockReport{BlockInfo: roundBI, Handler: path}
		}
	}

	return false, BlockReport{}
}

// String return BlockReport as readable string.
func (br *BlockReport) String() string {
	round := "empty"
	if br.BlockInfo.Round != 0 {
		round = strconv.Itoa(br.BlockInfo.Round)
	}
	hash := "empty"
	if br.BlockInfo.Hash != "" {
		hash = br.BlockInfo.Hash
	}
	nodeID := "empty"
	if br.NodeID != "" {
		nodeID = br.NodeID
	}
	path := "empty"
	if br.Handler != "" {
		path = br.Handler
	}
	return fmt.Sprintf("round: %s; hash: %s; path: %s; nodeID: %s", round, hash, path, nodeID)
}

// Encode encodes BlockReport to the bytes.
func (br *BlockReport) Encode() ([]byte, error) {
	return json.Marshal(br)
}

// Decode decodes BlockReport from the bytes.
func (br *BlockReport) Decode(blob []byte) error {
	return json.Unmarshal(blob, br)
}
