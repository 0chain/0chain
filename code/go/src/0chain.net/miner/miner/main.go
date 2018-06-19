package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

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
}

func initEntities() {
	memoryStorage := memorystore.GetStorageProvider()

	chain.SetupEntity(memoryStorage)
	round.SetupEntity(memoryStorage)
	block.SetupEntity(memoryStorage)
	block.SetupBlockSummaryEntity(memoryStorage)

	client.SetupEntity(memoryStorage)
	transaction.SetupEntity(memoryStorage)

	miner.SetupNotarizationEntity()
}

/*Chain - the chain this miner will be working on */
var Chain string

func main() {
	host := flag.String("host", "", "hostname")
	port := flag.Int("port", 7220, "port")
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	nodesFile := flag.String("nodes_file", "config/single_node.txt", "nodes_file")
	keysFile := flag.String("keys_file", "config/single_node_miner_keys.txt", "keys_file")
	maxDelay := flag.Int("max_delay", 0, "max_delay")
	blockSize := flag.Int("block_size", 0, "block_size") // 0 => take from the config file
	flag.Parse()
	config.Configuration.DeploymentMode = byte(*deploymentMode)
	viper.SetDefault("server_chain.network.relay_time", 200)
	config.SetupConfig()

	if config.Development() {
		logging.InitLogging("development")
	} else {
		logging.InitLogging("production")
	}

	//TODO: for docker compose mapping, we can't use the host
	//address := fmt.Sprintf("%v:%v", *host, *port)
	address := fmt.Sprintf(":%v", *port)

	config.Configuration.Host = *host
	config.Configuration.Port = *port
	config.Configuration.ChainID = viper.GetString("server_chain.id")
	config.Configuration.MaxDelay = *maxDelay

	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	publicKey, privateKey := encryption.ReadKeys(reader)
	reader.Close()
	config.SetServerChainID(config.Configuration.ChainID)
	serverChain := chain.Provider().(*chain.Chain)
	serverChain.ID = datastore.ToKey(config.Configuration.ChainID)
	serverChain.Decimals = int8(viper.GetInt("server_chain.decimals"))
	serverChain.BlockSize = viper.GetInt32("server_chain.block.size")
	miner.SetNetworkRelayTime(viper.GetDuration("server_chain.network.relay_time") * time.Millisecond)
	if config.Development() {
		if *blockSize > 0 {
			serverChain.BlockSize = int32(*blockSize)
		}
	}

	if *nodesFile == "" {
		panic("Please specify --node_file file.txt option with a file.txt containing peer nodes")
	}
	reader, err = os.Open(*nodesFile)
	if err != nil {
		log.Fatalf("%v", err)
	}
	node.ReadNodes(reader, serverChain.Miners, serverChain.Sharders, serverChain.Blobbers)
	reader.Close()

	miner.SetupMinerChain(serverChain)
	chain.SetServerChain(&miner.GetMinerChain().Chain)

	serverChain.Miners.ComputeProperties()
	serverChain.Sharders.ComputeProperties()
	serverChain.Blobbers.ComputeProperties()

	if node.Self == nil {
		Logger.DPanic("node definition for self node doesn't exist")
	} else {
		if node.Self.PublicKey != publicKey {
			panic(fmt.Sprintf("Pulbic key from the keys file and nodes file don't match %v %v", publicKey, node.Self.PublicKey))
		}
		node.Self.SetPrivateKey(privateKey)
		Logger.Info("self identity", zap.Any("set_index", node.Self.Node.SetIndex), zap.Any("id", node.Self.Node.GetKey()))
	}

	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()
	initEntities()
	miner.GetMinerChain().SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"))

	mode := "main net"
	if config.Development() {
		mode = "development"
	} else if config.TestNet() {
		mode = "test net"
	}
	Logger.Info("CPU information", zap.Int("available_cpus", runtime.NumCPU()))
	Logger.Info("Starting miner", zap.String("port", address), zap.String("chain_id", config.GetServerChainID()), zap.String("mode", mode))

	server := &http.Server{
		Addr:           address,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	common.HandleShutdown(server)

	serverChain.SetupWorkers(ctx)
	node.SetupN2NHandlers()
	serverChain.SetupNodeHandlers()

	initServer()
	initHandlers()
	miner.SetupM2MSenders()
	miner.SetupM2MReceivers()
	miner.SetupM2SSenders()
	miner.SetupWorkers()
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
	msg := miner.BlockMessage{Type: miner.MessageStartRound, Round: msr}
	msgChannel := mc.GetBlockMessageChannel()
	msgChannel <- &msg
	mc.SendRoundStart(common.GetRootContext(), sr)
}

/*HomePageHandler - provides basic info when accessing the home page of the server */
func HomePageHandler(w http.ResponseWriter, r *http.Request) {
	mc := miner.GetMinerChain()
	fmt.Fprintf(w, "<div>Running since %v ...\n", startTime)
	fmt.Fprintf(w, "<div>Working on the chain: %v</div>\n", mc.GetKey())
	fmt.Fprintf(w, "<div>I am a %v with set rank of (%v) <ul><li>id:%v</li><li>public_key:%v</li></ul></div>\n", node.Self.GetNodeTypeName(), node.Self.SetIndex, node.Self.GetKey(), node.Self.PublicKey)
}
