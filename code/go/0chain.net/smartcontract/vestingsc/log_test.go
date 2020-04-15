package vestingsc

import (
	"context"
	"math/rand"
	"net/url"
	"testing"
	"time"

	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_log(t *testing.T) {

	const n = logPartLength * 3

	var (
		ctx     = context.Background()
		clients = make([]*Client, 0, n)

		vsc      = newTestVestingSC()
		balances = newTestBalances()

		resp interface{}
		err  error

		rsrc = rand.NewSource(time.Now().UnixNano())
		rndr = rand.New(rsrc)

		rnd = func() string {
			return clients[rndr.Intn(len(clients))].id
		}

		now common.Timestamp = 0
	)

	setConfig(t, balances)

	// before
	resp, err = vsc.getLastPartHandler(ctx, nil, balances)
	require.NoError(t, err)
	assert.EqualValues(t, int64(0), resp)

	resp, err = vsc.getPartHandler(ctx, url.Values{
		"part": []string{"0"},
	}, balances)
	require.NoError(t, err)
	assert.EqualValues(t, &logPart{Part: 0, Txns: nil}, resp)

	_, err = vsc.getPartHandler(ctx, url.Values{
		"part": []string{"150"},
	}, balances)
	require.Error(t, err)

	for i := 0; i < n; i++ {
		clients = append(clients, newClient(100, balances))
	}

	for _, cl := range clients {
		var d1, d2 = rnd(), rnd()
		for d1 == cl.id {
			d1 = rnd()
		}
		for d2 == cl.id {
			d2 = rnd()
		}
		_, err = cl.add(t, vsc, &addRequest{
			Description:  "for testing",
			StartTime:    10,
			Duration:     2 * time.Second,
			Friquency:    3 * time.Second,
			Destinations: []string{d1, d2},
			Amount:       10,
		}, 100, now, balances)
		require.NoError(t, err)
	}

	// after
	resp, err = vsc.getLastPartHandler(ctx, nil, balances)
	require.NoError(t, err)
	assert.EqualValues(t, int64(2), resp)

	resp, err = vsc.getPartHandler(ctx, url.Values{
		"part": []string{"2"},
	}, balances)
	require.NoError(t, err)
	require.IsType(t, &logPart{}, resp)
	var lp = resp.(*logPart)
	assert.Equal(t, int64(2), lp.Part)
	assert.Len(t, lp.Txns, logPartLength)

	_, err = vsc.getPartHandler(ctx, url.Values{
		"part": []string{"3"},
	}, balances)
	require.Error(t, err)
}
