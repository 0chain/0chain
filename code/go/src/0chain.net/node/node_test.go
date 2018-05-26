package node

import (
	"context"
	"fmt"
	"os"
	"testing"

	"0chain.net/client"
	"0chain.net/datastore"
	"0chain.net/encryption"
)

var Miners = NewPool(NodeTypeMiner)

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

/*Company - a test data type */
type Company struct {
	datastore.IDField
	Domain string `json:"domain"`
	Name   string `json:"name,omitempty"`
}

func (c *Company) GetEntityName() string {
	return "company"
}

func (c *Company) Validate(ctx context.Context) error {
	return nil
}

func (c *Company) Read(ctx context.Context, id string) error {
	return datastore.Read(ctx, id, c)
}

func (c *Company) Write(ctx context.Context) error {
	return datastore.Write(ctx, c)
}

func (c *Company) Delete(ctx context.Context) error {
	return datastore.Delete(ctx, c)
}

// TODO: Assuming node2 & 3 are running - figure out a way to make this self-contained without the dependency
func TestNode2NodeCommunication(t *testing.T) {
	publicKey, _ := encryption.GenerateKeys()
	entity := client.Provider().(*client.Client)
	entity.ID = encryption.Hash(publicKey)
	entity.PublicKey = publicKey

	n1 := &Node{ID: "node1", Type: NodeTypeMiner, Host: "", Port: 7071, Status: NodeStatusActive}
	n2 := &Node{ID: "node2", Type: NodeTypeMiner, Host: "", Port: 7072, Status: NodeStatusActive}
	n3 := &Node{ID: "node3", Type: NodeTypeMiner, Host: "", Port: 7073, Status: NodeStatusActive}

	Self = &SelfNode{}
	Self.Node = n1
	Self.privateKey = "aa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"
	np := NewPool(NodeTypeMiner)
	np.AddNode(n2)
	np.AddNode(n3)

	options := SendOptions{MaxRelayLength: 0, CurrentRelayLength: 0, Compress: true}
	sendHandler := SendEntityHandler("v1/_n2n/entity/post", options)
	np.SendAtleast(2, sendHandler(entity))
}
