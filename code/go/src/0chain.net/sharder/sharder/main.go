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
	"0chain.net/blockstore"
	"0chain.net/chain"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/persistencestore"
	"0chain.net/sharder"
	"0chain.net/transaction"
	"go.uber.org/zap"
)

func initServer() {
	// TODO; when a new server is brought up, it needs to first download all the state before it can start accepting requests
}

func initHandlers() {
	if config.Configuration.TestMode {
		http.HandleFunc("/_hash", encryption.HashHandler)
		http.HandleFunc("/_sign", common.ToJSONResponse(encryption.SignHandler))
	}
	node.SetupHandlers()
	chain.SetupHandlers()
	client.SetupHandlers()
	transaction.SetupHandlers()
	transaction.SetupSharderHandlers()
	block.SetupHandlers()
}

func initEntities() {
	//TODO: For now using memory storage, but we don't need it.
	memoryStorage := memorystore.GetStorageProvider()
	block.SetupEntity(memoryStorage)
	chain.SetupEntity(memoryStorage)
	client.SetupEntity(memoryStorage)
	transaction.SetupEntity(memoryStorage)

	persistencestore.InitSession()
	persistenceStorage := persistencestore.GetStorageProvider()
	block.SetupBlockSummaryEntity(persistenceStorage)
	transaction.SetupTxnSummaryEntity(persistenceStorage)
}

/*Chain - the chain this miner will be working on */
var Chain string

func main() {
	host := flag.String("host", "", "hostname")
	port := flag.Int("port", 7320, "port")
	chainID := flag.String("chain", "", "chain id")
	testMode := flag.Bool("test", false, "test mode?")
	nodesFile := flag.String("nodes_file", "config/single_node.txt", "nodes_file")
	keysFile := flag.String("keys_file", "config/single_node_sharder_keys.txt", "keys_file")
	induceDelay := flag.Bool("induce_delay", false, "induce_delay")
	flag.Parse()

	if *testMode {
		LoggerInit("development", "appLogs")
	} else {
		LoggerInit("production", "appLogs")
	}
	address := fmt.Sprintf("%v:%v", *host, *port)
	config.SetServerChainID(*chainID)
	serverChain := chain.Provider().(*chain.Chain)
	serverChain.ID = datastore.ToKey(config.GetServerChainID())
	serverChain.Decimals = 10
	serverChain.BlockSize = 10000

	chain.SetServerChain(serverChain)
	sharder.SetupSharderChain(serverChain)

	config.Configuration.Host = *host
	config.Configuration.Port = *port
	config.Configuration.ChainID = *chainID
	config.Configuration.TestMode = *testMode
	config.Configuration.InduceDelay = *induceDelay

	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	publicKey, privateKey := encryption.ReadKeys(reader)
	reader.Close()

	if *nodesFile == "" {
		panic("Please specify --node_file file.txt option with a file.txt containing peer nodes")
	}

	reader, err = os.Open(*nodesFile)
	if err != nil {
		panic(err)
	}
	node.ReadNodes(reader, serverChain.Miners, serverChain.Sharders, serverChain.Blobbers)
	serverChain.Miners.ComputeProperties()
	serverChain.Sharders.ComputeProperties()
	serverChain.Blobbers.ComputeProperties()
	reader.Close()
	if node.Self == nil {
		panic("node definition for self node doesn't exist")
	} else {
		if node.Self.PublicKey != publicKey {
			fmt.Printf("self: %v\n", node.Self)
			panic(fmt.Sprintf("Pulbic key from the keys file and nodes file don't match %v %v", publicKey, node.Self.PublicKey))
		}
		node.Self.SetPrivateKey(privateKey)
	}

	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()

	mode := "main net"
	if *testMode {
		mode = "test net"
		serverChain.BlockSize = 10000
	}
	Logger.Info("CPU information", zap.Int("No of CPU available", runtime.NumCPU()))
	fmt.Printf("Starting %v on %v for chain %v in %v mode ...\n", os.Args[0], address, config.GetServerChainID(), mode)

	/*
		l, err := net.Listen("tcp", address)
		if err != nil {
			log.Fatalf("Listen: %v", err)
		}
		defer l.Close()
		l = netutil.LimitListener(l, 1000)
	*/
	server := &http.Server{
		Addr:           address,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	common.HandleShutdown(server)

	blockstore.SetupFSBlockStore("data/blocks")

	initEntities()
	serverChain.SetupWorkers(ctx)
	node.SetupN2NHandlers()
	serverChain.SetupNodeHandlers()
	sharder.SetupM2SReceivers()
	sharder.SetupWorkers()

	initServer()
	initHandlers()

	fmt.Printf("Ready to listen to the requests\n")
	//log.Fatal(server.Serve(l))
	log.Fatal(server.ListenAndServe())
}
