package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"0chain.net/core/config"
	"0chain.net/core/encryption"
	"0chain.net/rest"
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
	"0chain.net/core/memorystore"
	"0chain.net/core/viper"
	"0chain.net/miner"
	"0chain.net/smartcontract/setupsc"
	"github.com/0chain/common/core/logging"
)

func main() {

	var (
		workdir       string
		redisHost     string
		redisPort     int
		redisTxnsHost string
		redisTxnsPort int
	)

	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	keysFile := flag.String("keys_file", "", "keys_file")
	dkgFile := flag.String("dkg_file", "", "dkg_file")
	delayFile := flag.String("delay_file", "", "delay_file")
	magicBlockFile := flag.String("magic_block_file", "", "magic_block_file")
	initialStatesFile := flag.String("initial_states", "", "initial_states")

	flag.StringVar(&workdir, "work_dir", "", "work_dir")
	flag.StringVar(&redisHost, "redis_host", "", "default redis pool host")
	flag.IntVar(&redisPort, "redis_port", 0, "default redis pool port")
	flag.StringVar(&redisTxnsHost, "redis_txns_host", "", "TransactionDB redis host")
	flag.IntVar(&redisTxnsPort, "redis_txns_port", 0, "TransactionDB redis port")

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
	initEntities(workdir, redisHost, redisPort, redisTxnsHost, redisTxnsPort)
	serverChain := chain.NewChainFromConfig()

	signatureScheme := serverChain.GetSignatureScheme()

	reader, err := readKeysFromAws()
	if err != nil {
		file, err := readKeysFromFile(keysFile)
		if err != nil {
			panic(err)
		}
		logging.Logger.Info("using miner keys from local")
		initScheme(signatureScheme, file)
		_ = file.Close()
	} else {
		logging.Logger.Info("using miner keys from aws")
		initScheme(signatureScheme, reader)
	}

	if err := node.Self.SetSignatureScheme(signatureScheme); err != nil {
		logging.Logger.Panic(fmt.Sprintf("Invalid signature scheme: %v", err))
	}

	miner.SetupMinerChain(serverChain)
	mc := miner.GetMinerChain()
	mc.SetDiscoverClients(viper.GetBool("server_chain.client.discover"))
	mc.SetGenerationTimeout(viper.GetInt("server_chain.block.generation.timeout"))
	mc.SetSyncStateTimeout(viper.GetDuration("server_chain.state.sync.timeout") * time.Second)
	mc.SetBCStuckCheckInterval(viper.GetDuration("server_chain.stuck.check_interval") * time.Second)
	mc.SetBCStuckTimeThreshold(viper.GetDuration("server_chain.stuck.time_threshold") * time.Second)
	mc.SetRetryWaitTime(viper.GetInt("server_chain.block.generation.retry_wait_time"))
	mc.SetupConfigInfoDB(workdir)
	mc.SetupStateCache()
	chain.SetServerChain(serverChain)

	miner.SetNetworkRelayTime(viper.GetDuration("network.relay_time") * time.Millisecond)
	node.ReadConfig()

	if *initialStatesFile == "" {
		*initialStatesFile = filepath.Join(workdir, viper.GetString("network.initial_states"))
	}

	initStates := state.NewInitStates()
	initStateErr := initStates.Read(*initialStatesFile)

	// if there's no magic_block_file commandline flag, use configured then
	if *magicBlockFile == "" {
		*magicBlockFile = filepath.Join(workdir, viper.GetString("network.magic_block_file"))
	}

	var magicBlock *block.MagicBlock
	dnsURL := viper.GetString("network.dns_url")
	if dnsURL == "" {
		magicBlock, err = readMagicBlock(*magicBlockFile)
		if err != nil {
			logging.Logger.Panic("can't read magic block file", zap.Error(err))
			return
		}
	} else {
		magicBlock, err = chain.GetMagicBlockFrom0DNS(dnsURL)
		if err != nil {
			logging.Logger.Panic("can't read magic block from DNS", zap.Error(err))
			return
		}
	}

	if state.Debug() {
		block.SetupStateLogger(filepath.Join(workdir, "/tmp/state.txt"))
	}

	// TODO: put it in a better place
	go mc.StartLFMBWorker(ctx)

	gb := mc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"),
		magicBlock, initStates)

	mb := mc.GetLatestMagicBlock()
	logging.Logger.Info("Miners in main", zap.Int("size", mb.Miners.Size()))

	if !mb.IsActiveNode(node.Self.Underlying().GetKey(), 0) {
		hostName, n2nHostName, portNum, path, description, err := readNonGenesisHostAndPort(keysFile)
		if err != nil {
			logging.Logger.Panic("Error reading keys file. Non-genesis miner has no host or port number",
				zap.Error(err))
		}

		logging.Logger.Info("Inside nonGenesis", zap.String("host_name", hostName),
			zap.String("n2n_host_name", n2nHostName), zap.Int("port_num", portNum), zap.String("path", path), zap.String("description", description))

		node.Self.Underlying().Host = hostName
		node.Self.Underlying().N2NHost = n2nHostName
		node.Self.Underlying().Port = portNum
		node.Self.Underlying().Path = path
		node.Self.Underlying().Description = description
	} else {
		if initStateErr != nil {
			logging.Logger.Panic("Failed to read initialStates", zap.Error(initStateErr))
		}
	}

	if node.Self.Underlying().GetKey() == "" {
		logging.Logger.Panic("node definition for self node doesn't exist")
	}
	if node.Self.Underlying().Type != node.NodeTypeMiner {
		logging.Logger.Panic("node not configured as miner")
	}
	err = common.NewError("saving self as client", "client save")
	for err != nil {
		_, err = client.PutClient(ctx, &node.Self.Underlying().Client)
	}
	if config.Development() {
		if *delayFile != "" {
			node.ReadNetworkDelays(*delayFile)
		}
	}

	mode := "main net"
	if config.Development() {
		mode = "development"
	} else if config.TestNet() {
		mode = "test net"
	}

	var address = fmt.Sprintf(":%v", node.Self.Underlying().Port)

	logging.Logger.Info("Starting miner", zap.String("build_tag", build.BuildTag), zap.String("go_version", runtime.Version()), zap.Int("available_cpus", runtime.NumCPU()), zap.String("port", address))
	logging.Logger.Info("Chain info", zap.String("chain_id", config.GetServerChainID()), zap.String("mode", mode))
	logging.Logger.Info("Self identity", zap.Int("set_index", node.Self.Underlying().SetIndex), zap.String("id", node.Self.Underlying().GetKey()))

	registerInConductor(node.Self.Underlying().GetKey())

	var server *http.Server
	var profServer *http.Server

	mux := http.NewServeMux()
	http.DefaultServeMux = mux
	common.ConfigRateLimits()

	if config.Development() {
		if viper.GetBool("development.pprof") {
			// start pprof server
			pprofMux := http.NewServeMux()
			profServer = &http.Server{
				Addr:           fmt.Sprintf(":%d", node.Self.Underlying().Port-1000),
				ReadTimeout:    30 * time.Second,
				MaxHeaderBytes: 1 << 20,
				Handler:        pprofMux,
			}
			initProfHandlers(pprofMux)
			go func() {
				err2 := profServer.ListenAndServe()
				logging.Logger.Error("Http server shut down", zap.Error(err2))
			}()
		}

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
	memorystore.GetInfo()
	initN2NHandlers(mc)

	initWorkers(ctx)

	// load previous MB and related DKG if any. Don't load the latest, since
	// it can be promoted (not finalized).
	mc.LoadMagicBlocksAndDKG(ctx)

	if err = mc.WaitForActiveSharders(ctx); err != nil {
		logging.Logger.Error("failed to wait sharders", zap.Error(err))
	}

	// TODO: all update latest magic block from sharders should be adjusted when VC is enabled
	// this is because miners will now start from the LFB it stopped, so would not start immediately
	// from the LFB from sharders, therefore, the latest magic block from sharders would be incorrect
	// if err = mc.UpdateLatestMagicBlockFromSharders(ctx); err != nil {
	// 	logging.Logger.Panic(fmt.Sprintf("can't update LFMB from sharders, err: %v", err))
	// }

	// ignoring error and without retries, restart round will resolve it
	// if there is errors
	mc.SetupLatestAndPreviousMagicBlocks(ctx)

	if err := mc.LoadMinersPublicKeys(); err != nil {
		logging.Logger.Error("failed to load miners public keys", zap.Error(err))
	}

	mb = mc.GetLatestMagicBlock()
	if mb.StartingRound == 0 && mb.IsActiveNode(node.Self.Underlying().GetKey(), mb.StartingRound) {
		genesisDKG := viper.GetInt64("network.genesis_dkg")
		var (
			oldDKGShare *bls.DKGSummary
			dkgShare    = &bls.DKGSummary{
				SecretShares: make(map[string]string),
			}
		)

		dkgShare.ID = strconv.FormatInt(mb.MagicBlockNumber, 10)
		if genesisDKG == 0 {
			oldDKGShare, err = miner.ReadDKGSummaryFile(*dkgFile)
			if err != nil {
				logging.Logger.Panic(fmt.Sprintf("Error reading DKG file. ERROR: %v", err.Error()))
			}
		} else {
			oldDKGShare, err = miner.LoadDKGSummary(ctx, strconv.FormatInt(genesisDKG, 10))
			if err != nil {
				if mc.ChainConfig.IsViewChangeEnabled() {
					logging.Logger.Error(fmt.Sprintf("Can't load genesis dkg: ERROR: %v", err.Error()))
				} else {
					logging.Logger.Panic(fmt.Sprintf("Can't load genesis dkg: ERROR: %v", err.Error()))
				}
			}
		}
		dkgShare.SecretShares = oldDKGShare.SecretShares
		mpks, err := magicBlock.Mpks.GetMpkMap()
		if err != nil {
			logging.Logger.Panic("Get mpks map failed", zap.Error(err))
		}

		if err = dkgShare.Verify(bls.ComputeIDdkg(node.Self.Underlying().GetKey()), mpks); err != nil {
			if mc.ChainConfig.IsViewChangeEnabled() {
				logging.Logger.Error("Failed to verify genesis dkg", zap.Error(err))
			} else {
				logging.Logger.Panic(fmt.Sprintf("Failed to verify genesis dkg: ERROR: %v", err.Error()))
			}

		}
		if err = miner.StoreDKGSummary(ctx, dkgShare); err != nil {
			logging.Logger.Panic(fmt.Sprintf("Failed to store genesis dkg: ERROR: %v", err.Error()))
		}

		if err := miner.SetDKG(ctx, mb); err != nil {
			logging.Logger.Panic(fmt.Sprintf("Failed to set DKG for genesis MB"))
		}
	}

	initHandlers(mc)

	go func() {
		logging.Logger.Info("Ready to listen to the requests")
		err2 := server.ListenAndServe()
		logging.Logger.Info("Http server shut down", zap.Error(err2))
	}()

	// go mc.RegisterClient()
	chain.StartTime = time.Now().UTC()

	// start restart round event worker before the StartProtocol to be able
	// to subscribe to its events
	go mc.RestartRoundEventWorker(ctx)

	miner.StartProtocol(ctx, gb)
	mc.SetStarted()
	miner.SetupWorkers(ctx)

	//if config.Development() {
	//	go TransactionGenerator(mc.Chain, workdir)
	//}

	setupSCDoneC := make(chan struct{})
	if mc.ChainConfig.IsFeeEnabled() {
		go func() {
			mc.SetupSC(ctx)
			setupSCDoneC <- struct{}{}
		}()

		// start the dkg process worker so that when view change is on, it can start to
		// process the phase events immediately.
		go mc.DKGProcess(ctx)
	}

	shutdown := common.HandleShutdown(server, []func(){
		shutdownIntegrationTests,
		done,
		func() {
			<-setupSCDoneC
		},
		chain.CloseStateDB})
	if profServer != nil {
		shutdownProf := common.HandleShutdown(profServer, nil)
		<-shutdownProf
	}
	<-shutdown
	time.Sleep(2 * time.Second)
	logging.Logger.Info("0chain miner shut down gracefully")
}

