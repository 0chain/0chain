package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	_ "net/http/pprof"

	"0chain.net/block"
	"0chain.net/blockstore"
	"0chain.net/chain"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/ememorystore"
	"0chain.net/encryption"
	"0chain.net/logging"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/persistencestore"
	"0chain.net/round"
	"0chain.net/sharder"
	"0chain.net/transaction"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func initServer() {
	// TODO; when a new server is brought up, it needs to first download all the state before it can start accepting requests
}

func initHandlers() {
	if config.Development() {
		http.HandleFunc("/_hash", encryption.HashHandler)
		http.HandleFunc("/_sign", common.ToJSONResponse(encryption.SignHandler))
		http.HandleFunc("/_start", StartChainHandler)
	}
	node.SetupHandlers()
	chain.SetupHandlers()
	client.SetupHandlers()
	transaction.SetupHandlers()
	transaction.SetupSharderHandlers()
	block.SetupHandlers()
	sharder.SetupHandlers()
}

func initEntities() {
	//TODO: For now using memory storage, but we don't need it.
	memoryStorage := memorystore.GetStorageProvider()
	chain.SetupEntity(memoryStorage)
	round.SetupEntity(memoryStorage)
	block.SetupEntity(memoryStorage)

	block.SetupBlockSummaryDB()
	ememoryStorage := ememorystore.GetStorageProvider()
	block.SetupBlockSummaryEntity(ememoryStorage)

	client.SetupEntity(memoryStorage)
	transaction.SetupEntity(memoryStorage)

	persistencestore.InitSession()
	persistenceStorage := persistencestore.GetStorageProvider()
	transaction.SetupTxnSummaryEntity(persistenceStorage)
	transaction.SetupTxnConfirmationEntity(persistenceStorage)
}

/*Chain - the chain this miner will be working on */
var Chain string

func main() {
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	nodesFile := flag.String("nodes_file", "config/single_node.txt", "nodes_file")
	keysFile := flag.String("keys_file", "config/single_node_sharder_keys.txt", "keys_file")
	maxDelay := flag.Int("max_delay", 0, "max_delay")
	flag.Parse()
	viper.SetDefault("server_chain.network.relay_time", 200)
	viper.SetDefault("logging.level", "info")
	config.SetupConfig()

	if *deploymentMode == 0 {
		logging.InitLogging("development")
	} else {
		logging.InitLogging("production")
	}

	config.Configuration.ChainID = viper.GetString("server_chain.id")
	config.Configuration.DeploymentMode = byte(*deploymentMode)
	config.Configuration.MaxDelay = *maxDelay

	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	_, publicKey, privateKey := encryption.ReadKeys(reader)
	node.Self.SetKeys(publicKey, privateKey)
	reader.Close()

	config.SetServerChainID(config.Configuration.ChainID)
	serverChain := chain.Provider().(*chain.Chain)
	serverChain.ID = datastore.ToKey(config.Configuration.ChainID)
	serverChain.Decimals = int8(viper.GetInt("server_chain.decimals"))
	serverChain.BlockSize = viper.GetInt32("server_chain.block.size")
	serverChain.NumGenerators = viper.GetInt("server_chain.block.generators")
	serverChain.NotarizationThreshold = viper.GetInt("server_chain.block.notarization_threshold")
	chain.SetNetworkRelayTime(viper.GetDuration("server_chain.network.relay_time") * time.Millisecond)

	if *nodesFile == "" {
		panic("Please specify --nodes_file file.txt option with a file.txt containing nodes including self")
	}
	reader, err = os.Open(*nodesFile)
	if err != nil {
		log.Fatalf("%v", err)
	}
	node.ReadNodes(reader, serverChain.Miners, serverChain.Sharders, serverChain.Blobbers)
	reader.Close()
	if node.Self.ID == "" {
		Logger.Panic("node definition for self node doesn't exist")
	} else {
		Logger.Info("self identity", zap.Any("set_index", node.Self.Node.SetIndex), zap.Any("id", node.Self.Node.GetKey()))
	}
	address := fmt.Sprintf(":%v", node.Self.Port)

	sharder.SetupSharderChain(serverChain)
	chain.SetServerChain(&sharder.GetSharderChain().Chain)

	serverChain.Miners.ComputeProperties()
	serverChain.Sharders.ComputeProperties()
	serverChain.Blobbers.ComputeProperties()

	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()

	mode := "main net"
	if config.Development() {
		mode = "development"
	} else if config.TestNet() {
		mode = "test net"
	}
	Logger.Info("CPU information", zap.Int("No of CPU available", runtime.NumCPU()))
	Logger.Info("Starting sharder", zap.String("port", address), zap.String("chain_id", config.GetServerChainID()), zap.String("mode", mode))

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

	blockstore.SetupFSBlockStore("data/blocks")

	initEntities()
	sharder.GetSharderChain().SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"))

	serverChain.SetupWorkers(ctx)
	node.SetupN2NHandlers()
	serverChain.SetupNodeHandlers()
	sharder.SetupM2SReceivers()
	sharder.SetupWorkers()

	initServer()
	initHandlers()

	Logger.Info("Ready to listen to the requests")
	//log.Fatal(server.Serve(l))
	log.Fatal(server.ListenAndServe())
}

/*StartChainHandler - start the chain (for now just clears the state) */
func StartChainHandler(w http.ResponseWriter, r *http.Request) {
	sharder.ClearWorkerState()
	sc := sharder.GetSharderChain()
	sc.Initialize()
	sc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"))
}
