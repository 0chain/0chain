package storagesc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_challenge_pool_moveToWritePool(t *testing.T) {

	const allocID, until, earlier = "alloc_hex", 20, 10

	var (
		wp = new(writePool)
		ap = wp.allocPool(allocID, until)
	)

	require.Nil(t, ap)

	ap = new(allocationPool)
	ap.AllocationID = allocID
	ap.ExpireAt = 0
	wp.Pools.add(ap)

	require.NotNil(t, wp.allocPool(allocID, until))
	require.NotNil(t, wp.allocPool(allocID, earlier))
}
