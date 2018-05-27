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
	"0chain.net/encryption"
	"0chain.net/node"
	"0chain.net/transaction"
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
	block.SetupHandlers()
}

func initEntities() {
	block.SetupEntity()
	chain.SetupEntity()
	client.SetupEntity()
	transaction.SetupEntity()
}

/*Chain - the chain this miner will be working on */
var Chain string

func main() {
	host := flag.String("host", "", "hostname")
	port := flag.Int("port", 7220, "port")
	chainID := flag.String("chain", "", "chain id")
	testMode := flag.Bool("test", false, "test mode?")
	nodesFile := flag.String("nodes_file", "config/single_node.txt", "nodes_file")
	keysFile := flag.String("keys_file", "config/single_node_keys.txt", "keys_file")
	flag.Parse()

	address := fmt.Sprintf("%v:%v", *host, *port)
	chain.SetServerChainID(*chainID)
	serverChain := chain.Provider().(*chain.Chain)
	serverChain.ID = chain.GetServerChainID()
	chain.SetServerChain(serverChain)

	config.Configuration.Host = *host
	config.Configuration.Port = *port
	config.Configuration.ChainID = *chainID
	config.Configuration.TestMode = *testMode

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
		block.BLOCK_SIZE = 100
	}
	fmt.Printf("Num CPUs available %v\n", runtime.NumCPU())
	fmt.Printf("Starting %v on %v for chain %v in %v mode ...\n", os.Args[0], address, chain.GetServerChainID(), mode)

	initEntities()
	serverChain.SetupWorkers(ctx)
	node.SetupN2NHandlers()
	serverChain.SetupNodeHandlers()

	initServer()
	initHandlers()
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
	//log.Fatal(server.Serve(l))
	log.Fatal(server.ListenAndServe())
}
