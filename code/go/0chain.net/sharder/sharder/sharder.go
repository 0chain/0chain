package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"0chain.net/core/config"
	"0chain.net/rest"
	"0chain.net/sharder/blockstore"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/build"
	"0chain.net/core/common"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"0chain.net/core/viper"
	"0chain.net/sharder"
	"0chain.net/smartcontract/setupsc"
	"github.com/0chain/common/core/logging"
	. "github.com/0chain/common/core/logging"
)

func main() {

	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	keysFile := flag.String("keys_file", "", "keys_file")
	magicBlockFile := flag.String("magic_block_file", "", "magic_block_file")
	initialStatesFile := flag.String("initial_states", "", "initial_states")
	flag.String("nodes_file", "", "nodes_file (deprecated)")
	workdir := ""
	flag.StringVar(&workdir, "work_dir", "", "work_dir")

	flag.Parse()
	config.Configuration().DeploymentMode = byte(*deploymentMode)
	config.SetupDefaultConfig()
	config.SetupConfig(workdir)
	config.SetupSmartContractConfig(workdir)
	initIntegrationsTests()

	if config.Development() {
		logging.InitLogging("development", workdir)
	} else {
		logging.InitLogging("production", workdir)
	}

	config.Configuration().ChainID = viper.GetString("server_chain.id")
	transaction.SetTxnTimeout(int64(viper.GetInt("server_chain.transaction.timeout")))
	config.SetServerChainID(config.Configuration().ChainID)
	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()
	initEntities(workdir)
	sViper := viper.Sub("storage")
	blockstore.Init(workdir, sViper)
	serverChain := chain.NewChainFromConfig()
	signatureScheme := serverChain.GetSignatureScheme()

	reader, err := readKeysFromAws()
	if err != nil {
		file, err := readKeysFromFile(keysFile)
		if err != nil {
			panic(err)
		}
		logging.Logger.Info("using sharder keys from local")
		initScheme(signatureScheme, file)
		_ = file.Close()
	} else {
		logging.Logger.Info("using sharder keys from aws")
		initScheme(signatureScheme, reader)
	}

	if err := serverChain.SetupEventDatabase(); err != nil {
		logging.Logger.Panic("Error setting up events database", zap.Error(err))
	}

	sharder.SetupSharderChain(serverChain)
	sc := sharder.GetSharderChain()
	sc.SetupConfigInfoDB(workdir)
	sc.SetSyncStateTimeout(viper.GetDuration("server_chain.state.sync.timeout") * time.Second)
	sc.SetBCStuckCheckInterval(viper.GetDuration("server_chain.stuck.check_interval") * time.Second)
	sc.SetBCStuckTimeThreshold(viper.GetDuration("server_chain.stuck.time_threshold") * time.Second)
	sc.SetupStateCache()
	chain.SetServerChain(serverChain)
	chain.SetNetworkRelayTime(viper.GetDuration("network.relay_time") * time.Millisecond)
	node.ReadConfig()

	if *initialStatesFile == "" {
		*initialStatesFile = filepath.Join(workdir, viper.GetString("network.initial_states"))

	}

	initStates := state.NewInitStates()
	initStateErr := initStates.Read(*initialStatesFile)
	if initStateErr != nil {
		Logger.Panic("Failed to read initialStates", zap.Error(initStateErr))
		return
	}

	// if there's no magic_block_file commandline flag, use configured then
	if *magicBlockFile == "" {
		*magicBlockFile = filepath.Join(workdir, viper.GetString("network.magic_block_file"))
	}

	var magicBlock *block.MagicBlock
	dnsURL := viper.GetString("network.dns_url")
	if dnsURL == "" {
		magicBlock, err = readMagicBlock(*magicBlockFile)
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
		block.SetupStateLogger(filepath.Join(workdir, "/tmp/state.txt"))
	}

	// TODO: put it in a better place
	go sc.StartLFMBWorker(ctx)

	sc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"), magicBlock, initStates)

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
	Logger.Info("Self identity", zap.Int("set_index", selfNode.SetIndex),
		zap.String("id", selfNode.GetKey()))

	registerInConductor(node.Self.Underlying().GetKey())

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
	sc.SetupHealthyRound()

	common.ConfigRateLimits()
	initN2NHandlers(sc)
	initWorkers(ctx)

	// start sharding from the LFB stored
	if err = sc.LoadLatestBlocksFromStore(common.GetRootContext()); err != nil {
		Logger.Error("load latest blocks from store: " + err.Error())
		return
	}

	Logger.Info("finish load latest blocks from store")

	sharder.SetupWorkers(ctx)

	startBlocksInfoLogs(sc)

	if serverChain.GetCurrentMagicBlock().MagicBlockNumber <
		serverChain.GetLatestMagicBlock().MagicBlockNumber {

		serverChain.SetCurrentRound(0)
	}

	initServer()
	initHandlers(sc)

	if sc.ChainConfig.IsFeeEnabled() {
		logging.Logger.Info("setting up sharder(sc)")
		go sc.SetupSC(ctx)
	}

	// Do a deep scan from finalized block till DeepWindow
	go sc.HealthCheckWorker(ctx, sharder.DeepScan) // 4) progressively checks the health for each round

	// Do a proximity scan from finalized block till ProximityWindow
	go sc.HealthCheckWorker(ctx, sharder.ProximityScan) // 4) progressively checks the health for each round

	shutdown := common.HandleShutdown(server, []func(){shutdownIntegrationTests, done, chain.CloseStateDB})
	Logger.Info("Ready to listen to the requests")
	chain.StartTime = time.Now().UTC()
	Listen(server)

	<-shutdown
	time.Sleep(2 * time.Second)
	logging.Logger.Info("0chain miner shut down gracefully")

}

