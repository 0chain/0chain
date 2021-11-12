package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"time"

	"go.uber.org/zap"

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
	"0chain.net/core/viper"
	"0chain.net/sharder"
	"0chain.net/sharder/blockstore"
	"0chain.net/smartcontract/setupsc"
)

func main() {
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	keysFile := flag.String("keys_file", "", "keys_file")
	magicBlockFile := flag.String("magic_block_file", "", "magic_block_file")
	minioFile := flag.String("minio_file", "", "minio_file")
	initialStatesFile := flag.String("initial_states", "", "initial_states")
	flag.String("nodes_file", "", "nodes_file (deprecated)")
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

	reader, err := os.Open(*minioFile)
	if err != nil {
		panic(err)
	}

	if err != nil {
		panic(err)
	}
	reader.Close()

	config.Configuration.ChainID = viper.GetString("server_chain.id")
	transaction.SetTxnTimeout(int64(viper.GetInt("server_chain.transaction.timeout")))

	reader, err = os.Open(*keysFile)
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
	sc.SetSyncStateTimeout(viper.GetDuration("server_chain.state.sync.timeout") * time.Second)
	sc.SetBCStuckCheckInterval(viper.GetDuration("server_chain.stuck.check_interval") * time.Second)
	sc.SetBCStuckTimeThreshold(viper.GetDuration("server_chain.stuck.time_threshold") * time.Second)
	chain.SetServerChain(serverChain)
	chain.SetNetworkRelayTime(viper.GetDuration("network.relay_time") * time.Millisecond)
	node.ReadConfig()

	if *initialStatesFile == "" {
		*initialStatesFile = viper.GetString("network.initial_states")
	}

	initStates := state.NewInitStates()
	initStateErr := initStates.Read(*initialStatesFile)

	// if there's no magic_block_file commandline flag, use configured then
	if *magicBlockFile == "" {
		*magicBlockFile = viper.GetString("network.magic_block_file")
	}

	var magicBlock *block.MagicBlock
	dnsURL := viper.GetString("network.dns_url")
	if dnsURL == "" {
		magicBlock, err = chain.ReadMagicBlockFile(*magicBlockFile)
		if err != nil {
			Logger.Panic("can't read magic block file", zap.Error(err))
			return
		}
	} else {
		magicBlock, err = chain.GetMagicBlockFrom0DNS(dnsURL)
		if err != nil {
			Logger.Panic("can't read magic block from DNS", zap.Error(err))
			return
		}
	}

	if state.Debug() {
		block.SetupStateLogger("/tmp/state.txt")
	}

	blockstore.InitializeStore(viper.GetStringMap("storage"), ctx)

	sc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"),
		magicBlock, initStates)
	Logger.Info("sharder node", zap.Any("node", node.Self))

	var selfNode = node.Self.Underlying()
	if selfNode.GetKey() == "" {
		Logger.Panic("node definition for self node doesn't exist")
	}

	var mb = sc.GetLatestMagicBlock()
	if !mb.IsActiveNode(selfNode.GetKey(), 0) {
		hostName, n2nHost, portNum, path, description, err := readNonGenesisHostAndPort(keysFile)
		if err != nil {
			Logger.Panic("Error reading keys file. Non-genesis miner has no host or port number", zap.Error(err))
		}
		Logger.Info("Inside nonGenesis", zap.String("hostname", hostName), zap.Int("port Num", portNum), zap.String("path", path))
		selfNode.Host = hostName
		selfNode.N2NHost = n2nHost
		selfNode.Port = portNum
		selfNode.Type = node.NodeTypeSharder
		selfNode.Path = path
		selfNode.Description = description
	} else {
		if initStateErr != nil {
			Logger.Panic("Failed to read initialStates", zap.Any("Error", initStateErr))
		}
	}
	if selfNode.Type != node.NodeTypeSharder {
		Logger.Panic("node not configured as sharder")
	}

	mode := "main net"
	if config.Development() {
		mode = "development"
	} else if config.TestNet() {
		mode = "test net"
	}

	address := fmt.Sprintf(":%v", selfNode.Port)

	Logger.Info("Starting sharder", zap.String("build_tag", build.BuildTag),
		zap.String("go_version", runtime.Version()),
		zap.Int("available_cpus", runtime.NumCPU()),
		zap.String("port", address))
	Logger.Info("Chain info", zap.String("chain_id", config.GetServerChainID()),
		zap.String("mode", mode))
	Logger.Info("Self identity", zap.Any("set_index", selfNode.SetIndex),
		zap.Any("id", selfNode.GetKey()))

	initIntegrationsTests(node.Self.Underlying().GetKey())
	defer shutdownIntegrationTests()

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

	common.ConfigRateLimits()
	initN2NHandlers()
	initWorkers(ctx)

	// start sharding from the LFB stored
	if err = sc.LoadLatestBlocksFromStore(common.GetRootContext()); err != nil {
		Logger.Error("load latest blocks from store: " + err.Error())
		return
	}

	startBlocksInfoLogs(sc)

	if err := sc.UpdateLatestMagicBlockFromSharders(ctx); err != nil {
		Logger.Fatal("update LFMB from sharders", zap.Error(err))
	}

	if serverChain.GetCurrentMagicBlock().MagicBlockNumber <
		serverChain.GetLatestMagicBlock().MagicBlockNumber {

		serverChain.SetCurrentRound(0)
	}

	initServer()
	initHandlers()

	go sc.RegisterClient()
	if config.DevConfiguration.IsFeeEnabled {
		go sc.SetupSC(ctx)
	}

	// Do a deep scan from finalized block till DeepWindow
	go sc.HealthCheckWorker(ctx, sharder.DeepScan) // 4) progressively checks the health for each round

	// Do a proximity scan from finalized block till ProximityWindow
	go sc.HealthCheckWorker(ctx, sharder.ProximityScan) // 4) progressively checks the health for each round

	defer done(ctx)

	Logger.Info("Ready to listen to the requests")
	chain.StartTime = time.Now().UTC()
	Listen(server)
	defer server.Shutdown(ctx)

	// wait for SIGINT to exit
	common.WaitSigInt()
}

