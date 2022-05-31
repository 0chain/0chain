package storagesc

import (
	"testing"

	"0chain.net/chaincore/currency"

	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//
// test extension
//

func (aps allocationPools) gimmeAll() (total currency.Coin) {
	for _, ap := range aps {
		total += ap.Balance
	}
	return
}

func (aps allocationPools) allocTotal(allocID string, now int64) (
	total currency.Coin) {

	for _, ap := range aps {
		if ap.ExpireAt < common.Timestamp(now) {
			continue
		}
		if ap.AllocationID == allocID {
			total += ap.Balance
		}
	}
	return
}

func (aps allocationPools) allocBlobberTotal(allocID, blobberID string,
	now int64) (total currency.Coin) {

	for _, ap := range aps {
		if ap.ExpireAt < common.Timestamp(now) {
			continue
		}
		if ap.AllocationID != allocID {
			continue
		}
		for _, bp := range ap.Blobbers {
			if bp.BlobberID == blobberID {
				total += bp.Balance
			}
		}
	}
	return
}

//
// blobber pools
//

func Test_blobberPools(t *testing.T) {
	// getIndex
	// get
	// removeByIndex
	// remove
	// add

	var (
		bps   blobberPools
		bp    *blobberPool
		i, ok = bps.getIndex("blobber_id")
	)
	require.Zero(t, i)
	require.False(t, ok)

	bp, ok = bps.get("blobber_id")
	require.Nil(t, bp)
	require.False(t, ok)

	require.False(t, bps.remove("blobber_id"))

	var (
		b1, b2, b3, b4, b5 = "b1", "b2", "b3", "b4", "b5"
		random             = []string{b4, b1, b3, b5, b2}
		ordered            = []string{b1, b2, b3, b4, b5}
	)

	for _, b := range random {
		require.True(t, bps.add(&blobberPool{BlobberID: b}))
	}
	require.Len(t, bps, len(random))

	for i, o := range ordered {
		require.Equal(t, bps[i].BlobberID, o)
	}

	// uniqueness

	for _, b := range random {
		require.False(t, bps.add(&blobberPool{BlobberID: b}))
	}
	require.Len(t, bps, len(random))

	for i, o := range ordered {
		require.Equal(t, bps[i].BlobberID, o)
	}

	i, ok = bps.getIndex(b3)
	require.True(t, ok)
	bp, ok = bps.get(b3)
	require.True(t, ok)
	require.Equal(t, bps[i], bp)

	bps.removeByIndex(i)
	bps.remove(b4)
	for i, o := range []string{b1, b2, b5} {
		require.Equal(t, bps[i].BlobberID, o)
	}
}

//
// allocation pools
//

func Test_allocationPools(t *testing.T) {
	// allocationCut
	// blobberCut
	// removeEmpty
	// stat

	var (
		aps   allocationPools
		ap    *allocationPool
		i, ok = aps.getIndex("alloc_id")
	)
	require.Zero(t, i)
	require.False(t, ok)

	ap, ok = aps.get("alloc_id")
	require.Nil(t, ap)
	require.False(t, ok)

	var (
		a1, a2, a3, a4, a5 = "a1", "a2", "a3", "a4", "a5"
		random             = []string{a4, a1, a3, a5, a2}
		ordered            = []string{a1, a2, a3, a4, a5}
	)

	for _, a := range random {
		aps.add(&allocationPool{AllocationID: a})
	}
	require.Len(t, aps, len(random))

	for i, o := range ordered {
		require.Equal(t, aps[i].AllocationID, o)
	}

	// uniqueness

	for _, a := range random {
		aps.add(&allocationPool{AllocationID: a})
	}
	require.Len(t, aps, len(random)*2)

	for i, o := range []string{a1, a1, a2, a2, a3, a3, a4, a4, a5, a5} {
		require.Equal(t, aps[i].AllocationID, o)
	}

	i, ok = aps.getIndex(a3)
	require.True(t, ok)
	ap, ok = aps.get(a3)
	require.True(t, ok)
	require.Equal(t, aps[i], ap)

	// special methods

	//
	// allocation cut
	//

	var cut = aps.allocationCut(a1)
	require.EqualValues(t, []*allocationPool{
		&allocationPool{AllocationID: a1},
		&allocationPool{AllocationID: a1},
	}, cut)

	cut = aps.allocationCut(a2)
	require.EqualValues(t, []*allocationPool{
		&allocationPool{AllocationID: a2},
		&allocationPool{AllocationID: a2},
	}, cut)

	cut = aps.allocationCut(a3)
	require.EqualValues(t, []*allocationPool{
		&allocationPool{AllocationID: a3},
		&allocationPool{AllocationID: a3},
	}, cut)

	cut = aps.allocationCut(a4)
	require.EqualValues(t, []*allocationPool{
		&allocationPool{AllocationID: a4},
		&allocationPool{AllocationID: a4},
	}, cut)

	cut = aps.allocationCut(a5)
	require.EqualValues(t, []*allocationPool{
		&allocationPool{AllocationID: a5},
		&allocationPool{AllocationID: a5},
	}, cut)

	aps = allocationPools{
		&allocationPool{
			ExpireAt:     10,
			AllocationID: "a1",
			Blobbers:     blobberPools{},
		},
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 101},
				&blobberPool{BlobberID: "b2", Balance: 152},
			},
		},
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 0},
			},
		},
		&allocationPool{
			ExpireAt:     10,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 103},
				&blobberPool{BlobberID: "b2", Balance: 154},
			},
		},
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a3",
			Blobbers:     blobberPools{},
		},
	}

	assert.EqualValues(t, []*allocationPool{
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 101},
				&blobberPool{BlobberID: "b2", Balance: 152},
			},
		},
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 0},
			},
		},
		&allocationPool{
			ExpireAt:     10,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 103},
				&blobberPool{BlobberID: "b2", Balance: 154},
			},
		},
	}, aps.allocationCut(a2))

	assert.Len(t, aps.allocationCut("a10"), 0)

	//
	// remove expired
	//

	cut = aps.allocationCut(a2)
	cut = removeBlobberExpired(cut, "b1", 0)
	assert.EqualValues(t, []*allocationPool{
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 101},
				&blobberPool{BlobberID: "b2", Balance: 152},
			},
		},
		&allocationPool{
			ExpireAt:     10,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 103},
				&blobberPool{BlobberID: "b2", Balance: 154},
			},
		},
	}, cut)

	sortExpireAt(cut)
	assert.EqualValues(t, []*allocationPool{
		&allocationPool{
			ExpireAt:     10,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 103},
				&blobberPool{BlobberID: "b2", Balance: 154},
			},
		},
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 101},
				&blobberPool{BlobberID: "b2", Balance: 152},
			},
		},
	}, cut)

	cut = removeBlobberExpired(cut, "b2", 15)
	assert.EqualValues(t, []*allocationPool{
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 101},
				&blobberPool{BlobberID: "b2", Balance: 152},
			},
		},
	}, cut)

	require.EqualValues(t, []*allocationPool{
		&allocationPool{
			ExpireAt:     10,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 103},
				&blobberPool{BlobberID: "b2", Balance: 154},
			},
		},
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 101},
				&blobberPool{BlobberID: "b2", Balance: 152},
			},
		},
	}, aps.blobberCut(a2, "b2", 0))

	require.EqualValues(t, []*allocationPool{
		&allocationPool{
			ExpireAt:     10,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 103},
				&blobberPool{BlobberID: "b2", Balance: 154},
			},
		},
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 101},
				&blobberPool{BlobberID: "b2", Balance: 152},
			},
		},
	}, aps.blobberCut(a2, "b2", 0))

	require.EqualValues(t, []*allocationPool{
		&allocationPool{
			ExpireAt:     20,
			AllocationID: "a2",
			Blobbers: blobberPools{
				&blobberPool{BlobberID: "b1", Balance: 101},
				&blobberPool{BlobberID: "b2", Balance: 152},
			},
		},
	}, aps.blobberCut(a2, "b2", 15))

	require.EqualValues(t, []*allocationPool{}, aps.blobberCut(a2, "b2", 21))

}

