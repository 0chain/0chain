package storagesc

import (
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"fmt"
)

func (sc *StorageSmartContract) newWrite(statectx c_state.StateContextI, writeSize int64) error {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	raw, err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	switch err {
	case nil, util.ErrValueNotPresent:
		var ok bool
		if stats, ok = raw.(*StorageStats); !ok {
			return fmt.Errorf("unexpected node type")
		}
		stats.Stats.NumWrites++
		stats.Stats.UsedSize += writeSize
		err = statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
		return err
	default:
		return err
	}
}

func (sc *StorageSmartContract) newRead(statectx c_state.StateContextI, readSize int64) error {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	raw, err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	if err != nil && err != util.ErrValueNotPresent {
		return err
	}
	var ok bool
	if stats, ok = raw.(*StorageStats); !ok {
		return fmt.Errorf("unexpected node type")
	}
	stats.Stats.ReadsSize += readSize
	err = statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
	return err
}

func (sc *StorageSmartContract) newChallenge(statectx c_state.StateContextI, challengeTimestamp common.Timestamp) error {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	raw, err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	if err != nil && err != util.ErrValueNotPresent {
		return err
	}

	var ok bool
	if stats, ok = raw.(*StorageStats); !ok {
		return fmt.Errorf("unexpected node type")
	}

	stats.Stats.OpenChallenges++
	stats.Stats.TotalChallenges++
	stats.LastChallengedSize = stats.Stats.UsedSize
	stats.LastChallengedTime = challengeTimestamp
	err = statectx.InsertTrieNode(stats.GetKey(sc.ID), stats)
	return err
}

func (sc *StorageSmartContract) challengeResolved(statectx c_state.StateContextI, challengedPassed bool) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	raw, err := statectx.GetTrieNode(stats.GetKey(sc.ID), stats)
	if err != nil {
		return
	}
	var ok bool
	if stats, ok = raw.(*StorageStats); !ok {
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