func Listen(server *http.Server) {
	var err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err) // fatal listening error
	}
	// (best effort) graceful shutdown
}

func done(ctx context.Context) {
	sc := sharder.GetSharderChain()
	sc.Stop()
}

func startBlocksInfoLogs(sc *sharder.Chain) {
	var (
		lfb  = sc.GetLatestFinalizedBlock()
		lfmb = sc.GetLatestFinalizedMagicBlockBrief()
	)

	Logger.Info("start from LFB ", zap.Int64("round", lfb.Round),
		zap.String("hash", lfb.Hash))
	Logger.Info("start from LFMB",
		zap.Int64("round", lfmb.StartingRound),
		zap.String("hash", lfmb.MagicBlockHash)) // hash of block with the magic block
}

func initServer() {
	// TODO; when a new server is brought up, it needs to first download all the state before it can start accepting requests
}

func readNonGenesisHostAndPort(keysFile *string) (string, string, int, string, string, error) {
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
		return "", "", 0, "", "", errors.New("error reading Host")
	}

	h := scanner.Text()
	Logger.Info("Host inside", zap.String("host", h))

	result = scanner.Scan()
	if result == false {
		return "", "", 0, "", "", errors.New("error reading n2n host")
	}

	n2nh := scanner.Text()
	Logger.Info("N2NHost inside", zap.String("n2n_host", n2nh))

	scanner.Scan()
	po, err := strconv.ParseInt(scanner.Text(), 10, 32)
	p := int(po)
	if err != nil {
		return "", "", 0, "", "", err
	}

	result = scanner.Scan()
	if result == false {
		return h, n2nh, p, "", "", nil
	}

	path := scanner.Text()
	Logger.Info("Path inside", zap.String("path", path))

	result = scanner.Scan()
	if result == false {
		return h, n2nh, p, path, "", nil
	}

	description := scanner.Text()
	Logger.Info("Description inside", zap.String("description", description))
	return h, n2nh, p, path, description, nil
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
	sharder.SetupX2SResponders()
}

func initWorkers(ctx context.Context) {
	serverChain := chain.GetServerChain()
	serverChain.SetupWorkers(ctx)
	sharder.SetupWorkers(ctx)
}
