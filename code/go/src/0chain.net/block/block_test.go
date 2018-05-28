package block

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/node"
	"0chain.net/transaction"
)

func TestBlockGeneration(t *testing.T) {
	SetUpSelf()
	common.SetupRootContext(node.GetNodeContext())
	transaction.SetupEntity()
	SetupEntity()
	client.SetupEntity()
	ctx := common.GetRootContext()
	ctx = datastore.WithConnection(ctx)
	BLOCK_SIZE = 1
	a := Provider().(*Block)
	a.GenerateBlock(ctx, config.GetServerChainID(), nil)
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(a)
	fmt.Printf("%v\n", buf)
}

func SetUpSelf() {
	var sn node.SelfNode
	var n node.Node
	n.Type = node.NodeTypeMiner
	n.PublicKey = "1c2313e4d2115b88c516b3e27cead994a0902c83411506e7804ad9c1fb276624"
	n.ID = encryption.Hash(n.PublicKey)
	sn.SetPrivateKey("1ad5c839b37be0d87e7eb71c3d6c81197f6a990a34007387defa694b2ed66cbc1c2313e4d2115b88c516b3e27cead994a0902c83411506e7804ad9c1fb276624")
	sn.Node = &n
	node.Self = &sn
}
