package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
)

func (ssc *StorageSmartContract) newWrite(statectx cstate.StateContextI, writeSize int64) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	statsBytes, err := statectx.GetTrieNode(stats.GetKey(ssc.ID))
	if statsBytes != nil {
		err = stats.Decode(statsBytes.Encode())
		if err != nil {
			Logger.Error("storage stats decode error")
			return
		}
	}
	
	stats.Stats.NumWrites++
	stats.Stats.UsedSize += writeSize
	statectx.InsertTrieNode(stats.GetKey(ssc.ID), stats)

}

func (ssc *StorageSmartContract) newRead(statectx cstate.StateContextI, numReads int64) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	statsBytes, err := statectx.GetTrieNode(stats.GetKey(ssc.ID))
	if err != nil {
		return
	}
	if statsBytes != nil {
		err = stats.Decode(statsBytes.Encode())
		if err != nil {
			Logger.Error("storage stats decode error")
			return
		}
	}
	
	stats.Stats.NumReads+=numReads
	statectx.InsertTrieNode(stats.GetKey(ssc.ID), stats)
}

func (ssc *StorageSmartContract) newChallenge(statectx cstate.StateContextI, challengeTimestamp common.Timestamp) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	statsBytes, err := statectx.GetTrieNode(stats.GetKey(ssc.ID))
	if err != nil {
		return
	}
	if statsBytes != nil {
		err = stats.Decode(statsBytes.Encode())
		if err != nil {
			Logger.Error("storage stats decode error")
			return
		}
	}
	
	stats.Stats.OpenChallenges++
	stats.Stats.TotalChallenges++
	stats.LastChallengedSize = stats.Stats.UsedSize
	stats.LastChallengedTime = challengeTimestamp
	statectx.InsertTrieNode(stats.GetKey(ssc.ID), stats)
}

func (ssc *StorageSmartContract) challengeResolved(statectx cstate.StateContextI, challengedPassed bool) {
	stats := &StorageStats{}
	stats.Stats = &StorageAllocationStats{}
	statsBytes, err := statectx.GetTrieNode(stats.GetKey(ssc.ID))
	if err != nil {
		return
	}
	if statsBytes != nil {
		err = stats.Decode(statsBytes.Encode())
		if err != nil {
			Logger.Error("storage stats decode error")
			return
		}
	}
	
	stats.Stats.OpenChallenges--
	if challengedPassed {
		stats.Stats.SuccessChallenges++
	} else {
		stats.Stats.FailedChallenges++
	}
	statectx.InsertTrieNode(stats.GetKey(ssc.ID), stats)
}
