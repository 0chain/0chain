package chain

import (
	"sync"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"github.com/blang/semver/v4"
	"go.uber.org/zap"
)

type scVersionsManager interface {
	Set(minerID datastore.Key, v semver.Version) error
	GetConsensusVersion() *semver.Version
	UpdateNodesList(nodes map[datastore.Key]struct{})
}

type scVersions struct {
	mutex sync.RWMutex
	nodes map[datastore.Key]struct{}

	versions         map[datastore.Key]semver.Version
	thresholdPercent int
}

func newSCVersionsManager(nodes map[datastore.Key]struct{}, thresholdPercent int) *scVersions {
	scv := &scVersions{
		nodes:            nodes,
		versions:         make(map[datastore.Key]semver.Version),
		thresholdPercent: thresholdPercent,
	}

	return scv
}

func (scv *scVersions) Set(minerID datastore.Key, v semver.Version) error {
	scv.mutex.Lock()
	defer scv.mutex.Unlock()
	// check if the minerID exist in magic block
	if _, ok := scv.nodes[minerID]; !ok {
		return common.NewErrorf("miner_not_exist_in_mb",
			"miner does not exist in magic block, id: %v, miners num: %d", minerID, len(scv.nodes))
	}
	scv.versions[minerID] = v

	return nil
}

func (scv *scVersions) GetConsensusVersion() *semver.Version {
	scv.mutex.RLock()
	defer scv.mutex.RUnlock()
	// TODO: make sure the threshold is correct
	threshold := scv.thresholdPercent * len(scv.nodes) / 100

	if len(scv.versions) < threshold {
		logging.Logger.Debug("sc_versions - versions num not reached threshold",
			zap.Int("num", len(scv.versions)),
			zap.Int("threshold", threshold))
		return nil
	}

	vcounts := make(map[string]int)
	var maxVoted = struct {
		v    semver.Version
		vote int
	}{}

	for _, vv := range scv.versions {
		s := vv.String()
		if _, ok := vcounts[s]; !ok {
			vcounts[s] = 1
			continue
		}

		vcounts[s]++
		if maxVoted.v.EQ(vv) {
			maxVoted.vote = vcounts[s]
			continue
		}

		if maxVoted.vote < vcounts[s] {
			maxVoted.v = vv
			maxVoted.vote = vcounts[s]
		}
	}

	if maxVoted.vote >= threshold {
		logging.Logger.Debug("new sc version reached threshold",
			zap.Int("threshold", threshold),
			zap.Int("voted", maxVoted.vote))
		return &maxVoted.v
	}

	return nil
}

func (scv *scVersions) UpdateNodesList(nodes map[datastore.Key]struct{}) {
	logging.Logger.Debug("sc versions update magic block nodes list", zap.Int("num", len(nodes)))
	scv.mutex.Lock()
	defer scv.mutex.Unlock()
	// add new nodes if any
	for id := range nodes {
		if _, ok := scv.nodes[id]; !ok {
			scv.nodes[id] = struct{}{}
		}
	}

	// remove nodes that do not exist in nodes list anymore
	for id := range scv.nodes {
		if _, ok := nodes[id]; !ok {
			scv.remove(id)
		}
	}
}

func (scv *scVersions) remove(id string) {
	delete(scv.nodes, id)
	delete(scv.versions, id)
}
