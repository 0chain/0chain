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
	blobberWeightPartitionSize     = 5 // debug, change to 50 to align with the blobber challenge ready partitions later
	blobberPartWeightPartitionsKey = encryption.Hash("blobber_part_weight_partitions")
)

type BlobberWeight struct {
	BlobberID string `msg:"bid"`
	Weight    int    `msg:"w"`
}

func (bw *BlobberWeight) GetID() string {
	return bw.BlobberID
}

// PartitionWeightBlobber represents weight of a partition
type PartitionWeightBlobber struct {
	Index  int `msg:"i"` // partition index
	Weight int `msg:"w"`
}

type PartitionWeightsBlobber struct {
	Parts []PartitionWeightBlobber `msg:"ps"`
}

// blobberWeightPartitions is a wrapper for blobber weights partitions.Partitions
type blobberWeightPartitions struct {
	p           *partitions.Partitions
	partWeights *PartitionWeightsBlobber
}

func (bp *blobberWeightPartitions) iterBlobberWeight(state state.StateContextI, partIndex int, cf forEachFunc) error {
	var err error
	if ferr := bp.p.ForEach(state, partIndex, func(id string, v []byte) (stop bool) {
		bw := BlobberWeight{}
		_, err = bw.UnmarshalMsg(v)
		if err != nil {
			err = fmt.Errorf("unmarshal blobber weight: %v", err)
			stop = true
			return
		}

		stop, err = cf(id, &bw)
		if err != nil {
			stop = true
			return
		}

		return stop
	}); ferr != nil {
		return ferr
	}

	return err
}

func (bp *blobberWeightPartitions) init(state state.StateContextI, weights []BlobberWeight) error {
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
	for _, partIndex := range partIndexs {
		w := partWeightMap[partIndex]
		if err := bp.partWeights.add(state, w); err != nil {
			return err
		}
	}

	return bp.save(state)
}

func BlobberWeightsPartitions(state state.StateContextI) (*blobberWeightPartitions, error) {
	p, err := partitions.CreateIfNotExists(state, blobberWeightPartitionKey, blobberWeightPartitionSize)
	if err != nil {
		return nil, err
	}

	// load the partition weight if exist
	var partWeights PartitionWeightsBlobber
	if err := state.GetTrieNode(blobberPartWeightPartitionsKey, &partWeights); err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}

		// TODO: insert if not exist
	}

	return &blobberWeightPartitions{p: p, partWeights: &partWeights}, nil
}

// BlobberPartitionsWeights stores all the partitions weight
// this will be store in a MPT node instead of partitions
type BlobberPartitionsWeights struct {
	Parts []PartitionWeightBlobber `msg:"ps"`
}

type forEachFunc func(id string, bw *BlobberWeight) (bool, error)
type iterPartFunc func(partIndex int, cf forEachFunc) error

func (bpw *BlobberPartitionsWeights) TotalWeight() int {
	total := 0
	for _, w := range bpw.Parts {
		total += w.Weight
	}
	return total
}

// Pick picks a blobber based on the random value and weight
func (bpw *BlobberPartitionsWeights) Pick(state state.StateContextI, rd *rand.Rand, bwp *blobberWeightPartitions) (string, error) {
	r := rd.Intn(bpw.TotalWeight())
	var blobberID string
	for _, p := range bpw.Parts {
		br := r // remaining weight before minus the whole partition weight
		r -= p.Weight
		if r <= 0 {
			// iterate through the partition to find the blobber
			if err := bwp.iterBlobberWeight(state, p.Index,
				func(id string, bw *BlobberWeight) (stop bool, err error) {
					br -= bw.Weight
					if br <= 0 {
						blobberID = bw.BlobberID
						// find the blobber, break and return
						return true, nil
					}
					return false, nil
				}); err != nil {
				return "", err
			}

			if blobberID == "" {
				return "", fmt.Errorf("could not pick a blobber, should not happen")
			}

			return blobberID, nil
		}
	}
	return "", fmt.Errorf("could not pick a blobber, should not happen")
}
