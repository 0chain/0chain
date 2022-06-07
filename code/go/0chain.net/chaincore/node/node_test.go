package node

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"0chain.net/chaincore/client"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var Miners *Pool

func init() {
	logging.Logger = zap.NewNop()
	logging.N2n = zap.NewNop()
	Miners = NewPool(NodeTypeMiner)
	createMiners(Miners)

}

func createMiners(np *Pool) {
	sd := Node{Host: "127.0.0.1", Port: 7071, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme1 := encryption.NewED25519Scheme()
	err := sigScheme1.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sd.SetSignatureScheme(sigScheme1)
	np.AddNode(&sd)

	sb := Node{Host: "127.0.0.2", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme2 := encryption.NewED25519Scheme()
	err = sigScheme2.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sb.SetSignatureScheme(sigScheme2)
	np.AddNode(&sb)

	ns := Node{Host: "127.0.0.3", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme3 := encryption.NewED25519Scheme()
	err = sigScheme3.GenerateKeys()
	if err != nil {
		panic(err)
	}
	ns.SetSignatureScheme(sigScheme3)
	np.AddNode(&ns)

	nr := Node{Host: "127.0.0.4", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme4 := encryption.NewED25519Scheme()
	err = sigScheme4.GenerateKeys()
	if err != nil {
		panic(err)
	}
	nr.SetSignatureScheme(sigScheme4)
	np.AddNode(&nr)

	gg := Node{Host: "127.0.0.5", Port: 7070, Type: NodeTypeMiner, Status: NodeStatusActive}
	sigScheme5 := encryption.NewED25519Scheme()
	err = sigScheme5.GenerateKeys()
	if err != nil {
		panic(err)
	}
	gg.SetSignatureScheme(sigScheme5)
	np.AddNode(&gg)
}

func TestNodeSetup(t *testing.T) {
	Miners.Print(bytes.NewBuffer(nil))
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
	entity.SetSignatureScheme(sigScheme)

	n1 := Provider()
	n1.Type = NodeTypeMiner
	n1.Port = 7071
	n1.Status = NodeStatusActive
	s1 := encryption.NewED25519Scheme()
	s1.GenerateKeys()
	n1.SetSignatureScheme(s1)

	n2 := Provider()
	n2.Type = NodeTypeMiner
	n2.Port = 7072
	n2.Status = NodeStatusActive
	s2 := encryption.NewED25519Scheme()
	s2.GenerateKeys()
	n2.SetSignatureScheme(s2)

	n3 := Provider()
	n3.Type = NodeTypeMiner
	n3.Port = 7073
	n3.Status = NodeStatusActive
	s3 := encryption.NewED25519Scheme()
	s3.GenerateKeys()
	n3.SetSignatureScheme(s3)
	//n3.ID = "103c274502661e78a2b5c470057e57699e372a4382a4b96b29c1bec993b1d19c"

	Self = &SelfNode{}
	Self.Node = n1
	Self.SetSignatureScheme(s1)
	// Self.privateKey = "aa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"
	np := NewPool(NodeTypeMiner)
	np.AddNode(n1)
	np.AddNode(n2)
	np.AddNode(n3)

	options := SendOptions{MaxRelayLength: 0, CurrentRelayLength: 0, Compress: true, CODEC: datastore.CodecMsgpack}
	sendHandler := SendEntityHandler("/v1/_n2n/entity", &options)
	_ = np.SendAtleast(context.Background(), 2, sendHandler(entity))
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
		nd.SetSignatureScheme(sigScheme)
		sharders.AddNode(&nd)
	}
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
	_ = ps.ScoreHashString(np, hash)
}

func TestNodeTypeNames(t *testing.T) {
	require.Equal(t, 3, len(NodeTypeNames))
	require.Equal(t, "Miner", NodeTypeNames[NodeTypeMiner].Value)
	require.Equal(t, "Sharder", NodeTypeNames[NodeTypeSharder].Value)
	require.Equal(t, "Blobber", NodeTypeNames[NodeTypeBlobber].Value)
}

func TestGetNodeTypeNames(t *testing.T) {
	n := Provider()
	n.Type = NodeTypeMiner
	n.SetIndex = 2
	require.Equal(t, "Miner002", n.GetPseudoName())
	n.Type = NodeTypeSharder
	require.Equal(t, "Sharder002", n.GetPseudoName())
	n.Type = NodeTypeBlobber
	require.Equal(t, "Blobber002", n.GetPseudoName())
	n.Type = 100
	require.Equal(t, "Unknown002", n.GetPseudoName())
}
