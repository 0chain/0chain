package storagesc

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func init() {
	// set for testing only
	blobberWeightPartitionSize = 5
}

func TestBlobberWeightPartitionsWrapPick(t *testing.T) {
	state := newTestBalances(t, false)
	bp, err := blobberWeightsPartitions(state)
	require.NoError(t, err)

	weights := []BlobberWeight{
		{BlobberID: "blobber1", Weight: 10},
		{BlobberID: "blobber2", Weight: 20},
		{BlobberID: "blobber3", Weight: 30},
		{BlobberID: "blobber4", Weight: 15},
		{BlobberID: "blobber5", Weight: 25},
		{BlobberID: "blobber6", Weight: 35},
		{BlobberID: "blobber7", Weight: 40},
		{BlobberID: "blobber8", Weight: 50},
		{BlobberID: "blobber9", Weight: 60},
		{BlobberID: "blobber10", Weight: 70},
		{BlobberID: "blobber11", Weight: 80},
	}

	// Initialize blobberWeightPartitionsWrap
	err = bp.init(state, weights)
	if err != nil {
		t.Fatal(err)
	}

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
		fmt.Println("blobberID:", bw.BlobberID, "weight:", bw.Weight, "picked:", pickMap[bw.BlobberID])
	}

}

func TestBlobberWeightPartitionsWrapUpdateWeight(t *testing.T) {
	state := newTestBalances(t, false)
	bp, err := blobberWeightsPartitions(state)
	require.NoError(t, err)

	weights := []BlobberWeight{
		{BlobberID: "blobber1", Weight: 10},
		{BlobberID: "blobber2", Weight: 20},
		{BlobberID: "blobber3", Weight: 30},
		{BlobberID: "blobber4", Weight: 15},
		{BlobberID: "blobber5", Weight: 25},
		{BlobberID: "blobber6", Weight: 35},
		{BlobberID: "blobber7", Weight: 40},
		{BlobberID: "blobber8", Weight: 50},
		{BlobberID: "blobber9", Weight: 60},
		{BlobberID: "blobber10", Weight: 70},
		{BlobberID: "blobber11", Weight: 80},
	}

	// Initialize blobberWeightPartitionsWrap
	err = bp.init(state, weights)
	if err != nil {
		t.Fatal(err)
	}
	b1PartWeight := bp.partWeights.Parts[0]
	require.Equal(t, 100, b1PartWeight.Weight)

	err = bp.updateWeight(state, BlobberWeight{BlobberID: "blobber1", Weight: 11})
	require.NoError(t, err)

	b1w := BlobberWeight{}
	_, err = bp.p.Get(state, "blobber1", &b1w)
	require.NoError(t, err)
	require.Equal(t, 11, b1w.Weight)

	require.Equal(t, 101, bp.partWeights.Parts[0].Weight)

	// reload from state
	nbp, err := blobberWeightsPartitions(state)
	loc, err := nbp.p.Get(state, "blobber1", &b1w)
	require.NoError(t, err)
	require.Equal(t, 11, b1w.Weight)

	require.NoError(t, err)
	require.Equal(t, 101, nbp.partWeights.Parts[loc].Weight)
}

func TestBlobberWeightPartitionsWrapAdd(t *testing.T) {
	weights := []BlobberWeight{
		{BlobberID: "blobber1", Weight: 10},
		{BlobberID: "blobber2", Weight: 20},
		{BlobberID: "blobber3", Weight: 30},
		{BlobberID: "blobber4", Weight: 40},
		{BlobberID: "blobber5", Weight: 50},
		{BlobberID: "blobber6", Weight: 60},
	}

	testCases := []struct {
		name               string
		bw                 BlobberWeight
		initWeights        []BlobberWeight
		expectedPartWeight int
	}{
		{
			name:               "Add new BlobberWeight",
			initWeights:        weights[:3],
			bw:                 BlobberWeight{BlobberID: "blobber4", Weight: 40},
			expectedPartWeight: 100,
		},
		{
			name:               "Add to empty partition",
			initWeights:        []BlobberWeight{},
			bw:                 BlobberWeight{BlobberID: "blobber1", Weight: 10},
			expectedPartWeight: 10,
		},
		{
			name:               "Add to last one of a partition",
			initWeights:        weights[:4],
			bw:                 BlobberWeight{BlobberID: "blobber5", Weight: 50},
			expectedPartWeight: 150,
		},
		{
			name:               "Add to first one of a new partition",
			initWeights:        weights[:5],
			bw:                 BlobberWeight{BlobberID: "blobber6", Weight: 60},
			expectedPartWeight: 60,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := newTestBalances(t, false)
			bp, err := blobberWeightsPartitions(state)
			require.NoError(t, err)

			err = bp.init(state, tc.initWeights)
			require.NoError(t, err)

			err = bp.add(state, tc.bw)
			require.NoError(t, err)

			// Verify that the new BlobberWeight is added correctly
			bw := BlobberWeight{}
			_, err = bp.p.Get(state, tc.bw.BlobberID, &bw)
			require.NoError(t, err)
			require.Equal(t, tc.bw, bw)

			// load bp from state
			bp, err = blobberWeightsPartitions(state)
			require.NoError(t, err)

			var bw2 BlobberWeight
			loc, err := bp.p.Get(state, tc.bw.BlobberID, &bw2)
			require.NoError(t, err)
			require.Equal(t, tc.bw, bw2)

			require.Equal(t, tc.expectedPartWeight, bp.partWeights.Parts[loc].Weight)
		})
	}
}
