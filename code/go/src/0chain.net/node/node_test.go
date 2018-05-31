package node

import (
	"context"
	"fmt"
	"os"
	"testing"

	"0chain.net/client"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/memorystore"
)

var Miners = NewPool(NodeTypeMiner)

func TestNodeSetup(t *testing.T) {
	sd := Node{Host: "127.0.0.1", Port: 7071, Type: NodeTypeMiner, Status: NodeStatusActive}
	publicKey, _ := encryption.GenerateKeys()
	sd.ID = encryption.Hash(publicKey)
	sd.PublicKey = publicKey
	Miners.AddNode(&sd)

	sb := Node{Host: "127.0.0.2", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	publicKey, _ = encryption.GenerateKeys()
	sb.ID = encryption.Hash(publicKey)
	sb.PublicKey = publicKey
	Miners.AddNode(&sb)

	ns := Node{Host: "127.0.0.3", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	publicKey, _ = encryption.GenerateKeys()
	ns.ID = encryption.Hash(publicKey)
	ns.PublicKey = publicKey
	Miners.AddNode(&ns)

	nr := Node{Host: "127.0.0.4", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	publicKey, _ = encryption.GenerateKeys()
	nr.ID = encryption.Hash(publicKey)
	nr.PublicKey = publicKey
	Miners.AddNode(&nr)

	gg := Node{Host: "127.0.0.5", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	publicKey, _ = encryption.GenerateKeys()
	gg.ID = encryption.Hash(publicKey)
	gg.PublicKey = publicKey
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

var companyEntityMetadata = &datastore.EntityMetadataImpl{Name: "company", MemoryDB: "company"}

func (c *Company) GetEntityMetadata() datastore.EntityMetadata {
	return companyEntityMetadata
}

func (c *Company) GetEntityName() string {
	return "company"
}

func (c *Company) Validate(ctx context.Context) error {
	return nil
}

func (c *Company) Read(ctx context.Context, id datastore.Key) error {
	return memorystore.Read(ctx, id, c)
}

func (c *Company) Write(ctx context.Context) error {
	return memorystore.Write(ctx, c)
}

func (c *Company) Delete(ctx context.Context) error {
	return memorystore.Delete(ctx, c)
}

// TODO: Assuming node2 & 3 are running - figure out a way to make this self-contained without the dependency
func TestNode2NodeCommunication(t *testing.T) {
	publicKey, _ := encryption.GenerateKeys()
	entity := client.Provider().(*client.Client)
	entity.ID = datastore.ToKey(encryption.Hash(publicKey))
	entity.PublicKey = publicKey

	n1 := &Node{Type: NodeTypeMiner, Host: "", Port: 7071, Status: NodeStatusActive}
	n1.ID = "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509"
	n2 := &Node{Type: NodeTypeMiner, Host: "", Port: 7072, Status: NodeStatusActive}
	n2.ID = "5fbb6924c222e96df6c491dfc4a542e1bbfc75d821bcca992544899d62121b55"
	n3 := &Node{Type: NodeTypeMiner, Host: "", Port: 7073, Status: NodeStatusActive}
	n3.ID = "103c274502661e78a2b5c470057e57699e372a4382a4b96b29c1bec993b1d19c"

	Self = &SelfNode{}
	Self.Node = n1
	Self.privateKey = "aa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"
	np := NewPool(NodeTypeMiner)
	np.AddNode(n2)
	np.AddNode(n3)

	options := SendOptions{MaxRelayLength: 0, CurrentRelayLength: 0, Compress: true, CODEC: datastore.CodecMsgpack}
	sendHandler := SendEntityHandler("/v1/_n2n/entity/post", &options)
	sentTo := np.SendAtleast(2, sendHandler(entity))
	fmt.Printf("sent to %v nodes\n", len(sentTo))
}
