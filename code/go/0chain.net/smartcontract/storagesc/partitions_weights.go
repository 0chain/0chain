// This file implements the partition weights for challenge-ready blobber partitions with metadata weights.
//
// There are two main parts involved: the partitions of blobber weights and a node that records the weights of each partition.
// When selecting a blobber to challenge, a random value in the range [0, total weight of all partitions] is generated.
// The partition is then selected based on the random value, and a blobber is picked from that partition.
// The implementation details of this selection process can be found in the `pick()` method below.
//
// In addition to the `pick()` method, the core function of this partition weights is to keep the blobber weights and partition weights in sync,
// especially when a blobber weight is added or removed.
package storagesc

import (
	"errors"
	"fmt"
	"math/rand"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

//go:generate msgp -io=false -tests=false -v

var (
	blobberPartWeightPartitionsKey = encryption.Hash("blobber_part_weight_partitions")
)

// PartitionWeight represents weight of a partition
type PartitionWeight struct {
	Weight int `msg:"w"`
}

// PartitionsWeights stores all the partitions weight
type PartitionsWeights struct {
	Parts []PartitionWeight `msg:"ps"`
}

func (pws *PartitionsWeights) save(state state.StateContextI) error {
	_, err := state.InsertTrieNode(blobberPartWeightPartitionsKey, pws)
	return err
}

func (pws *PartitionsWeights) totalWeight() int {
	total := 0
	for _, w := range pws.Parts {
		logging.Logger.Info("Jayash total partition weight ", zap.Any("weight", w.Weight), zap.Any("total", total))
		total += w.Weight
	}
	return total
}

// pick picks a blobber based on the random value and weights
func (pws *PartitionsWeights) pick(state state.StateContextI, rd *rand.Rand, bwp *blobberWeightPartitionsWrap) (string, error) {
	totalWeight := pws.totalWeight()
	logging.Logger.Info("Jayash picking a blobber", zap.Any("weight", totalWeight), zap.Any("parts", pws.Parts))
	if totalWeight <= 0 {
		logging.Logger.Error("Jayash bad weight", zap.Any("weight", totalWeight), zap.Any("parts", pws.Parts))
		return "", errors.New("bad weight")
	}
	r := rd.Intn(totalWeight)
	var blobberID string
	for pidx, pw := range pws.Parts {
		br := r // remaining weight before minus the whole partition weight
		r -= pw.Weight
		if r <= 0 {
			// iterate through the partition to find the blobber
			if err := bwp.iterBlobberWeight(state, pidx,
				func(id string, bw *ChallengeReadyBlobber) (stop bool) {
					logging.Logger.Info("Jayash picking a blobber 2", zap.String("blobber_id", id), zap.Any("weight", bw.GetWeight()), zap.Any("bw", bw), zap.Any("br", br))
					br -= int(bw.GetWeight())
					if br <= 0 {
						blobberID = bw.BlobberID
						// find the blobber, break and return
						stop = true
					}
					return
				}); err != nil {
				return "", err
			}

			if blobberID == "" {
				return "", fmt.Errorf("could not pic a blobber, blobber weights may not synced")
			}

			return blobberID, nil
		}
	}
	return "", fmt.Errorf("could not pick a blobber, blobber weights may not synced")
}

// blobberWeightPartitionsWrap is a wrapper for blobber weights partitions.Partitions and
// partitions weights node
type blobberWeightPartitionsWrap struct {
	p           *partitions.Partitions // challenge ready blobbers partitions
	partWeights *PartitionsWeights     // partitions weights
	needSync    bool                   // indicates if the partitions weights need to be synced
}

func blobberWeightsPartitions(state state.StateContextI, p *partitions.Partitions) (*blobberWeightPartitionsWrap, error) {
	// load the partition weight if exist
	var partWeights PartitionsWeights
	var needSync bool
	if err := state.GetTrieNode(blobberPartWeightPartitionsKey, &partWeights); err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		// mark it as needs sync if see 'value not present' error, means the first time
		needSync = true
	}

	return &blobberWeightPartitionsWrap{p: p, partWeights: &partWeights, needSync: needSync}, nil
}

type forEachFunc func(id string, bw *ChallengeReadyBlobber) bool

func (bp *blobberWeightPartitionsWrap) pick(state state.StateContextI, rd *rand.Rand) (string, error) {
	return bp.partWeights.pick(state, rd, bp)
}

