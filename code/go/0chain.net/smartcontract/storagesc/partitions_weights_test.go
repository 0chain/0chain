package storagesc

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
	err = bp.p.Get(state, "blobber1", &b1w)
	require.NoError(t, err)
	require.Equal(t, 11, b1w.Weight)

	require.Equal(t, 101, bp.partWeights.Parts[0].Weight)

	// reload from state
	nbp, err := blobberWeightsPartitions(state)
	err = nbp.p.Get(state, "blobber1", &b1w)
	require.NoError(t, err)
	require.Equal(t, 11, b1w.Weight)

	require.NoError(t, err)
	require.Equal(t, 101, nbp.partWeights.Parts[0].Weight)
}
