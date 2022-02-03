package storagesc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"fmt"
)

func (sc *StorageSmartContract) newWrite(statectx c_state.StateContextI, writeSize int64) {
	stats, err := GetStorageStats(statectx, sc.ID)
	if err != nil {
		return
	}

	stats.Stats.NumWrites++
	stats.Stats.UsedSize += writeSize
	statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)

}

func (sc *StorageSmartContract) newRead(statectx c_state.StateContextI, numReads int64) {
	stats, err := GetStorageStats(statectx, sc.ID)
	if err != nil {
		return
	}

	stats.Stats.NumReads += numReads
	statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
}

func (sc *StorageSmartContract) newChallenge(statectx c_state.StateContextI, challengeTimestamp common.Timestamp) {
	stats, err := GetStorageStats(statectx, sc.ID)
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
	stats, err := GetStorageStats(statectx, sc.ID)
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

func GetStorageStats(statectx c_state.StateContextI, id string) (*StorageStats, error) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	raw, err := statectx.GetTrieNode(stats.GetKey(id), stats)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		if val, ok := raw.(*StorageStats); !ok {
			return stats, fmt.Errorf("unexpected node type")
		} else {
			stats = val
		}
	}
	return stats, nil
}
