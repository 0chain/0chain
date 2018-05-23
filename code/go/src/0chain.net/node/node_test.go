package node

import (
	"fmt"
	"os"
	"testing"

	"0chain.net/encryption"
)

func TestNodeSetup(t *testing.T) {
	sd := Node{Host: "127.0.0.1", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive, ID: "sd"}
	sd.PublicKey = encryption.Hash(sd.ID)
	Miners.AddNode(&sd)

	sb := Node{Host: "127.0.0.2", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive, ID: "sb"}
	sb.PublicKey = encryption.Hash(sb.ID)
	Miners.AddNode(&sb)

	ns := Node{Host: "127.0.0.3", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive, ID: "ns"}
	ns.PublicKey = encryption.Hash(ns.ID)
	Miners.AddNode(&ns)

	nr := Node{Host: "127.0.0.4", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive, ID: "ns"}
	nr.PublicKey = encryption.Hash(nr.ID)
	Miners.AddNode(&nr)

	gg := Node{Host: "127.0.0.5", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive, ID: "gg"}
	gg.PublicKey = encryption.Hash(gg.ID)
	Miners.AddNode(&gg)
	Miners.Print(os.Stdout)
}

func TestNodeGetRandomNodes(t *testing.T) {
	fmt.Printf("testing random\n")
	for idx, n := range Miners.GetRandomNodes(2) {
		fmt.Printf("%v: %v\n", idx, *n)
	}
}