func initScheme(signatureScheme encryption.SignatureScheme, reader io.Reader) {
	err2 := signatureScheme.ReadKeys(reader)
	if err2 != nil {
		logging.Logger.Panic("Error reading keys file")
	}
	if err := node.Self.SetSignatureScheme(signatureScheme); err != nil {
		logging.Logger.Panic(fmt.Sprintf("Invalid signature scheme: %v", err))
	}
}

func readKeysFromAws() (io.Reader, error) {
	minerSecretName := os.Getenv("MINER_SECRET_NAME")
	keys, err := common.GetSecretsFromAWS(minerSecretName, "us-east-2")
	if err != nil {
		return nil, err
	}
	return strings.NewReader(keys), nil
}

func readKeysFromFile(keysFile *string) (*os.File, error) {
	reader, err := os.Open(*keysFile)
	return reader, err
}

func done() {
	mc := miner.GetMinerChain()
	mc.Stop()
}

func readNonGenesisHostAndPort(keysFile *string) (string, string, int, string, string, error) {
	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	scanner.Scan() // throw away the publickey
	scanner.Scan() // throw away the secretkey
	result := scanner.Scan()
	if !result {
		return "", "", 0, "", "", errors.New("error reading Host")
	}

	h := scanner.Text()
	logging.Logger.Info("Host inside", zap.String("host", h))

	result = scanner.Scan()
	if !result {
		return "", "", 0, "", "", errors.New("error reading n2n host")
	}

	n2nh := scanner.Text()
	logging.Logger.Info("N2NHost inside", zap.String("n2n_host", n2nh))

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
	logging.Logger.Info("Path inside", zap.String("path", path))

	result = scanner.Scan()
	if !result {
		return h, n2nh, p, path, "", nil
	}

	description := scanner.Text()
	logging.Logger.Info("Description inside", zap.String("description", description))
	return h, n2nh, p, path, description, nil

}

