package node

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/core/datastore"
	"github.com/stretchr/testify/require"
)

func TestRequestEntityHandlerNotModified(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))

	defer svr.Close()

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	options := &SendOptions{Timeout: 3 * time.Second, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	LatestFinalizedMagicBlockRequestor := RequestEntityHandler("/v1/block/latest-finalized-magic-block",
		options, blockEntityMetadata)

	var value int
	handler := func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {
		value = 1
		return nil, nil
	}

	rhandler := LatestFinalizedMagicBlockRequestor(nil, handler)

	nd := Provider()
	nd.N2NHost = "127.0.0.1"
	ss := strings.Split(svr.URL, ":")
	var err error
	nd.Port, err = strconv.Atoi(ss[2])
	require.NoError(t, err)

	require.True(t, rhandler(context.Background(), nd))
	require.Equal(t, 0, value)
}
