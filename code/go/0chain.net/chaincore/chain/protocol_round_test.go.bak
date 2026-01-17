package chain

import (
	"0chain.net/chaincore/node"
	"0chain.net/core/config"
	"0chain.net/core/memorystore"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/setupsc"
	"github.com/0chain/common/core/logging"

	"context"
	"fmt"
	"strconv"
	"testing"

	"0chain.net/core/common"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	config.SetupDefaultConfig()
	viper.Set("server_chain.smart_contract.faucet", true)
	viper.Set("server_chain.smart_contract.miner", true)
	viper.Set("server_chain.smart_contract.storage", true)
	viper.Set("server_chain.smart_contract.vesting", true)
	viper.Set("server_chain.smart_contract.zcn", true)
	viper.Set("server_chain.smart_contract.multisig", true)
	config.SmartContractConfig = viper.New()
	config.SmartContractConfig.Set("smart_contracts.faucetsc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.minersc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.vestingsc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.storagesc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")

	setupsc.SetupSmartContracts()
	logging.InitLogging("development", "")
	common.ConfigRateLimits()
	block.SetupEntity(memorystore.GetStorageProvider())
}

func TestChain_GetLatestFinalizedMagicBlockRound(t *testing.T) {
	lfmb := &block.Block{
		HashIDField: datastore.HashIDField{Hash: "lfmb"},
	}
	cancel, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	common.SetupRootContext(cancel)
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
			chain := NewChainFromConfig()

			//chain := &Chain{
			//	magicBlockStartingRoundsMap: map[int64]*block.Block{},
			//	getLFMB:                     make(chan *block.Block),
			//	updateLFMB:                  make(chan *updateLFMBWithReply, 1),
			//}
			chain.Initialize()
			mb := block.NewMagicBlock()
			mb.Miners = node.NewPool(node.NodeTypeMiner)
			mb.Miners.NodesMap = make(map[string]*node.Node)
			fmt.Println("len(mb.Miners.NodesMap)", len(mb.Miners.NodesMap))
			fmt.Println("Size", mb.Miners.Size())
			fmt.Println("MinGenerators", chain.MinGenerators())
			fmt.Println("GeneratorPercent", chain.GeneratorsPercent())
			mb.Sharders = node.NewPool(node.NodeTypeSharder)
			mb.Sharders.NodesMap = make(map[string]*node.Node)
			chain.SetMagicBlock(mb)
			lfmb.MagicBlock = mb

			ctx, cancel := context.WithCancel(context.Background())
			doneC := make(chan struct{})
			go func() {
				chain.StartLFMBWorker(ctx)
				close(doneC)
			}()
			chain.updateLatestFinalizedMagicBlock(ctx, lfmb)
			for _, r := range test.MagicBlocks {
				chain.magicBlockStartingRoundsMap[r] = &block.Block{
					HashIDField: datastore.HashIDField{Hash: strconv.FormatInt(r, 10)},
				}
				chain.magicBlockStartingRounds.Add(r)
			}

			for _, checkRound := range test.CheckRounds {
				mr := &round.Round{Number: checkRound.Round}
				got := chain.GetLatestFinalizedMagicBlockRound(mr.GetRoundNumber())
				require.NotNil(t, got)
				if checkRound.WantRound == -1 {
					assert.Equal(t, lfmb, got)
				} else {
					assert.Equal(t, chain.magicBlockStartingRoundsMap[checkRound.WantRound].Hash, got.Hash)
				}
			}

			cancel()
			<-doneC
		})
	}
}