func initEntities(workdir string, redisHost string, redisPort int, redisTxnsHost string, redisTxnsPort int) {
	if len(redisHost) > 0 && redisPort > 0 {
		memorystore.InitDefaultPool(redisHost, redisPort)
	} else {
		//inside docker
		memorystore.InitDefaultPool(os.Getenv("REDIS_HOST"), 6379)
	}

	memoryStorage := memorystore.GetStorageProvider()

	chain.SetupEntity(memoryStorage, workdir)
	round.SetupEntity(memoryStorage)
	round.SetupVRFShareEntity(memoryStorage)
	block.SetupEntity(memoryStorage)
	block.SetupBlockSummaryEntity(memoryStorage)
	block.SetupStateChange(memoryStorage)
	state.SetupPartialState(memoryStorage)
	state.SetupStateNodes(memoryStorage)
	client.SetupEntity(memoryStorage)
	client.SetupClientDB()

	transaction.SetupTransactionDB(redisTxnsHost, redisTxnsPort)
	transaction.SetupEntity(memoryStorage)

	miner.SetupNotarizationEntity()
	miner.SetupStartChainEntity()

	ememoryStorage := ememorystore.GetStorageProvider()
	bls.SetupDKGEntity()
	bls.SetupDKGSummary(ememoryStorage)
	bls.SetupDKGDB(workdir)
	setupsc.SetupSmartContracts()

	block.SetupMagicBlockData(ememoryStorage)
	block.SetupMagicBlockDataDB(workdir)

	block.SetupDKGKeyEntity(ememoryStorage)
	block.SetupDKGKeyDB(workdir)
}

