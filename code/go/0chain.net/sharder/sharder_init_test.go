package sharder_test

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/spf13/viper"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/sharder"
	"0chain.net/sharder/blockstore"
	"0chain.net/smartcontract/setupsc"
)

func init() {
	var (
		deploymentMode byte = 0
		magicBlockFile      = flag.String("magic_block_file", "", "magic_block_file")
	)
	config.Configuration.DeploymentMode = deploymentMode
	config.SetupDefaultConfig()
	config.DevConfiguration.ViewChange = true
	viper.Set("minio.enabled", true)
	viper.Set("minio.worker_frequency", 1)

	config.Configuration.ChainID = viper.GetString("server_chain.id")
	transaction.SetTxnTimeout(int64(viper.GetInt("server_chain.transaction.timeout")))

	config.SetServerChainID(config.Configuration.ChainID)
	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()
	initEntities()
	viper.Set("server_chain.block.max_block_size", 100)
	serverChain := chain.NewChainFromConfig()
	signatureScheme := serverChain.GetSignatureScheme()
	err := signatureScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}

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

	// startBlocksInfoLogs(sc)
	sc.SetupHealthyRound()

	// initWorkers(ctx)
	common.ConfigRateLimits()
	initN2NHandlers()

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

func initWorkers(ctx context.Context) {
	serverChain := chain.GetServerChain()
	serverChain.SetupWorkers(ctx)
	sharder.SetupWorkers(ctx)
}

func initN2NHandlers() {
	node.SetupN2NHandlers()
	sharder.SetupM2SReceivers()
	sharder.SetupM2SResponders()
	chain.SetupX2XResponders()
	chain.SetupX2MRequestors()
	chain.SetupX2SRequestors()
	sharder.SetupS2SRequestors()
	sharder.SetupS2SResponders()
	sharder.SetupX2SResponders()
}

func done() {
	sc := sharder.GetSharderChain()
	sc.Stop()
}

func initEntities() {
	if err := os.MkdirAll("data/rocksdb/state", 0700); err != nil {
		panic(err)
	}

	memoryStorage := storeMock{}

	chain.SetupEntity(memoryStorage)
	block.SetupEntity(memoryStorage)

	round.SetupRoundSummaryDB()
	block.SetupBlockSummaryDB()

	ememoryStorage := storeMock{}
	block.SetupBlockSummaryEntity(ememoryStorage)
	block.SetupStateChange(memoryStorage)
	state.SetupPartialState(memoryStorage)
	state.SetupStateNodes(memoryStorage)
	round.SetupEntity(ememoryStorage)
	client.SetupEntity(memoryStorage)
	transaction.SetupEntity(memoryStorage)

	persistenceStorage := storeMock{}
	transaction.SetupTxnSummaryEntity(persistenceStorage)
	transaction.SetupTxnConfirmationEntity(persistenceStorage)
	block.SetupMagicBlockMapEntity(persistenceStorage)

	sharder.SetupBlockSummaries()
	sharder.SetupRoundSummaries()
	setupsc.SetupSmartContracts()
}

func setupBlockStorageProvider() {
	blockstore.SetupStore(blockStoreMock{cloud: make(map[string]struct{}), blocks: make(map[string]*block.Block)})
}
