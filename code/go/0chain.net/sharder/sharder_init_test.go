package sharder_test

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/block"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/chain"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/client"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/config"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/node"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/round"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/state"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/transaction"
	"github.com/0chain/0chain/code/go/0chain.net/core/common"
	"github.com/0chain/0chain/code/go/0chain.net/core/ememorystore"
	"github.com/0chain/0chain/code/go/0chain.net/core/logging"
	"github.com/0chain/0chain/code/go/0chain.net/core/memorystore"
	"github.com/0chain/0chain/code/go/0chain.net/core/persistencestore"
	"github.com/0chain/0chain/code/go/0chain.net/core/viper"
	"github.com/0chain/0chain/code/go/0chain.net/sharder"
	"github.com/0chain/0chain/code/go/0chain.net/sharder/blockstore"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/setupsc"
)

func init() {
	var (
		deploymentMode byte = 0
		magicBlockFile      = flag.String("magic_block_file", "", "magic_block_file")
	)
	config.Configuration.DeploymentMode = deploymentMode
	config.SetupDefaultConfig()

	config.Configuration.ChainID = viper.GetString("server_chain.id")
	transaction.SetTxnTimeout(int64(viper.GetInt("server_chain.transaction.timeout")))

	config.SetServerChainID(config.Configuration.ChainID)
	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()
	initEntities()
	serverChain := chain.NewChainFromConfig()
	signatureScheme := serverChain.GetSignatureScheme()

	node.Self.SetSignatureScheme(signatureScheme)

	sharder.SetupSharderChain(serverChain)
	sc := sharder.GetSharderChain()
	sc.SetupConfigInfoDB()
	chain.SetServerChain(serverChain)
	chain.SetNetworkRelayTime(viper.GetDuration("network.relay_time") * time.Millisecond)
	node.ReadConfig()

	// if there's no magic_block_file commandline flag, use configured then
	if *magicBlockFile == "" {
		*magicBlockFile = viper.GetString("network.magic_block_file")
	}

	setupBlockStorageProvider()

	var selfNode = node.Self.Underlying()
	if selfNode.GetKey() == "" {
		logging.Logger.Panic("node definition for self node doesn't exist")
	}

	// start sharding from the LFB stored
	if err := sc.LoadLatestBlocksFromStore(common.GetRootContext()); err != nil {
		logging.Logger.Error("load latest blocks from store: " + err.Error())
		return
	}

	//startBlocksInfoLogs(sc)

	common.ConfigRateLimits()

	if serverChain.GetCurrentMagicBlock().MagicBlockNumber <
		serverChain.GetLatestMagicBlock().MagicBlockNumber {

		serverChain.SetCurrentRound(0)
	}

	sharder.GetSharderChain().HealthCheckSetup(ctx, sharder.DeepScan)
	sharder.GetSharderChain().HealthCheckSetup(ctx, sharder.ProximityScan)

	defer done()

	chain.StartTime = time.Now().UTC()

	logging.InitLogging("development")
}

func done() {
	sc := sharder.GetSharderChain()
	sc.Stop()
}

func initEntities() {
	if err := os.MkdirAll("data/rocksdb/state", 0700); err != nil {
		panic(err)
	}

	memoryStorage := memorystore.GetStorageProvider()

	chain.SetupEntity(memoryStorage)
	block.SetupEntity(memoryStorage)

	round.SetupRoundSummaryDB()
	block.SetupBlockSummaryDB()

	ememoryStorage := ememorystore.GetStorageProvider()
	block.SetupBlockSummaryEntity(ememoryStorage)
	block.SetupStateChange(memoryStorage)
	state.SetupPartialState(memoryStorage)
	state.SetupStateNodes(memoryStorage)
	round.SetupEntity(ememoryStorage)
	client.SetupEntity(memoryStorage)
	transaction.SetupEntity(memoryStorage)

	//persistencestore.InitSession()
	persistenceStorage := persistencestore.GetStorageProvider()
	transaction.SetupTxnSummaryEntity(persistenceStorage)
	transaction.SetupTxnConfirmationEntity(persistenceStorage)
	block.SetupMagicBlockMapEntity(persistenceStorage)

	sharder.SetupBlockSummaries()
	sharder.SetupRoundSummaries()
	setupsc.SetupSmartContracts()
}

func setupBlockStorageProvider() {
	fsbs := blockstore.NewFSBlockStore("data/blocks", nil)
	blockStorageProvider := viper.GetString("server_chain.block.storage.provider")
	switch blockStorageProvider {
	case "", "blockstore.FSBlockStore":
		blockstore.SetupStore(fsbs)
	case "blockstore.BlockDBStore":
		blockstore.SetupStore(blockstore.NewBlockDBStore(fsbs))
	case "blockstore.MultiBlockstore":
		var bs = []blockstore.BlockStore{
			fsbs,
			blockstore.NewBlockDBStore(
				blockstore.NewFSBlockStore("data/blocksdb", nil),
			),
		}
		blockstore.SetupStore(blockstore.NewMultiBlockStore(bs))
	default:
		panic(fmt.Sprintf("uknown block store provider - %v", blockStorageProvider))
	}
}
