package miner

import (
	"fmt"
	"testing"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"github.com/stretchr/testify/require"
)

func TestAddVRFShareCache(t *testing.T) {
	vrfc := newVRFSharesCache()

	require.NotNil(t, vrfc.vrfShares)
	require.NotNil(t, vrfc.mutex)

	n1 := node.Node{}
	n1.ID = "share1"
	v1 := &round.VRFShare{Share: "share1"}
	v1.SetParty(&n1)

	n2 := node.Node{}
	n2.ID = "share2"
	v2 := &round.VRFShare{Share: "share2"}
	v2.SetParty(&n2)

	n3 := node.Node{}
	n3.ID = "share3"
	v3 := &round.VRFShare{Share: "share3"}
	v3.SetParty(&n3)

	vrfc.add(v1)
	vrfc.add(v2)
	vrfc.add(v3)

	require.Equal(t, 3, len(vrfc.vrfShares))
}

func TestVRFSharesCacheGetAll(t *testing.T) {
	vrfc := newVRFSharesCache()

	require.NotNil(t, vrfc.vrfShares)
	require.NotNil(t, vrfc.mutex)
	n1 := node.Node{}
	n1.ID = "share1"
	v1 := &round.VRFShare{Share: "share1"}
	v1.SetParty(&n1)

	n2 := node.Node{}
	n2.ID = "share2"
	v2 := &round.VRFShare{Share: "share2"}
	v2.SetParty(&n2)

	n3 := node.Node{}
	n3.ID = "share3"
	v3 := &round.VRFShare{Share: "share3"}
	v3.SetParty(&n3)

	vrfc.add(v1)
	vrfc.add(v2)
	vrfc.add(v3)

	vs := vrfc.getAll()
	require.Len(t, vs, 3)

	vsm := make(map[string]struct{})
	for _, v := range vs {
		vsm[v.Share] = struct{}{}
	}

	for _, s := range []string{"share1", "share2", "share3"} {
		_, ok := vsm[s]
		require.True(t, ok)
	}

}

func TestVRFSharesCacheClean(t *testing.T) {
	tt := []struct {
		name            string
		sharesNum       int
		cleanBelowCount int
		leftNum         int
	}{
		{
			name:            "clean 0",
			sharesNum:       3,
			cleanBelowCount: -1,
			leftNum:         3,
		},
		{
			name:            "clean 1",
			sharesNum:       3,
			cleanBelowCount: 0,
			leftNum:         2,
		},
		{
			name:            "clean 2",
			sharesNum:       3,
			cleanBelowCount: 1,
			leftNum:         1,
		},
		{
			name:            "clean 3",
			sharesNum:       3,
			cleanBelowCount: 3,
			leftNum:         0,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("%s %d_%d", tc.name, tc.sharesNum, tc.leftNum), func(t *testing.T) {
			vrfc := setupVRFSharesCache(t, tc.sharesNum)
			vrfc.clean(tc.cleanBelowCount)

			require.Len(t, vrfc.getAll(), tc.leftNum)
		})
	}

}

func setupVRFSharesCache(t *testing.T, n int) *vrfSharesCache {
	vrfc := newVRFSharesCache()

	require.NotNil(t, vrfc.vrfShares)
	require.NotNil(t, vrfc.mutex)
	for i := 0; i < n; i++ {
		nd := node.Node{}
		nd.ID = fmt.Sprintf("share%d", i)
		v := &round.VRFShare{Share: fmt.Sprintf("share%d", i), RoundTimeoutCount: i}
		v.SetParty(&nd)
		vrfc.add(v)
	}

	return vrfc
}
