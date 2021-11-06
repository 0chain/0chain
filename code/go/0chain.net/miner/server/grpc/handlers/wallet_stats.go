package handlers

import (
	"bytes"
	"context"
	"html/template"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/core/common"
	"0chain.net/core/memorystore"
	"0chain.net/miner"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/0chain/errors"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

//go:embed wallet_stats.html
var walletStats string

// GetWalletStats returns the wallet stats
func (m *minerGRPCService) GetWalletStats(ctx context.Context, req *minerproto.GetWalletStatsRequest) (*minerproto.GetWalletStatsResponse, error) {
	walletsWithTokens, walletsWithoutTokens, totalWallets, round := GetWalletTable(false)

	tmpl, err := template.New("html_form").Parse(walletStats)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse html form")
	}

	// data to insert in the HTML form
	var params = struct {
		Round                int64
		WalletsWithTokens    int64
		WalletsWithoutTokens int64
		TotalWallets         int64
	}{
		Round:                round,
		WalletsWithTokens:    walletsWithTokens,
		WalletsWithoutTokens: walletsWithoutTokens,
		TotalWallets:         totalWallets,
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, params); err != nil {
		return nil, errors.Wrap(err, "could not execute html form")
	}

	return &minerproto.GetWalletStatsResponse{
		Body: &httpbody.HttpBody{
			ContentType: "text/html;charset=UTF-8",
			Data:        output.Bytes(),
		},
	}, nil
}

// GetWalletTable - returns the wallet table: walletsWithTokens, walletsWithoutTokens, totalWallets, round.
func GetWalletTable(latest bool) (int64, int64, int64, int64) {
	c := miner.GetMinerChain().Chain
	entity := client.NewClient()
	emd := entity.GetEntityMetadata()

	ctx := memorystore.WithEntityConnection(common.GetRootContext(), emd)
	defer memorystore.Close(ctx)

	collectionName := entity.GetCollectionName()
	mstore, ok := emd.GetStore().(*memorystore.Store)
	if !ok {
		return 0, 0, 0, 0
	}

	var b *block.Block = c.GetRoundBlocks(c.GetCurrentRound() - 1)[0]
	if !latest {
		b = c.GetLatestFinalizedBlock()
	}

	var walletsWithTokens, walletsWithoutTokens, totalWallets int64
	walletsWithTokens = b.ClientState.GetNodeDB().Size(ctx)
	totalWallets = mstore.GetCollectionSize(ctx, emd, collectionName)
	walletsWithoutTokens = totalWallets - walletsWithTokens
	return walletsWithTokens, walletsWithoutTokens, totalWallets, b.Round
}
