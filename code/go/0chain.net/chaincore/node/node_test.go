package node

import (
	"context"
	"fmt"
	"os"
	"testing"

	"0chain.net/chaincore/client"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
)

var Miners = NewPool(NodeTypeMiner)

func init() {
	logging.InitLogging("development")
	createMiners(Miners)
}

func createMiners(np *Pool) {
	sd := Node{Host: "127.0.0.1", Port: 7071, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme1 := encryption.NewED25519Scheme()
	err := sigScheme1.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sd.SetPublicKey(sigScheme1.GetPublicKey())
	np.AddNode(&sd)

	sb := Node{Host: "127.0.0.2", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme2 := encryption.NewED25519Scheme()
	err = sigScheme2.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sb.SetPublicKey(sigScheme2.GetPublicKey())
	np.AddNode(&sb)

	ns := Node{Host: "127.0.0.3", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme3 := encryption.NewED25519Scheme()
	err = sigScheme3.GenerateKeys()
	if err != nil {
		panic(err)
	}
	ns.SetPublicKey(sigScheme3.GetPublicKey())
	np.AddNode(&ns)

	nr := Node{Host: "127.0.0.4", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme4 := encryption.NewED25519Scheme()
	err = sigScheme4.GenerateKeys()
	if err != nil {
		panic(err)
	}
	nr.SetPublicKey(sigScheme4.GetPublicKey())
	np.AddNode(&nr)

	gg := Node{Host: "127.0.0.5", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme5 := encryption.NewED25519Scheme()
	err = sigScheme5.GenerateKeys()
	if err != nil {
		panic(err)
	}
	gg.SetPublicKey(sigScheme5.GetPublicKey())
	np.AddNode(&gg)
	np.ComputeProperties()
}

func TestNodeSetup(t *testing.T) {
	Miners.Print(os.Stdout)
}

func TestNodeGetRandomNodes(t *testing.T) {
	fmt.Printf("testing random\n")
	for idx, n := range Miners.GetRandomNodes(2) {
		fmt.Printf("%v: %v\n", idx, *n)
	}
}

// TODO: Assuming node2 & 3 are running - figure out a way to make this self-contained without the dependency
func TestNode2NodeCommunication(t *testing.T) {
	common.SetupRootContext(context.Background())
	client.SetupEntity(memorystore.GetStorageProvider())

	sigScheme := encryption.NewED25519Scheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	entity := client.Provider().(*client.Client)
	entity.SetPublicKey(sigScheme.GetPublicKey())

	n1 := &Node{Type: NodeTypeMiner, Host: "", Port: 7071, Status: NodeStatusActive}
	n1.ID = "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509"
	n2 := &Node{Type: NodeTypeMiner, Host: "", Port: 7072, Status: NodeStatusActive}
	n2.ID = "5fbb6924c222e96df6c491dfc4a542e1bbfc75d821bcca992544899d62121b55"
	n3 := &Node{Type: NodeTypeMiner, Host: "", Port: 7073, Status: NodeStatusActive}
	n3.ID = "103c274502661e78a2b5c470057e57699e372a4382a4b96b29c1bec993b1d19c"

	Self = &SelfNode{}
	Self.Node = n1
	// Self.privateKey = "aa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"
	np := NewPool(NodeTypeMiner)
	np.AddNode(n1)
	np.AddNode(n2)
	np.AddNode(n3)

	options := SendOptions{MaxRelayLength: 0, CurrentRelayLength: 0, Compress: true, CODEC: datastore.CodecMsgpack}
	sendHandler := SendEntityHandler("/v1/_n2n/entity/post", &options)
	sentTo := np.SendAtleast(2, sendHandler(entity))
	for _, r := range sentTo {
		fmt.Printf("sentTo:%v\n", r.GetKey())
	}
}

func TestPoolScorer(t *testing.T) {
	sharders := NewPool(NodeTypeSharder)
	for i := 1; i <= 30; i++ {
		nd := Node{Host: fmt.Sprintf("127.0.0.%v", i), Port: 7171, Type: NodeTypeSharder, Status: NodeStatusActive}
		sigScheme := encryption.NewED25519Scheme()
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
		publicKey := sigScheme.GetPublicKey()
		nd.SetPublicKey(publicKey)
		nd.SetID(nd.GetKey())
		sharders.AddNode(&nd)
	}
	sharders.ComputeProperties()
	hashes := []string{
		"fb5a64691303a34515d547ea972bfadad10f4a287bba6c434a064b6bd42baee0",
		"73b64d8e25c570d6a537b6b2d3023a3884468487c11e53886c3b13c87a9d4892",
		"30235f5cd366fb0ef7f927b4fb4fce1cff1786e9ca6f887dac48a80e4d29ce40",
		"c31c18b1aa9eb413c1a08d9bf118a9a1acc17dbda3509ea41088a32f06c21fcf",
		"87c30da10c4b2cdc7c227a7f9bda1a15209cea028af0a16cbc55efff0a9fee40",
		"0cad4773d086e83ef1bbbeb33a3de052f19d3f610bd9fd971d42114fc5157933",
	}
	for _, hash := range hashes {
		computeScore(sharders, hash)
	}
}

func computeScore(np *Pool, hash string) {
	ps := NewHashPoolScorer(encryption.NewXORHashScorer())
	nodes := ps.ScoreHashString(np, hash)
	fmt.Printf("block hash: %v\n", hash)
	for idx, ns := range nodes {
		fmt.Printf("%2v %v %2v %v %v\n", idx, ns.Node.GetKey(), ns.Node.SetIndex, ns.Score, ns.Node.IsInTop(nodes, 8))
	}
}
