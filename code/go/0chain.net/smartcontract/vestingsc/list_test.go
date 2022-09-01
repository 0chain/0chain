package vestingsc

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/0chain/common/core/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_clientPoolsKey(t *testing.T) {
	const clientID = "client_hex"
	var key = clientPoolsKey(ADDRESS, clientID)
	assert.NotZero(t, key)
	assert.True(t, strings.Contains(key, ADDRESS))
	assert.True(t, strings.Contains(key, clientID))
}

func Test_clientPools_Encode_Decode(t *testing.T) {
	var cpe, cpd clientPools
	cpe.Pools = []string{"pool_1", "pool_2"}
	require.NoError(t, cpd.Decode(cpe.Encode()))
	assert.EqualValues(t, cpe, cpd)
}

// getIndex, removeByIndex, remove, and add
func Test_clientPools(t *testing.T) {
	var (
		cpe   clientPools
		i, ok = cpe.getIndex("not_found")
	)

	assert.Zero(t, i)
	assert.False(t, ok)

	cpe.Pools = []string{"to_remove"}
	cpe.removeByIndex(0)
	assert.Len(t, cpe.Pools, 0)

	assert.False(t, cpe.remove("not_found"))

	cpe.add("a")
	cpe.add("c")
	assert.Equal(t, cpe.Pools, []string{"a", "c"})
	cpe.add("b")
	assert.Equal(t, cpe.Pools, []string{"a", "b", "c"})
	assert.False(t, cpe.add("b"))
	assert.Equal(t, cpe.Pools, []string{"a", "b", "c"})

	cpe.Pools = []string{"a", "b", "c"}
	assert.True(t, cpe.remove("b")) // middle
	cpe.Pools = []string{"a", "b", "c"}
	assert.True(t, cpe.remove("a")) // first
	cpe.Pools = []string{"a", "b", "c"}
	assert.True(t, cpe.remove("c"))  // last
	assert.False(t, cpe.remove("z")) // not found (after)
	cpe.Pools = []string{"x", "y", "z"}
	assert.False(t, cpe.remove("a")) // not found (before)
}

// getClientPoolsBytes, getClientPools, getOrCreateClientPools,
// getClientPoolsHandler and save
func TestVestingSmartContract(t *testing.T) {

	const clientID = "client_hex"

	var (
		vsc      = newTestVestingSC()
		balances = newTestBalances()
		set      *clientPools
		get      *clientPools
		err      error
	)

	_, err = vsc.getClientPools(clientID, balances)
	assert.Equal(t, util.ErrValueNotPresent, err)

	get, err = vsc.getOrCreateClientPools(clientID, balances)
	require.NoError(t, err)
	assert.Equal(t, new(clientPools), get)

	set = new(clientPools)
	set.Pools = []string{"a", "b", "c"}
	err = set.save(vsc.ID, clientID, balances)
	require.NoError(t, err)

	get, err = vsc.getClientPools(clientID, balances)
	require.NoError(t, err)
	assert.Equal(t, set, get)

	get, err = vsc.getOrCreateClientPools(clientID, balances)
	require.NoError(t, err)
	assert.Equal(t, set, get)

	var (
		ctx    = context.Background()
		params = make(url.Values)
	)
	params.Set("client_id", clientID)
	var got interface{}
	got, err = vsc.getClientPoolsHandler(ctx, params, balances)
	_, err = vsc.getOrCreateClientPools(clientID, balances)
	require.NoError(t, err)
	assert.Equal(t, set, got)
}