func Test_allocationPools_sortExpiry(t *testing.T) {
	tests := []struct {
		name string
		aps  allocationPools
		want allocationPools
	}{
		{name: "sort by expiry",
			aps: allocationPools{
				&allocationPool{
					ExpireAt:     100,
					AllocationID: "a1",
					Blobbers:     blobberPools{},
				},
				&allocationPool{
					ExpireAt:     15,
					AllocationID: "a2",
					Blobbers: blobberPools{
						&blobberPool{BlobberID: "b1", Balance: 101},
						&blobberPool{BlobberID: "b2", Balance: 152},
					},
				},
				&allocationPool{
					ExpireAt:     210,
					AllocationID: "a2",
					Blobbers: blobberPools{
						&blobberPool{BlobberID: "b1", Balance: 0},
					},
				},
				&allocationPool{
					ExpireAt:     125,
					AllocationID: "a2",
					Blobbers: blobberPools{
						&blobberPool{BlobberID: "b1", Balance: 103},
						&blobberPool{BlobberID: "b2", Balance: 154},
					},
				},
				&allocationPool{
					ExpireAt:     3,
					AllocationID: "a3",
					Blobbers:     blobberPools{},
				},
			},
			want: allocationPools{
				&allocationPool{
					ExpireAt:     3,
					AllocationID: "a3",
					Blobbers:     blobberPools{},
				},
				&allocationPool{
					ExpireAt:     15,
					AllocationID: "a2",
					Blobbers: blobberPools{
						&blobberPool{BlobberID: "b1", Balance: 101},
						&blobberPool{BlobberID: "b2", Balance: 152},
					},
				},
				&allocationPool{
					ExpireAt:     100,
					AllocationID: "a1",
					Blobbers:     blobberPools{},
				},
				&allocationPool{
					ExpireAt:     125,
					AllocationID: "a2",
					Blobbers: blobberPools{
						&blobberPool{BlobberID: "b1", Balance: 103},
						&blobberPool{BlobberID: "b2", Balance: 154},
					},
				},
				&allocationPool{
					ExpireAt:     210,
					AllocationID: "a2",
					Blobbers: blobberPools{
						&blobberPool{BlobberID: "b1", Balance: 0},
					},
				},
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.aps.sortExpiry()
			require.EqualValues(t, tt.want, tt.aps)
		})
	}
}