func initScheme(signatureScheme encryption.SignatureScheme, reader io.Reader) {
	err2 := signatureScheme.ReadKeys(reader)
	if err2 != nil {
		Logger.Panic("Error reading keys file")
	}
	if err := node.Self.SetSignatureScheme(signatureScheme); err != nil {
		Logger.Panic(fmt.Sprintf("Invalid signature scheme: %v", err))
	}
}

func readKeysFromFile(keysFile *string) (*os.File, error) {
	reader, err := os.Open(*keysFile)
	return reader, err
}

func readKeysFromAws() (io.Reader, error) {
	sharderSecretName := os.Getenv("SHARDER_SECRET_NAME")
	keys, err := common.GetSecretsFromAWS(sharderSecretName, "us-east-2")
	if err != nil {
		return nil, err
	}
	return strings.NewReader(keys), nil
}

func Listen(server *http.Server) {
	var err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err) // fatal listening error
	}
	// (best effort) graceful shutdown
}

func done() {
	sc := sharder.GetSharderChain()
	sc.Stop()
}

func startBlocksInfoLogs(sc *sharder.Chain) {
	var (
		lfb  = sc.GetLatestFinalizedBlock()
		lfmb = sc.GetLatestFinalizedMagicBlockBrief()
	)
	if lfmb == nil {
		Logger.Error("can't get flmb brief")
		return
	}

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
	if !result {
		return "", "", 0, "", "", errors.New("error reading Host")
	}

	h := scanner.Text()
	Logger.Info("Host inside", zap.String("host", h))

	result = scanner.Scan()
	if !result {
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
	if !result {
		return h, n2nh, p, "", "", nil
	}

	path := scanner.Text()
	Logger.Info("Path inside", zap.String("path", path))

	result = scanner.Scan()
	if !result {
		return h, n2nh, p, path, "", nil
	}

	description := scanner.Text()
	Logger.Info("Description inside", zap.String("description", description))
	return h, n2nh, p, path, description, nil
}

func initHandlers(c chain.Chainer) {
	if config.Development() {
		http.HandleFunc("/_hash", common.Recover(encryption.HashHandler))
		http.HandleFunc("/_sign", common.Recover(common.ToJSONResponse(encryption.SignHandler)))
		chain.SetupDebugStateHandlers()
		rest.SetupHandlers()
	}

	// common
	node.SetupHandlers()
	sharder.SetupHandlers()
	block.SetupHandlers()
	diagnostics.SetupHandlers()
	chain.SetupStateHandlers()

	// sharder only
	chain.SetupSharderHandlers(c)
	chain.GetServerChain().SetupSharderNodeHandlers()
	chain.SetupScRestApiHandlers()
	chain.SetupSharderStateHandlers()
}

func initEntities(workdir string) {
	memoryStorage := memorystore.GetStorageProvider()

	chain.SetupEntity(memoryStorage, workdir)
	block.SetupEntity(memoryStorage)

	round.SetupRoundSummaryDB(workdir)
	block.SetupBlockSummaryDB(workdir)
	block.SetupMagicBlockMapDB(workdir)
	block.SetupBlockEventDB(workdir)

	transaction.SetupTxnSummaryDB(workdir)
	ememoryStorage := ememorystore.GetStorageProvider()
	block.SetupBlockSummaryEntity(ememoryStorage)
	block.SetupBlockEventEntity(ememoryStorage)

	block.SetupStateChange(memoryStorage)
	state.SetupPartialState(memoryStorage)
	state.SetupStateNodes(memoryStorage)
	round.SetupEntity(ememoryStorage)
	client.SetupEntity(memoryStorage)
	transaction.SetupEntity(memoryStorage)

	transaction.SetupTxnSummaryEntity(ememoryStorage)
	block.SetupMagicBlockMapEntity(ememoryStorage)

	sharder.SetupBlockSummaries()
	sharder.SetupRoundSummaries()
	setupsc.SetupSmartContracts()

	bls.SetupDKGEntity()
	bls.SetupDKGSummary(ememoryStorage)
	bls.SetupDKGDB(workdir)
	setupsc.SetupSmartContracts()

	block.SetupMagicBlockData(ememoryStorage)
	block.SetupMagicBlockDataDB(workdir)
}

func initN2NHandlers(c *sharder.Chain) {
	node.SetupN2NHandlers()
	sharder.SetupM2SReceivers()
	sharder.SetupM2SResponders(c)
	chain.SetupX2XResponders(c.Chain)
	chain.SetupX2MRequestors()
	chain.SetupX2SRequestors()
	sharder.SetupS2SRequestors()
	sharder.SetupS2SResponders()
	sharder.SetupX2SResponders()

	chain.SetupLFBTicketSender()
}

func initWorkers(ctx context.Context) {
	serverChain := chain.GetServerChain()
	serverChain.SetupWorkers(ctx)
}
