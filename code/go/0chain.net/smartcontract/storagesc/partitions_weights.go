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
	"fmt"
	"math/rand"
	"sort"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
	"github.com/0chain/common/core/util"
)

//go:generate msgp -io=false -tests=false -v

var (
	blobberWeightPartitionKey      = encryption.Hash("blobber_weight_partitions")
	blobberWeightPartitionSize     = 50
	blobberPartWeightPartitionsKey = encryption.Hash("blobber_part_weight_partitions")
)

type BlobberWeight struct {
	BlobberID string `msg:"bid"`
	Weight    int    `msg:"w"`
}

func (bw *BlobberWeight) GetID() string {
	return bw.BlobberID
}

// PartitionWeight represents weight of a partition
type PartitionWeight struct {
	Weight int `msg:"w"`
}

// PartitionsWeights stores all the partitions weight
type PartitionsWeights struct {
	Parts []PartitionWeight `msg:"ps"`
}

func (pws *PartitionsWeights) set(pwv []PartitionWeight) {
	pws.Parts = make([]PartitionWeight, len(pwv))
	copy(pws.Parts, pwv)
}

func (pws *PartitionsWeights) save(state state.StateContextI) error {
	_, err := state.InsertTrieNode(blobberPartWeightPartitionsKey, pws)
	return err
}

func (pws *PartitionsWeights) totalWeight() int {
	total := 0
	for _, w := range pws.Parts {
		total += w.Weight
	}
	return total
}

// pick picks a blobber based on the random value and weights
func (pws *PartitionsWeights) pick(state state.StateContextI, rd *rand.Rand, bwp *blobberWeightPartitionsWrap) (string, error) {
	r := rd.Intn(pws.totalWeight())
	var blobberID string
	for pidx, pw := range pws.Parts {
		br := r // remaining weight before minus the whole partition weight
		r -= pw.Weight
		if r <= 0 {
			// iterate through the partition to find the blobber
			if err := bwp.iterBlobberWeight(state, pidx,
				func(id string, bw *BlobberWeight) (stop bool) {
					br -= bw.Weight
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
	p           *partitions.Partitions
	partWeights *PartitionsWeights
}

func blobberWeightsPartitions(state state.StateContextI) (*blobberWeightPartitionsWrap, error) {
	p, err := partitions.CreateIfNotExists(state, blobberWeightPartitionKey, blobberWeightPartitionSize)
	if err != nil {
		return nil, err
	}

	// load the partition weight if exist
	var partWeights PartitionsWeights
	if err := state.GetTrieNode(blobberPartWeightPartitionsKey, &partWeights); err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
	}

	return &blobberWeightPartitionsWrap{p: p, partWeights: &partWeights}, nil
}

type forEachFunc func(id string, bw *BlobberWeight) bool
type iterPartFunc func(partIndex int, cf forEachFunc) error

func (bp *blobberWeightPartitionsWrap) pick(state state.StateContextI, rd *rand.Rand) (string, error) {
	return bp.partWeights.pick(state, rd, bp)
}

func (bp *blobberWeightPartitionsWrap) init(state state.StateContextI, weights []BlobberWeight) error {
	// TODO: init with hard fork activator
	partWeightMap := make(map[int]int)
	for _, w := range weights {
		loc, err := bp.p.AddX(state, &w)
		if err != nil {
			return err
		}
		partWeightMap[loc] += w.Weight
	}

	partIndexs := make([]int, 0, len(partWeightMap))
	for pi := range partWeightMap {
		partIndexs = append(partIndexs, pi)
	}

	sort.Ints(partIndexs)
	// add to partition weight
	partWeights := make([]PartitionWeight, 0, len(partWeightMap))
	for _, partIndex := range partIndexs {
		w := partWeightMap[partIndex]
		// partWeights = append(partWeights, PartitionWeight{Index: partIndex, Weight: w})
		partWeights = append(partWeights, PartitionWeight{Weight: w})
	}

	bp.partWeights.set(partWeights)

	return bp.save(state)
}

func (bp *blobberWeightPartitionsWrap) add(state state.StateContextI, bw BlobberWeight) error {
	loc, err := bp.p.AddX(state, &bw)
	if err != nil {
		return err
	}

	// update the partition weight
	if loc >= len(bp.partWeights.Parts) {
		bp.partWeights.Parts = append(bp.partWeights.Parts, PartitionWeight{Weight: bw.Weight})
		return bp.save(state)
	}

	bp.partWeights.Parts[loc].Weight += bw.Weight
	return bp.save(state)
}

// remove removes the blobber weight from the partitions and update the partition weight.
// remove  is a bit complex as partitions will replace the removed one with the last part's tail item, so
// the partition weight should be updated accordingly. Also,
// if the last partition is empty, the partion weight should be removed
func (bp *blobberWeightPartitionsWrap) remove(state state.StateContextI, blobberID string) error {
	// get the blobber weight to be removed
	bw := BlobberWeight{}
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
		bp.partWeights.Parts[removeLocs.From].Weight -= bw.Weight
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
	repBw := BlobberWeight{}
	_, err = repBw.UnmarshalMsg(removeLocs.ReplaceItem)
	if err != nil {
		return err
	}

	// reduce the weight of the replace item's partition
	bp.partWeights.Parts[removeLocs.Replace].Weight -= repBw.Weight
	// apply the difference to the removed item's partition
	diff := repBw.Weight - bw.Weight
	bp.partWeights.Parts[removeLocs.From].Weight += diff
	return bp.save(state)
}

func (bp *blobberWeightPartitionsWrap) updateWeight(state state.StateContextI, bw BlobberWeight) error {
	var diff int
	partIndex, err := bp.p.Update(state, bw.BlobberID, func(v []byte) ([]byte, error) {
		savedBw := BlobberWeight{}
		_, err := savedBw.UnmarshalMsg(v)
		if err != nil {
			return nil, err
		}

		diff = bw.Weight - savedBw.Weight
		savedBw.Weight = bw.Weight
		return savedBw.MarshalMsg(nil)
	})
	if err != nil {
		return err
	}

	bp.partWeights.Parts[partIndex].Weight += diff
	return bp.save(state)
}

func (bp *blobberWeightPartitionsWrap) iterBlobberWeight(state state.StateContextI, partIndex int, cf forEachFunc) error {
	var err error
	if ferr := bp.p.ForEach(state, partIndex, func(id string, v []byte) (stop bool) {
		bw := BlobberWeight{}
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
