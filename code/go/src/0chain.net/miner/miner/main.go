package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/client"
	"0chain.net/config"
	"0chain.net/encryption"
	"0chain.net/transaction"
)

func initServer() {
	// TODO; when a new server is brought up, it needs to first download all the state before it can start accepting requests
}

func initHandlers() {
	if config.Configuration.TestMode {
		http.HandleFunc("/_hash", encryption.HashHandler)
	}
	chain.SetupHandlers()
	client.SetupHandlers()
	transaction.SetupHandlers()
	block.SetupHandlers()
}

/*Chain - the chain this miner will be working on */
var Chain string

func main() {
	host := flag.String("host", "", "hostname")
	port := flag.Int("port", 7220, "port")
	chainID := flag.String("chain", "", "chain id")
	testmode := flag.Bool("test", false, "test mode?")
	flag.Parse()
	address := fmt.Sprintf("%v:%v", *host, *port)
	chain.SetServerChainID(*chainID)
	config.Configuration.Host = *host
	config.Configuration.Port = *port
	config.Configuration.ChainID = *chainID
	config.Configuration.TestMode = *testmode
	mode := "main net"
	if *testmode {
		mode = "test net"
	}
	fmt.Printf("Num CPUs available %v\n", runtime.NumCPU())
	//runtime.GOMAXPROCS(1)
	fmt.Printf("Starting %v on %v for chain %v in %v mode ...\n", os.Args[0], address, chain.GetServerChainID(), mode)
	initServer()
	initHandlers()
	if err := http.ListenAndServe(address, nil); err != nil {
		panic(err)
	}
}
