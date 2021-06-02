// +build integration_tests

package chain

import (
	"context"
	"net/http"

	crpc "github.com/0chain/0chain/code/go/0chain.net/conductor/conductrpc"
)

func revertString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

/*LatestFinalizedBlockHandler - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (
	interface{}, error) {

	var state = crpc.Client().State()
	if state.FinalizedBlock != nil {
		// bad
		var lfbs = GetServerChain().GetLatestFinalizedBlockSummary()
		lfbs.Hash = revertString(lfbs.Hash)
		return lfbs, nil
	}

	return GetServerChain().GetLatestFinalizedBlockSummary(), nil
}

/*LatestFinalizedMagicBlockHandler - provide the latest finalized magic block by this miner */
func LatestFinalizedMagicBlockHandler(ctx context.Context, r *http.Request) (
	interface{}, error) {

	var state = crpc.Client().State()
	if state.MagicBlock != nil {
		var lfmb = GetServerChain().GetLatestFinalizedMagicBlock()
		lfmb.Hash = revertString(lfmb.Hash)
		return lfmb, nil
	}

	return GetServerChain().GetLatestFinalizedMagicBlock(), nil
}
