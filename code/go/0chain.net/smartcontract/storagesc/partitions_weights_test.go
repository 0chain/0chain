package storagesc

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"0chain.net/chaincore/chain/state"
	partitions_v2 "0chain.net/smartcontract/partitions_v_2"
	"github.com/stretchr/testify/require"
)

var challengeReadyPartSize = 5

func testPreparePartWeights(t *testing.T, state state.StateContextI) *blobberWeightPartitionsWrap {
	p, err := partitions_v2.CreateIfNotExists(state, "test_challenge_ready_partitions", challengeReadyPartSize)
	require.NoError(t, err)
	bp, err := blobberWeightsPartitions(state, p)
	require.NoError(t, err)
	return bp
}

func initBlobberWeightParts(state state.StateContextI, bp *blobberWeightPartitionsWrap, weights []ChallengeReadyBlobber) error {
	partWeightMap := make(map[int]int)
	for _, w := range weights {
		loc, err := bp.p.AddX(state, &w)
		if err != nil {
			return err
		}
		partWeightMap[loc] += int(w.GetWeight())
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

	// func (pws *PartitionsWeights) set(pwv []PartitionWeight) {
	bp.partWeights.Parts = make([]PartitionWeight, len(partWeights))
	copy(bp.partWeights.Parts, partWeights)

	// bp.partWeights.set(partWeights)

	return bp.save(state)
}

func TestBlobberWeightPartitionsWrapPick(t *testing.T) {
	weights := []ChallengeReadyBlobber{
		{BlobberID: "blobber1", Stake: 1e10, UsedCapacity: 10},
		{BlobberID: "blobber2", Stake: 1e10, UsedCapacity: 20},
		{BlobberID: "blobber3", Stake: 1e10, UsedCapacity: 30},
		{BlobberID: "blobber4", Stake: 1e10, UsedCapacity: 15},
		{BlobberID: "blobber5", Stake: 1e10, UsedCapacity: 25},
		{BlobberID: "blobber6", Stake: 1e10, UsedCapacity: 35},
		{BlobberID: "blobber7", Stake: 1e10, UsedCapacity: 40},
		{BlobberID: "blobber8", Stake: 1e10, UsedCapacity: 50},
		{BlobberID: "blobber9", Stake: 1e10, UsedCapacity: 60},
		{BlobberID: "blobber10", Stake: 1e10, UsedCapacity: 70},
		{BlobberID: "blobber11", Stake: 1e10, UsedCapacity: 80},
	}

	state := newTestBalances(t, false)
	bp := testPreparePartWeights(t, state)
	err := initBlobberWeightParts(state, bp, weights)
	require.NoError(t, err)

	// Pick a blobber
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	pickMap := make(map[string]int)
	for i := 0; i < 500; i++ {
		blobberID, err := bp.pick(state, rd)
		if err != nil {
			t.Fatal(err)
		}
		pickMap[blobberID]++

		// Check if blobberID is in weights
		found := false
		for _, w := range weights {
			if w.BlobberID == blobberID {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected blobberID to be in weights, got %s", blobberID)
		}
	}
	for _, bw := range weights {
		fmt.Println("blobberID:", bw.BlobberID, "weight:", bw.GetWeight(), "picked:", pickMap[bw.BlobberID])
	}

}

func TestBlobberWeightPartitionsWrapUpdateWeight(t *testing.T) {
	state := newTestBalances(t, false)
	bp := testPreparePartWeights(t, state)

	weights := []ChallengeReadyBlobber{
		{BlobberID: "blobber1", Stake: 1e10, UsedCapacity: 10},
		{BlobberID: "blobber2", Stake: 1e10, UsedCapacity: 20},
		{BlobberID: "blobber3", Stake: 1e10, UsedCapacity: 30},
		{BlobberID: "blobber4", Stake: 1e10, UsedCapacity: 15},
		{BlobberID: "blobber5", Stake: 1e10, UsedCapacity: 25},
		{BlobberID: "blobber6", Stake: 1e10, UsedCapacity: 35},
		{BlobberID: "blobber7", Stake: 1e10, UsedCapacity: 40},
		{BlobberID: "blobber8", Stake: 1e10, UsedCapacity: 50},
		{BlobberID: "blobber9", Stake: 1e10, UsedCapacity: 60},
		{BlobberID: "blobber10", Stake: 1e10, UsedCapacity: 70},
		{BlobberID: "blobber11", Stake: 1e10, UsedCapacity: 80},
	}

	// Initialize blobberWeightPartitionsWrap
	err := initBlobberWeightParts(state, bp, weights)
	if err != nil {
		t.Fatal(err)
	}
	b1PartWeight := bp.partWeights.Parts[0]
	require.Equal(t, 100, b1PartWeight.Weight)

	err = bp.update(state, ChallengeReadyBlobber{BlobberID: "blobber1", Stake: 1e10, UsedCapacity: 11})
	require.NoError(t, err)

	b1w := ChallengeReadyBlobber{}
	_, err = bp.p.Get(state, "blobber1", &b1w)
	require.NoError(t, err)
	require.Equal(t, 11, int(b1w.UsedCapacity))

	require.Equal(t, 101, bp.partWeights.Parts[0].Weight)

	// reload from state
	nbp := testPreparePartWeights(t, state)
	loc, err := nbp.p.Get(state, "blobber1", &b1w)
	require.NoError(t, err)
	require.Equal(t, 11, int(b1w.GetWeight()))

	require.NoError(t, err)
	require.Equal(t, 101, nbp.partWeights.Parts[loc].Weight)
}

func TestBlobberWeightPartitionsWrapAdd(t *testing.T) {
	weights := []ChallengeReadyBlobber{
		{BlobberID: "blobber1", Stake: 1e10, UsedCapacity: 10},
		{BlobberID: "blobber2", Stake: 1e10, UsedCapacity: 20},
		{BlobberID: "blobber3", Stake: 1e10, UsedCapacity: 30},
		{BlobberID: "blobber4", Stake: 1e10, UsedCapacity: 40},
		{BlobberID: "blobber5", Stake: 1e10, UsedCapacity: 50},
		{BlobberID: "blobber6", Stake: 1e10, UsedCapacity: 60},
	}

	testCases := []struct {
		name               string
		bw                 ChallengeReadyBlobber
		initWeights        []ChallengeReadyBlobber
		expectedPartWeight int
	}{
		{
			name:               "Add new BlobberWeight",
			initWeights:        weights[:3],
			bw:                 ChallengeReadyBlobber{BlobberID: "blobber4", Stake: 1e10, UsedCapacity: 40},
			expectedPartWeight: 100,
		},
		{
			name:               "Add to empty partition",
			initWeights:        []ChallengeReadyBlobber{},
			bw:                 ChallengeReadyBlobber{BlobberID: "blobber1", Stake: 1e10, UsedCapacity: 10},
			expectedPartWeight: 10,
		},
		{
			name:               "Add to last one of a partition",
			initWeights:        weights[:4],
			bw:                 ChallengeReadyBlobber{BlobberID: "blobber5", Stake: 1e10, UsedCapacity: 50},
			expectedPartWeight: 150,
		},
		{
			name:               "Add to first one of a new partition",
			initWeights:        weights[:5],
			bw:                 ChallengeReadyBlobber{BlobberID: "blobber6", Stake: 1e10, UsedCapacity: 60},
			expectedPartWeight: 60,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := newTestBalances(t, false)
			bp := testPreparePartWeights(t, state)
			err := initBlobberWeightParts(state, bp, tc.initWeights)
			require.NoError(t, err)

			err = bp.add(state, tc.bw)
			require.NoError(t, err)

			// Verify that the new BlobberWeight is added correctly
			bw := ChallengeReadyBlobber{}
			_, err = bp.p.Get(state, tc.bw.BlobberID, &bw)
			require.NoError(t, err)
			require.Equal(t, tc.bw, bw)

			// load bp from state
			bp = testPreparePartWeights(t, state)

			var bw2 ChallengeReadyBlobber
			loc, err := bp.p.Get(state, tc.bw.BlobberID, &bw2)
			require.NoError(t, err)
			require.Equal(t, tc.bw, bw2)

			require.Equal(t, tc.expectedPartWeight, bp.partWeights.Parts[loc].Weight)
		})
	}
}
func TestBlobberWeightPartitionsWrapRemove(t *testing.T) {
	state := newTestBalances(t, false)
	bp := testPreparePartWeights(t, state)
	weights := []ChallengeReadyBlobber{
		{BlobberID: "blobber1", Stake: 1e10, UsedCapacity: 10},
		{BlobberID: "blobber2", Stake: 1e10, UsedCapacity: 20},
		{BlobberID: "blobber3", Stake: 1e10, UsedCapacity: 30},
		{BlobberID: "blobber4", Stake: 1e10, UsedCapacity: 15},
		{BlobberID: "blobber5", Stake: 1e10, UsedCapacity: 25},
		{BlobberID: "blobber6", Stake: 1e10, UsedCapacity: 35},
		{BlobberID: "blobber7", Stake: 1e10, UsedCapacity: 40},
		{BlobberID: "blobber8", Stake: 1e10, UsedCapacity: 50},
		{BlobberID: "blobber9", Stake: 1e10, UsedCapacity: 60},
		{BlobberID: "blobber10", Stake: 1e10, UsedCapacity: 70},
		{BlobberID: "blobber11", Stake: 1e10, UsedCapacity: 80},
	}

	// Initialize blobberWeightPartitionsWrap
	err := initBlobberWeightParts(state, bp, weights)
	require.NoError(t, err)

	// Test removing a blobber from the last partition
	err = bp.remove(state, "blobber11")
	require.NoError(t, err)
	require.Equal(t, 2, len(bp.partWeights.Parts))

	bp.save(state)

	// Test removing a blobber from the same partition as the replace item
	bp = testPreparePartWeights(t, state)
	require.NoError(t, err)

	err = bp.remove(state, "blobber10")
	require.NoError(t, err)
	require.Equal(t, 185, bp.partWeights.Parts[1].Weight)
	bp.save(state)

	// Test removing a blobber from a different partition than the replace item
	bp = testPreparePartWeights(t, state)
	require.NoError(t, err)
	err = bp.remove(state, "blobber2")
	require.NoError(t, err)
	// 100 - 20 + 60 = 140
	require.Equal(t, 140, bp.partWeights.Parts[0].Weight)
	// 185 - 60 = 125
	require.Equal(t, 125, bp.partWeights.Parts[1].Weight)
	bp.save(state)

	bp = testPreparePartWeights(t, state)
	require.NoError(t, err)
	// Test removing a blobber that doesn't exist
	err = bp.remove(state, "blobber12")
	require.Error(t, err)
	require.Equal(t, "item not found: blobber12", err.Error())
}

func TestBlobberWeightPartitionsWrapMigrate(t *testing.T) {
	state := newTestBalances(t, false)
	bp := testPreparePartWeights(t, state)
	// p, err := partitions.CreateIfNotExists(state, "test_crp", challengeReadyPartSize)

	for i := 1; i <= challengeReadyPartSize+1; i++ {
		err := bp.p.Add(state, &ChallengeReadyBlobber{
			BlobberID:    fmt.Sprintf("blobber%d", i),
			Stake:        1e10,
			UsedCapacity: uint64(i * 10)})
		require.NoError(t, err)
	}

	err := bp.sync(state, bp.p)
	require.NoError(t, err)

	// Verify that the blobber weights are correctly migrated
	for i := 1; i <= challengeReadyPartSize+1; i++ {
		bw := ChallengeReadyBlobber{}
		_, err = bp.p.Get(state, fmt.Sprintf("blobber%d", i), &bw)
		require.NoError(t, err)
		require.Equal(t, i*10, int(bw.GetWeight()))
	}

	// Verify that the partition weights are correctly migrated
	expectPartNum := (challengeReadyPartSize + 1) / challengeReadyPartSize
	if (challengeReadyPartSize+1)%challengeReadyPartSize != 0 {
		expectPartNum++
	}
	require.Equal(t, expectPartNum, len(bp.partWeights.Parts))
}
