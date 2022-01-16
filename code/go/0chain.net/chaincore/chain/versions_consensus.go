package chain

import (
	"sync"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"github.com/blang/semver/v4"
	"go.uber.org/zap"
)

type versionUpgrader interface {
	Add(id datastore.Key, v semver.Version) error
	GetConsensusVersion() *semver.Version
	UpdateNodesList(nodes map[datastore.Key]struct{})
}

type versionsConsensus struct {
	mutex sync.RWMutex
	nodes map[datastore.Key]struct{}

	versions         map[datastore.Key]semver.Version
	thresholdPercent int
}

func newVersionsConsensus(nodes map[datastore.Key]struct{}, thresholdPercent int) *versionsConsensus {
	vc := &versionsConsensus{
		nodes:            nodes,
		versions:         make(map[datastore.Key]semver.Version),
		thresholdPercent: thresholdPercent,
	}

	return vc
}

func (vc *versionsConsensus) Add(id datastore.Key, v semver.Version) error {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()
	// check if the id exist in magic block
	if _, ok := vc.nodes[id]; !ok {
		return common.NewErrorf("node_not_exist_in_mb",
			"node does not exist in magic block, id: %v, nodes num: %d", id, len(vc.nodes))
	}
	vc.versions[id] = v
	logging.Logger.Debug("add version", zap.String("id", id), zap.String("version", v.String()))
	return nil
}

func (vc *versionsConsensus) GetConsensusVersion() *semver.Version {
	vc.mutex.RLock()
	defer vc.mutex.RUnlock()
	// TODO: make sure the threshold is correct
	threshold := vc.thresholdPercent * len(vc.nodes) / 100

	if len(vc.versions) < threshold {
		logging.Logger.Debug("versions_consensus - versions num not reached threshold",
			zap.Int("num", len(vc.versions)),
			zap.Int("threshold", threshold))
		return nil
	}

	vCounts := make(map[string]int)
	var maxVoted = struct {
		v    semver.Version
		vote int
	}{}

	for _, vv := range vc.versions {
		s := vv.String()
		if _, ok := vCounts[s]; !ok {
			vCounts[s] = 1
			continue
		}

		vCounts[s]++
		if maxVoted.v.EQ(vv) {
			maxVoted.vote = vCounts[s]
			continue
		}

		if maxVoted.vote < vCounts[s] {
			maxVoted.v = vv
			maxVoted.vote = vCounts[s]
		}
	}

	if maxVoted.vote >= threshold {
		logging.Logger.Debug("versions_consensus - version reached threshold",
			zap.Int("threshold", threshold),
			zap.Int("voted", maxVoted.vote))
		return &maxVoted.v
	}

	logging.Logger.Debug("versions_consensus - version not reached threshold",
		zap.Int("threshold", threshold),
		zap.Int("voted", maxVoted.vote))
	return nil
}

func (vc *versionsConsensus) UpdateNodesList(nodes map[datastore.Key]struct{}) {
	logging.Logger.Debug("versions consensus - update magic block nodes list", zap.Int("num", len(nodes)))
	vc.mutex.Lock()
	defer vc.mutex.Unlock()
	// add new nodes if any
	for id := range nodes {
		if _, ok := vc.nodes[id]; !ok {
			vc.nodes[id] = struct{}{}
		}
	}

	// remove nodes that do not exist in nodes list anymore
	for id := range vc.nodes {
		if _, ok := nodes[id]; !ok {
			vc.remove(id)
		}
	}
}

func (vc *versionsConsensus) remove(id string) {
	delete(vc.nodes, id)
	delete(vc.versions, id)
}
