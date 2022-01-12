package chain

import (
	"sync"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"github.com/blang/semver/v4"
	"go.uber.org/zap"
)

type scVersionsManager interface {
	Set(minerID datastore.Key, v semver.Version) error
	GetConsensusVersion() *semver.Version
	UpdateMinersList(mb *block.MagicBlock)
}

type scVersions struct {
	mutex  sync.RWMutex
	miners map[datastore.Key]struct{}

	versions         map[datastore.Key]semver.Version
	thresholdPercent int
}

func newSCVersionsManager(mb *block.MagicBlock, thresholdPercent int) *scVersions {
	scv := &scVersions{
		miners:           make(map[datastore.Key]struct{}),
		versions:         make(map[datastore.Key]semver.Version),
		thresholdPercent: thresholdPercent,
	}

	scv.UpdateMinersList(mb)
	return scv
}

func (scv *scVersions) Set(minerID datastore.Key, v semver.Version) error {
	scv.mutex.Lock()
	defer scv.mutex.Unlock()
	// check if the minerID exist in magic block
	if _, ok := scv.miners[minerID]; !ok {
		return common.NewErrorf("miner_not exist in mb",
			"miner does not exist in magic block, id: %v", minerID)
	}

	if ev, ok := scv.versions[minerID]; ok && ev.EQ(v) {
		return nil
	}

	scv.versions[minerID] = v

	return nil
}

func (scv *scVersions) GetConsensusVersion() *semver.Version {
	scv.mutex.RLock()
	defer scv.mutex.RUnlock()
	// TODO: make sure the threshold is correct
	threshold := scv.thresholdPercent * len(scv.miners) / 100

	if len(scv.versions) < threshold {
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

func (scv *scVersions) UpdateMinersList(mb *block.MagicBlock) {
	mbs := mb.Miners.CopyNodesMap()
	scv.mutex.Lock()
	defer scv.mutex.Unlock()
	// add new miners if any
	for id := range mbs {
		if _, ok := scv.miners[id]; !ok {
			scv.miners[id] = struct{}{}
		}
	}

	// remove none exist miners
	for id := range scv.miners {
		if _, ok := mbs[id]; !ok {
			scv.remove(id)
		}
	}
}

func (scv *scVersions) remove(minerID string) {
	delete(scv.miners, minerID)
	delete(scv.versions, minerID)
}
