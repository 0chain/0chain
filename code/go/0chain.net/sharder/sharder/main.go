package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "net/http/pprof"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/build"
	"0chain.net/core/common"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/persistencestore"
	"0chain.net/sharder"
	"0chain.net/sharder/blockstore"
	"0chain.net/smartcontract/setupsc"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	keysFile := flag.String("keys_file", "", "keys_file")
	magicBlockFile := flag.String("magic_block_file", "", "magic_block_file")
	flag.Parse()
	config.Configuration.DeploymentMode = byte(*deploymentMode)
	config.SetupDefaultConfig()
	config.SetupConfig()
	config.SetupSmartContractConfig()

	if config.Development() {
		logging.InitLogging("development")
	} else {
		logging.InitLogging("production")
	}

	config.Configuration.ChainID = viper.GetString("server_chain.id")

	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}

	config.SetServerChainID(config.Configuration.ChainID)
	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()
	initEntities()
	serverChain := chain.NewChainFromConfig()
	signatureScheme := serverChain.GetSignatureScheme()
	err = signatureScheme.ReadKeys(reader)
	if err != nil {
		Logger.Panic("Error reading keys file")
	}
	node.Self.SetSignatureScheme(signatureScheme)
	reader.Close()

	sharder.SetupSharderChain(serverChain)
	sc := sharder.GetSharderChain()
	sc.SetupConfigInfoDB()
	chain.SetServerChain(serverChain)
	chain.SetNetworkRelayTime(viper.GetDuration("network.relay_time") * time.Millisecond)
	node.ReadConfig()
	magicBlock := readMagicBlockFile(magicBlockFile, sc, serverChain)
	if state.Debug() {
		chain.SetupStateLogger("/tmp/state.txt")
	}
	setupBlockStorageProvider()
	sc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"), magicBlock)
	Logger.Info("sharder node", zap.Any("node", node.Self))
	selfNode := node.Self.Underlying()
	if selfNode.GetKey() == "" {
		Logger.Panic("node definition for self node doesn't exist")
	}
	mb := sc.GetMagicBlock()
	if !mb.IsActiveNode(selfNode.GetKey(), 0) {
		hostName, n2nHost, portNum, err := readNonGenesisHostAndPort(keysFile)
		if err != nil {
			Logger.Panic("Error reading keys file. Non-genesis miner has no host or port number", zap.Error(err))
		}
		Logger.Info("Inside nonGenesis", zap.String("hostname", hostName), zap.Int("port Num", portNum))
		selfNode.Host = hostName
		selfNode.N2NHost = n2nHost
		selfNode.Port = portNum
		selfNode.Type = node.NodeTypeSharder
	}
	if selfNode.Type != node.NodeTypeSharder {
		Logger.Panic("node not configured as sharder")
	}

	// start sharding from the LFB stored
	if err = sc.LoadLatestBlocksFromStore(common.GetRootContext()); err != nil {
		Logger.DPanic("load latest blocks from store: " + err.Error())
		return
	}

	startBlocksInfoLogs(sc)

	mode := "main net"
	if config.Development() {
		mode = "development"
	} else if config.TestNet() {
		mode = "test net"
	}

	address := fmt.Sprintf(":%v", selfNode.Port)

	Logger.Info("Starting sharder", zap.String("build_tag", build.BuildTag), zap.String("go_version", runtime.Version()), zap.Int("available_cpus", runtime.NumCPU()), zap.String("port", address))
	Logger.Info("Chain info", zap.String("chain_id", config.GetServerChainID()), zap.String("mode", mode))
	Logger.Info("Self identity", zap.Any("set_index", selfNode.SetIndex), zap.Any("id", selfNode.GetKey()))

	var server *http.Server
	if config.Development() {
		// No WriteTimeout setup to enable pprof
		server = &http.Server{
			Addr:           address,
			ReadTimeout:    30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
	} else {
		server = &http.Server{
			Addr:           address,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
	}
	common.HandleShutdown(server)
	// setupBlockStorageProvider()
	sc.SetupHealthyRound()

	initWorkers(ctx)
	common.ConfigRateLimits()
	initN2NHandlers()
	getCurrentMagicBlock(sc)
	initServer()
	initHandlers()
	go sc.RegisterClient()
	go sc.InitSetupSC()

	// Do a deep scan from finalized block till DeepWindow
	go sc.HealthCheckWorker(ctx, sharder.DeepScan) // 4) progressively checks the health for each round

	// Do a proximity scan from finalized block till ProximityWindow
	go sc.HealthCheckWorker(ctx, sharder.ProximityScan) // 4) progressively checks the health for each round

	defer done(ctx)

	Logger.Info("Ready to listen to the requests")
	chain.StartTime = time.Now().UTC()
	log.Fatal(server.ListenAndServe())
}

func done(ctx context.Context) {
	sc := sharder.GetSharderChain()
	sc.Stop()
}

func startBlocksInfoLogs(sc *sharder.Chain) {
	lfb, lfmb := sc.GetLatestFinalizedBlock(), sc.GetLatestFinalizedMagicBlock()
	Logger.Info("start from LFB ", zap.Int64("round", lfb.Round),
		zap.String("hash", lfb.Hash))
	Logger.Info("start from LFMB",
		zap.Int64("round", lfmb.MagicBlock.StartingRound),
		zap.String("hash", lfmb.Hash)) // hash of block with the magic block
}

func initServer() {
	// TODO; when a new server is brought up, it needs to first download all the state before it can start accepting requests
}

func readNonGenesisHostAndPort(keysFile *string) (string, string, int, error) {
	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	scanner.Scan() //throw away the publickey
	scanner.Scan() //throw away the secretkey
	result := scanner.Scan()
	if result == false {
		return "", "", 0, errors.New("error reading Host")
	}

	h := scanner.Text()
	Logger.Info("Host inside", zap.String("host", h))

	result = scanner.Scan()
	if result == false {
		return "", "", 0, errors.New("error reading n2n host")
	}

	n2nh := scanner.Text()
	Logger.Info("N2NHost inside", zap.String("n2n_host", n2nh))

	scanner.Scan()
	po, err := strconv.ParseInt(scanner.Text(), 10, 32)
	p := int(po)
	if err != nil {
		return "", "", 0, err
	}
	return h, n2nh, p, nil

}

func initHandlers() {
	if config.Development() {
		http.HandleFunc("/_hash", common.Recover(encryption.HashHandler))
		http.HandleFunc("/_sign", common.Recover(common.ToJSONResponse(encryption.SignHandler)))
	}
	config.SetupHandlers()
	node.SetupHandlers()
	chain.SetupHandlers()
	block.SetupHandlers()
	sharder.SetupHandlers()
	diagnostics.SetupHandlers()
	chain.SetupStateHandlers()

	serverChain := chain.GetServerChain()
	serverChain.SetupNodeHandlers()
}

func readMagicBlockFile(magicBlockFile *string, sc *sharder.Chain, serverChain *chain.Chain) *block.MagicBlock {

	magicBlockConfigFile := viper.GetString("network.magic_block_file")
	if magicBlockConfigFile == "" {
		magicBlockConfigFile = *magicBlockFile
	}
	if magicBlockConfigFile == "" {
		panic("Please specify --nodes_file file.txt option with a file.txt containing nodes including self")
	}
	if strings.HasSuffix(magicBlockConfigFile, "json") {
		mBfile, err := ioutil.ReadFile(magicBlockConfigFile)
		if err != nil {
			Logger.Panic(fmt.Sprintf("failed to read magic block file: %v", err))
		}
		mB := block.NewMagicBlock()
		err = mB.Decode([]byte(mBfile))
		if err != nil {
			Logger.Panic(fmt.Sprintf("failed to decode magic block file: %v", err))
		}
		Logger.Info("parse magic block", zap.Any("block", mB))
		mB.Hash = mB.GetHash()
		Logger.Info("number of miners", zap.Any("number of miners", mB.Miners.Size()),
			zap.Any("number of sharders", mB.Sharders.Size()))
		return mB
	} else {
		Logger.Panic(fmt.Sprintf("magic block file (%v) is in the wrong format. It should be a json", magicBlockConfigFile))
	}
	return nil
}

func getCurrentMagicBlock(sc *sharder.Chain) {
	mbs := sc.GetLatestFinalizedMagicBlockFromSharder(common.GetRootContext())
	if len(mbs) == 0 {
		Logger.DPanic("No finalized magic block from sharder")
	}
	if len(mbs) > 1 {
		sort.Slice(mbs, func(i, j int) bool {
			return mbs[i].StartingRound < mbs[j].StartingRound
		})
	}
	magicBlock := mbs[0]
	Logger.Info("get current magic block", zap.Any("magic_block", magicBlock))

	var err = sc.MustVerifyChainHistory(common.GetRootContext(), magicBlock,
		sc.SaveMagicBlockHandler)
	if err != nil {
		Logger.DPanic(fmt.Sprintf("failed to verify chain history: %v", err.Error()))
	}

	err = sc.UpdateMagicBlock(magicBlock.MagicBlock)
	if err != nil {
		Logger.DPanic(fmt.Sprintf("failed to update magic block: %v", err.Error()))
	}
	sc.SetLatestFinalizedMagicBlock(magicBlock)
}

func initEntities() {
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

	persistencestore.InitSession()
	persistenceStorage := persistencestore.GetStorageProvider()
	transaction.SetupTxnSummaryEntity(persistenceStorage)
	transaction.SetupTxnConfirmationEntity(persistenceStorage)
	block.SetupMagicBlockMapEntity(persistenceStorage)

	sharder.SetupBlockSummaries()
	sharder.SetupRoundSummaries()
	setupsc.SetupSmartContracts()
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
}

func initWorkers(ctx context.Context) {
	serverChain := chain.GetServerChain()
	serverChain.SetupWorkers(ctx)
	sharder.SetupWorkers(ctx)
}

func setupBlockStorageProvider() {
	blockStorageProvider := viper.GetString("server_chain.block.storage.provider")
	if blockStorageProvider == "" || blockStorageProvider == "blockstore.FSBlockStore" {
		blockstore.SetupStore(blockstore.NewFSBlockStore("data/blocks"))
	} else if blockStorageProvider == "blockstore.BlockDBStore" {
		blockstore.SetupStore(blockstore.NewBlockDBStore("data/blocksdb"))
	} else if blockStorageProvider == "blockstore.MultiBlockstore" {
		var bs = []blockstore.BlockStore{blockstore.NewFSBlockStore("data/blocks"), blockstore.NewBlockDBStore("data/blocksdb")}
		blockstore.SetupStore(blockstore.NewMultiBlockStore(bs))
	} else {
		panic(fmt.Sprintf("uknown block store provider - %v", blockStorageProvider))
	}
}