func initHandlers(c chain.Chainer) {
	if config.Development() {
		rest.SetupHandlers()
		chain.SetupDebugStateHandlers()
	}

	//common
	node.SetupHandlers()
	block.SetupHandlers()
	diagnostics.SetupHandlers()
	client.SetupHandlers()
	chain.SetupStateHandlers()

	//miner only
	chain.SetupMinerHandlers(c)
	SetupHandlers()
	transaction.SetupHandlers()
	miner.SetupHandlers()
	chain.GetServerChain().SetupMinerNodeHandlers()
}

func initN2NHandlers(c *miner.Chain) {
	node.SetupN2NHandlers()
	miner.SetupM2MReceivers(c)
	miner.SetupM2MSenders()
	miner.SetupM2SSenders()
	miner.SetupM2MRequestors()

	miner.SetupX2MResponders()
	chain.SetupX2XResponders(c.Chain)
	chain.SetupX2MRequestors()
	chain.SetupX2SRequestors()

	chain.SetupLFBTicketSender()
}

func initWorkers(ctx context.Context) {
	serverChain := chain.GetServerChain()
	serverChain.SetupWorkers(ctx)
	transaction.SetupWorkers(ctx)
}

func initProfHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", common.UserRateLimit(pprof.Index))
	mux.HandleFunc("/debug/pprof/cmdline", common.UserRateLimit(pprof.Cmdline))
	mux.HandleFunc("/debug/pprof/profile", common.UserRateLimit(pprof.Profile))
	mux.HandleFunc("/debug/pprof/symbol", common.UserRateLimit(pprof.Symbol))
	mux.HandleFunc("/debug/pprof/trace", common.UserRateLimit(pprof.Trace))
}