// sync syncs the blobber weights from the challenge ready partitions to the partitions weight
func (bp *blobberWeightPartitionsWrap) sync(state state.StateContextI, crp *partitions.Partitions) error {
	bp.partWeights.Parts = make([]PartitionWeight, crp.Last.Loc+1)
	if err := crp.ForEach(state, func(partIndex int, _ string, v []byte) (stop bool) {
		crb := ChallengeReadyBlobber{}
		_, err := crb.UnmarshalMsg(v)
		if err != nil {
			logging.Logger.Error("unmarshal challenge ready blobber failed", zap.Error(err))
			stop = true
			return
		}

		bp.partWeights.Parts[partIndex].Weight += int(crb.GetWeight())
		return
	}); err != nil {
		return err
	}
	bp.p = crp

	return bp.partWeights.save(state)
}

func (bp *blobberWeightPartitionsWrap) add(state state.StateContextI, bw ChallengeReadyBlobber) error {
	loc, err := bp.p.AddX(state, &bw)
	if err != nil {
		return err
	}

	// update the partition weight
	if loc >= len(bp.partWeights.Parts) {
		bp.partWeights.Parts = append(bp.partWeights.Parts, PartitionWeight{Weight: int(bw.GetWeight())})
		return bp.save(state)
	}

	bp.partWeights.Parts[loc].Weight += int(bw.GetWeight())
	return bp.save(state)
}

// remove removes the blobber weight from the partitions and update the partition weight.
// remove  is a bit complex as partitions will replace the removed one with the last part's tail item, so
// the partition weight should be updated accordingly. Also,
// if the last partition is empty, the partion weight should be removed
func (bp *blobberWeightPartitionsWrap) remove(state state.StateContextI, blobberID string) error {
	// get the blobber weight to be removed
	bw := ChallengeReadyBlobber{}
	_, err := bp.p.Get(state, blobberID, &bw)
	if err != nil {
		return err
	}

	removeLocs, err := bp.p.RemoveX(state, blobberID)
	if err != nil {
		return err
	}

	if len(bp.partWeights.Parts)-1 != removeLocs.Replace {
		return fmt.Errorf("replace item must be from the last partition")
	}

	// update the partition weight
	//
	// if removed item and replace item are in the same partition, just reduce the weight
	if removeLocs.From == removeLocs.Replace {
		bp.partWeights.Parts[removeLocs.From].Weight -= int(bw.GetWeight())
		// remove if partition weight is 0,
		if bp.partWeights.Parts[removeLocs.From].Weight == 0 {
			// remove the last part weight, as 0 weight could only happen when it's last part
			bp.partWeights.Parts = bp.partWeights.Parts[:len(bp.partWeights.Parts)-1]
		}

		return bp.save(state)
	}

	// for removed item and replace item in different part
	//
	// 1. reduce the weight of the replace item's partition, i.e the last one
	// 2. apply the difference to the removed item's partition
	repBw := ChallengeReadyBlobber{}
	_, err = repBw.UnmarshalMsg(removeLocs.ReplaceItem)
	if err != nil {
		return err
	}

	// reduce the weight of the replace item's partition
	bp.partWeights.Parts[removeLocs.Replace].Weight -= int(repBw.GetWeight())
	// apply the difference to the removed item's partition
	diff := int(repBw.GetWeight()) - int(bw.GetWeight())
	bp.partWeights.Parts[removeLocs.From].Weight += diff
	return bp.save(state)
}

func (bp *blobberWeightPartitionsWrap) update(state state.StateContextI, bw ChallengeReadyBlobber) error {
	var diff int
	partIndex, err := bp.p.Update(state, bw.BlobberID, func(v []byte) ([]byte, error) {
		savedBw := ChallengeReadyBlobber{}
		_, err := savedBw.UnmarshalMsg(v)
		if err != nil {
			return nil, err
		}

		diff = int(bw.GetWeight()) - int(savedBw.GetWeight())
		return bw.MarshalMsg(nil)
	})
	if err != nil {
		return err
	}

	bp.partWeights.Parts[partIndex].Weight += diff
	return bp.save(state)
}

func (bp *blobberWeightPartitionsWrap) iterBlobberWeight(state state.StateContextI, partIndex int, cf forEachFunc) error {
	var err error
	if ferr := bp.p.ForEachPart(state, partIndex, func(_ int, id string, v []byte) (stop bool) {
		bw := ChallengeReadyBlobber{}
		_, err = bw.UnmarshalMsg(v)
		if err != nil {
			err = fmt.Errorf("unmarshal blobber weight: %v", err)
			stop = true
			return
		}

		return cf(id, &bw)
	}); ferr != nil {
		return ferr
	}

	return err
}

// save saves both the partitions and the partitions weights node to MPT
func (bp *blobberWeightPartitionsWrap) save(state state.StateContextI) error {
	if err := bp.partWeights.save(state); err != nil {
		return err
	}

	return bp.p.Save(state)
}
