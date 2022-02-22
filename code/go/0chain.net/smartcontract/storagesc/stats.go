package storagesc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

func (sc *StorageSmartContract) newWrite(statectx c_state.StateContextI, writeSize int64) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	switch err {
	case nil, util.ErrValueNotPresent:
		stats.Stats.NumWrites++
		stats.Stats.UsedSize += writeSize
		statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
	default:
		return
	}

}

func (sc *StorageSmartContract) newRead(statectx c_state.StateContextI, numReads int64) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	if err != nil {
		return
	}

	stats.Stats.NumReads += numReads
	statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
}

func (sc *StorageSmartContract) newChallenge(statectx c_state.StateContextI, challengeTimestamp common.Timestamp) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	if err != nil {
		return
	}

	stats.Stats.OpenChallenges++
	stats.Stats.TotalChallenges++
	stats.LastChallengedSize = stats.Stats.UsedSize
	stats.LastChallengedTime = challengeTimestamp
	statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
}

func (sc *StorageSmartContract) challengeResolved(statectx c_state.StateContextI, challengedPassed bool) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	if err != nil {
		return
	}

	stats.Stats.OpenChallenges--
	if challengedPassed {
		stats.Stats.SuccessChallenges++
	} else {
		stats.Stats.FailedChallenges++
	}
	statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
}
