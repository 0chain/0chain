package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	_ "net/http/pprof"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/diagnostics"
	"0chain.net/encryption"
	"0chain.net/logging"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/miner"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/transaction"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	nodesFile := flag.String("nodes_file", "config/single_node.txt", "nodes_file")
	keysFile := flag.String("keys_file", "config/single_node_miner_keys.txt", "keys_file")
	maxDelay := flag.Int("max_delay", 0, "max_delay")
	flag.Parse()
	config.Configuration.DeploymentMode = byte(*deploymentMode)
	config.SetupDefaultConfig()
	config.SetupConfig()

	if config.Development() {
		logging.InitLogging("development")
	} else {
		logging.InitLogging("production")
	}

	config.Configuration.ChainID = viper.GetString("server_chain.id")
	config.Configuration.MaxDelay = *maxDelay
	transaction.SetTxnTimeout(int64(viper.GetInt("server_chain.txn_timeout")))

	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}

	signatureScheme := encryption.NewED25519Scheme()
	err = signatureScheme.ReadKeys(reader)
	if err != nil {
		Logger.Panic("Error reading keys file")
	}
	node.Self.SetSignatureScheme(signatureScheme)
	reader.Close()
	config.SetServerChainID(config.Configuration.ChainID)

	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()
	initEntities()
	serverChain := chain.NewChainFromConfig()
	miner.SetupMinerChain(serverChain)
	mc := miner.GetMinerChain()
	mc.DiscoverClients = viper.GetBool("server_chain.client.discover")
	mc.GenerateTimeout = viper.GetInt("server_chain.generate_timeout")
	chain.SetServerChain(serverChain)

	miner.SetNetworkRelayTime(viper.GetDuration("network.relay_time") * time.Millisecond)
	node.ReadConfig()

	if *nodesFile == "" {
		panic("Please specify --nodes_file file.txt option with a file.txt containing nodes including self")
	}
	if strings.HasSuffix(*nodesFile, "txt") {
		reader, err = os.Open(*nodesFile)
		if err != nil {
			log.Fatalf("%v", err)
		}
		node.ReadNodes(reader, serverChain.Miners, serverChain.Sharders, serverChain.Blobbers)
		reader.Close()
	} else {
		mc.ReadNodePools(*nodesFile)
	}
	if node.Self.ID == "" {
		Logger.Panic("node definition for self node doesn't exist")
	}

	Logger.Info("self identity", zap.Any("set_index", node.Self.Node.SetIndex), zap.Any("id", node.Self.Node.GetKey()))

	if config.DevConfiguration.State {
		chain.SetupStateLogger("/tmp/state.txt")
	}

	mc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"))

	mode := "main net"
	if config.Development() {
		mode = "development"
	} else if config.TestNet() {
		mode = "test net"
	}

	address := fmt.Sprintf(":%v", node.Self.Port)
	Logger.Info("Starting miner", zap.Int("available_cpus", runtime.NumCPU()), zap.String("port", address), zap.String("chain_id", config.GetServerChainID()), zap.String("mode", mode))

	//TODO - get stake of miner from biding (currently hard coded)
	//serverChain.updateMiningStake(node.Self.Node.GetKey(), 100)  we do not want to expose this feature at this point.
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
	memorystore.GetInfo()

	initWorkers(ctx)
	initN2NHandlers()
	initServer()
	initHandlers()

	go StartProtocol(ctx)
	Logger.Info("Ready to listen to the requests")
	chain.StartTime = time.Now().UTC()
	log.Fatal(server.ListenAndServe())
}

func initServer() {
	// TODO; when a new server is brought up, it needs to first download all the state before it can start accepting requests
	time.Sleep(time.Second)
}

func initEntities() {
	memoryStorage := memorystore.GetStorageProvider()

	chain.SetupEntity(memoryStorage)
	round.SetupEntity(memoryStorage)
	round.SetupVRFShareEntity(memoryStorage)
	block.SetupEntity(memoryStorage)
	block.SetupBlockSummaryEntity(memoryStorage)

	client.SetupEntity(memoryStorage)

	transaction.SetupTransactionDB()
	transaction.SetupEntity(memoryStorage)

	miner.SetupNotarizationEntity()
}

func initHandlers() {
	if config.Development() {
		http.HandleFunc("/_hash", encryption.HashHandler)
		http.HandleFunc("/_sign", common.ToJSONResponse(encryption.SignHandler))
	}
	config.SetupHandlers()
	node.SetupHandlers()
	chain.SetupHandlers()
	client.SetupHandlers()
	transaction.SetupHandlers()
	block.SetupHandlers()
	miner.SetupHandlers()
	diagnostics.SetupHandlers()
	chain.SetupStateHandlers()

	serverChain := chain.GetServerChain()
	serverChain.SetupNodeHandlers()
}

func initN2NHandlers() {
	node.SetupN2NHandlers()
	miner.SetupM2MReceivers()
	miner.SetupM2MSenders()
	miner.SetupM2SSenders()
	miner.SetupM2SRequestors()

	miner.SetupX2MResponders()
	chain.SetupX2MRequestors()
}

func initWorkers(ctx context.Context) {
	serverChain := chain.GetServerChain()
	serverChain.SetupWorkers(ctx)
	miner.SetupWorkers(ctx)
	transaction.SetupWorkers(ctx)
}

/*StartProtocol - start the miner protocol */
func StartProtocol(ctx context.Context) {
	mc := miner.GetMinerChain()

	mc.Sharders.OneTimeStatusMonitor(ctx)
	lfBlocks := mc.GetLatestFinalizedBlockFromSharder(ctx)

	var lfb *block.Block
	//Sorting as per the latest finalized blocks from all the sharders
	sort.Slice(lfBlocks, func(i int, j int) bool { return lfBlocks[i].Round >= lfBlocks[j].Round })
	if len(lfBlocks) > 0 {
		lfb = lfBlocks[0]
	}

	sr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
	if lfb != nil {
		mc.SetLatestFinalizedBlock(ctx, lfb)
		sr.Number = lfb.Round
		mc.SetRandomSeed(sr, lfb.RoundRandomSeed)
	} else {
		sr.Number = 0
	}
	mr := mc.CreateRound(sr)

	if !mc.CanStartNetwork() {
		ticker := time.NewTicker(5 * chain.DELTA)
		for ts := range ticker.C {
			active := mc.Miners.GetActiveCount()
			Logger.Info("waiting for sufficient active nodes", zap.Time("ts", ts), zap.Int("active", active))
			if mc.CanStartNetwork() {
				break
			}
		}
	}
	if config.Development() {
		go TransactionGenerator(mc.BlockSize)
	}
	Logger.Info("starting the blockchain ...", zap.Int64("round", mr.GetRoundNumber()))
	mc.StartNextRound(ctx, mr)
}
