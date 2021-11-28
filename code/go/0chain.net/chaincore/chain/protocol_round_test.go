package chain

import (
	"context"
	"strconv"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChain_GetLatestFinalizedMagicBlockRound(t *testing.T) {
	lfmb := &block.Block{
		HashIDField: datastore.HashIDField{Hash: "lfmb"},
	}
	cases := []struct {
		Name        string
		MagicBlocks []int64
		CheckRounds []struct {
			Round     int64
			WantRound int64 //-1 from latestFinalizedMagicBlock
		}
	}{
		{
			Name:        "FromLatestFinalizedMagicBlock",
			MagicBlocks: []int64{},
			CheckRounds: []struct {
				Round     int64
				WantRound int64
			}{
				{Round: 1, WantRound: -1},
				{Round: 100, WantRound: -1},
			},
		},
		{
			Name:        "Correct",
			MagicBlocks: []int64{1, 101, 201, 301, 401},
			CheckRounds: []struct {
				Round     int64
				WantRound int64
			}{
				{Round: 1, WantRound: 1},
				{Round: 50, WantRound: 1},
				{Round: 100, WantRound: 1},
				{Round: 101, WantRound: 1},
				{Round: 102, WantRound: 1},
				{Round: 199, WantRound: 101},
				{Round: 380, WantRound: 301},
				{Round: 401, WantRound: 301},
				{Round: 502, WantRound: 401},
				{Round: 1001, WantRound: 401},
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			chain := &Chain{
				magicBlockStartingRounds: map[int64]*block.Block{},
				getLFMB:                  make(chan *block.Block),
				updateLFMB:               make(chan *updateLFMBWithReply, 1),
			}
			ctx, cancel := context.WithCancel(context.Background())
			doneC := make(chan struct{})
			go func() {
				chain.StartLFMBWorker(ctx)
				close(doneC)
			}()
			chain.updateLatestFinalizedMagicBlock(ctx, lfmb)
			for _, r := range test.MagicBlocks {
				chain.magicBlockStartingRounds[r] = &block.Block{
					HashIDField: datastore.HashIDField{Hash: strconv.FormatInt(r, 10)},
				}
			}

			for _, checkRound := range test.CheckRounds {
				mr := &round.Round{Number: checkRound.Round}
				got := chain.GetLatestFinalizedMagicBlockRound(mr.GetRoundNumber())
				require.NotNil(t, got)
				if checkRound.WantRound == -1 {
					assert.Equal(t, lfmb.Clone(), got)
				} else {
					assert.Equal(t, chain.magicBlockStartingRounds[checkRound.WantRound].Hash, got.Hash)
				}
			}

			cancel()
			<-doneC
		})
	}
}
