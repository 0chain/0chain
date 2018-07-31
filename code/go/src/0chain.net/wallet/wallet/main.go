package main

import (
	"0chain.net/client"
	"flag"
	"log"
	"os"
	"time"

	"0chain.net/logging"
	//. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/wallet"
	"fmt"
)

var my_wallet wallet.Wallet

var miners *wallet.Pool
var sharders *wallet.Pool
var blobbers *wallet.Pool

func initHandlers() {
	node.SetupHandlers()
	wallet.SetupC2MSenders()
}

func initEntities() {
	miners = &wallet.Pool{}
	sharders = &wallet.Pool{}
	blobbers = &wallet.Pool{}
	miners.Pool = node.NewPool(node.NodeTypeMiner)
	sharders.Pool = node.NewPool(node.NodeTypeSharder)
	blobbers.Pool = node.NewPool(node.NodeTypeBlobber)
	client.SetupEntityForWallet(memorystore.GetStorageProvider())
}

func init() {
	initEntities()
	initHandlers()
}

func main() {
	logging.InitLogging("development")
	nodesFile := flag.String("nodes_file", "config/single_machine_3_nodes.txt", "nodes_file")
	fmt.Println("THIS IS A FUCKING TEST!!!")
	my_wallet.Initialize()
	node.Self.SetKeys(my_wallet.PublicKey, my_wallet.PrivateKey)
	fmt.Printf("Public key: %v\n", my_wallet.PublicKey)
	fmt.Printf("Private key: %v\n", my_wallet.PrivateKey)
	fmt.Printf("Client ID: %v\n", my_wallet.ClientID)

	if *nodesFile == "" {
		panic("Please specify --nodes_file file.txt option with a file.txt containing nodes including self")
	}
	reader, err := os.Open(*nodesFile)
	if err != nil {
		log.Fatalf("%v", err)
	}

	if miners == nil {
		fmt.Println("MINERS NOT SET UP")
	}
	node.ReadNodes(reader, miners.Pool, sharders.Pool, blobbers.Pool)
	reader.Close()
	miners.Pool.ComputeProperties()
	c, _ := client.Provider().(*client.Client)
	c.ID = my_wallet.ClientID
	c.PublicKey = my_wallet.PublicKey

	//check miners
	time.Sleep(time.Second * 3)
	fmt.Printf("nodes count: %v\n", len(miners.Pool.Nodes))
	if miners.Pool.Nodes != nil {
		for _, m := range miners.Pool.Nodes {
			fmt.Printf("Miner id: %v\n", m.ID)
		}
	} else {
		fmt.Println("Miner nodes are nil")
	}

	miners.RegisterAll(wallet.PutClientSender(c))
	miners.RegisterAll(wallet.GetClientSender(c))
	fmt.Println("TEST FUCKING PASSED SO FAR!!!")
}
