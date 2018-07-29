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
	"0chain.net/chain"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
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

var startTime time.Time

func initServer() {
	// TODO; when a new server is brought up, it needs to first download all the state before it can start accepting requests
	time.Sleep(time.Second)
}

func initHandlers() {
	if config.Development() {
		http.HandleFunc("/_hash", encryption.HashHandler)
		http.HandleFunc("/_sign", common.ToJSONResponse(encryption.SignHandler))
		http.HandleFunc("/_start", StartChainHandler)
	}
	http.HandleFunc("/", HomePageHandler)
	node.SetupHandlers()
	chain.SetupHandlers()
	client.SetupHandlers()
	transaction.SetupHandlers()
	block.SetupHandlers()
	miner.SetupHandlers()
}

func initEntities() {
	memoryStorage := memorystore.GetStorageProvider()

	chain.SetupEntity(memoryStorage)
	round.SetupEntity(memoryStorage)
	block.SetupEntity(memoryStorage)
	block.SetupBlockSummaryEntity(memoryStorage)

	client.SetupEntity(memoryStorage)

	transaction.SetupTransactionDB()
	transaction.SetupEntity(memoryStorage)

	miner.SetupNotarizationEntity()
}

/*Chain - the chain this miner will be working on */
var  Chain string

func main() {
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	nodesFile := flag.String("nodes_file", "config/single_node.txt", "nodes_file")
	keysFile := flag.String("keys_file", "config/single_node_miner_keys.txt", "keys_file")
	maxDelay := flag.Int("max_delay", 0, "max_delay")
	flag.Parse()
	config.Configuration.DeploymentMode = byte(*deploymentMode)
	viper.SetDefault("server_chain.network.relay_time", 200)
	viper.SetDefault("logging.level", "info")
	config.SetupConfig()

	if config.Development() {
		logging.InitLogging("development")
	} else {
		logging.InitLogging("production")
	}

	config.Configuration.ChainID = viper.GetString("server_chain.id")
	config.Configuration.MaxDelay = *maxDelay

	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	error, publicKey, privateKey := encryption.ReadKeys(reader)
	if(error == false) {
		Logger.Info("Public key in Keys file =%v", zap.String("publicKey", publicKey))
		Logger.Panic("Error reading keys file")
	}

	node.Self.SetKeys(publicKey, privateKey)
	reader.Close()
	config.SetServerChainID(config.Configuration.ChainID)

	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()
	initEntities()
	serverChain := chain.NewChainFromConfig()
	miner.SetupMinerChain(serverChain)
	serverChain = &miner.GetMinerChain().Chain
	chain.SetServerChain(serverChain)

	miner.SetNetworkRelayTime(viper.GetDuration("server_chain.network.relay_time") * time.Millisecond)
	node.SetMaxConcurrentRequests(viper.GetInt("server_chain.network.max_concurrent_requests"))

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

	serverChain.Miners.ComputeProperties()
	serverChain.Sharders.ComputeProperties()
	serverChain.Blobbers.ComputeProperties()
	miner.GetMinerChain().SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"))

	mode := "main net"
	if config.Development() {
		mode = "development"
	} else if config.TestNet() {
		mode = "test net"
	}
	Logger.Info("Starting miner", zap.Int("available_cpus", runtime.NumCPU()), zap.String("port", address), zap.String("chain_id", config.GetServerChainID()), zap.String("mode", mode))

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
	serverChain.SetupWorkers(ctx)
	node.SetupN2NHandlers()
	serverChain.SetupNodeHandlers()

	initHandlers()
	miner.SetupM2MSenders()
	miner.SetupM2MReceivers()
	miner.SetupM2SSenders()
	miner.SetupWorkers()
	initServer()
	go StartProtocol()
	Logger.Info("Ready to listen to the requests")
	startTime = time.Now().UTC()
	log.Fatal(server.ListenAndServe())
}

/*StartChainHandler - start the chain if it's at Genesis round */
func StartChainHandler(w http.ResponseWriter, r *http.Request) {
	StartProtocol()
}

/*StartProtocol - start the miner protocol */
func StartProtocol() {
	mc := miner.GetMinerChain()
	mc.Initialize()
	mc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"))
	if mc.GetRound(1) != nil {
		return
	}
	sr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
	sr.Number = 1

	//TODO: For now, hardcoding a random seed for the first round
	sr.RandomSeed = 839695260482366265
	sr.ComputeRanks(mc.Miners.Size())
	msr := mc.CreateRound(sr)

	active := mc.Miners.GetActiveCount()
	if 3*active < 2*mc.Miners.Size() {
		ticker := time.NewTicker(5 * chain.DELTA)
		for ts := range ticker.C {
			Logger.Debug("waiting for sufficient nodes", zap.Time("ts", ts), zap.Int("active", active))
			if mc.CurrentRound != 0 {
				break
			}
			if mc.CanStartNetwork() {
				break
			}
		}
	}
	if config.Development() {
		go TransactionGenerator(mc.BlockSize)
	}
	msg := miner.BlockMessage{Type: miner.MessageStartRound, Round: msr}
	msgChannel := mc.GetBlockMessageChannel()
	if mc.CurrentRound == 0 {
		Logger.Info("starting the blockchain ...")
		msgChannel <- &msg
		mc.SendRoundStart(common.GetRootContext(), sr)
	}
}

/*HomePageHandler - provides basic info when accessing the home page of the server */
func HomePageHandler(w http.ResponseWriter, r *http.Request) {
	mc := miner.GetMinerChain()
	fmt.Fprintf(w, "<div>Running since %v ...\n", startTime)
	fmt.Fprintf(w, "<div>Working on the chain: %v</div>\n", mc.GetKey())
	fmt.Fprintf(w, "<div>I am a %v with set rank of (%v) <ul><li>id:%v</li><li>public_key:%v</li></ul></div>\n", node.Self.GetNodeTypeName(), node.Self.SetIndex, node.Self.GetKey(), node.Self.PublicKey)
}
