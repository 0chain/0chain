package miner

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/user"
	"testing"

	"0chain.net/core/encryption"
	"0chain.net/sharder/blockstore"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
)

var numOfTransactions int

func init() {
	flag.IntVar(&numOfTransactions, "num_txns", 4000, "number of transactions per block")
}

func getContext() (context.Context, func()) {
	ctx := common.GetRootContext()
	ctx = memorystore.WithConnection(ctx)
	return ctx, func() {
		memorystore.Close(ctx)
	}
}

func generateSingleBlock(ctx context.Context, prevBlock *block.Block, r *Round) (*block.Block, error) {
	b := block.Provider().(*block.Block)
	mc := GetMinerChain()
	if prevBlock == nil {
		gb := SetupGenesisBlock()
		prevBlock = gb
	}
	b.ChainID = prevBlock.ChainID
	mc.BlockSize = int32(numOfTransactions)
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	blockstore.SetupStore(blockstore.NewFSBlockStore(fmt.Sprintf("%v%s.0chain.net", usr.HomeDir, string(os.PathSeparator))))
	b, err = mc.GenerateRoundBlock(ctx, r)
	if err != nil {
		return nil, err
	}
	b.ComputeProperties()
	blockstore.GetStore().Write(b)
	return b, nil
}

func CreateRound(number int64) *Round {
	mc := GetMinerChain()
	r := round.NewRound(number)
	mr := mc.CreateRound(r)
	mc.AddRound(mr)
	return mr
}
func TestBlockVerification(t *testing.T) {
	SetUpSingleSelf()
	mc := GetMinerChain()
	ctx, clean := getContext()
	defer clean()

	mr := CreateRound(1)
	b, err := generateSingleBlock(ctx, nil, mr)
	if b != nil {
		_, err = mc.VerifyRoundBlock(ctx, mr, b)
	}
	if err != nil {
		t.Errorf("Block failed verification because %v", err.Error())
	} else {
		t.Log("Block passed verification")
	}
	common.Done()
}

func TestTwoCorrectBlocks(t *testing.T) {
	SetUpSingleSelf()
	ctx, clean := getContext()
	defer clean()
	mr := CreateRound(1)
	b0, err := generateSingleBlock(ctx, nil, mr)
	mc := GetMinerChain()
	if b0 != nil {
		var b1 *block.Block
		mr2 := CreateRound(2)
		b1, err = generateSingleBlock(ctx, b0, mr2)
		_, err = mc.VerifyRoundBlock(ctx, mr2, b1)
	}
	if err != nil {
		t.Errorf("Block failed verification because %v", err.Error())
	} else {
		t.Log("Block passed verification")
	}
	common.Done()
}
func TestTwoBlocksWrongRound(t *testing.T) {
	SetUpSingleSelf()
	ctx, clean := getContext()
	defer clean()
	mr := CreateRound(1)
	b0, err := generateSingleBlock(ctx, nil, mr)
	//mc := GetMinerChain()
	if b0 != nil {
		//var b1 *block.Block
		mr3 := CreateRound(3)
		_, err = generateSingleBlock(ctx, b0, mr3)
		//_, err = mc.VerifyRoundBlock(ctx, b1)
	}
	if err != nil {
		t.Log("Second block failed to generate")
	} else {
		t.Error("Second block generated")
	}
	common.Done()
}

func TestBlockVerificationBadHash(t *testing.T) {
	SetUpSingleSelf()
	ctx, clean := getContext()
	defer clean()
	mr := CreateRound(1)
	b, err := generateSingleBlock(ctx, nil, mr)
	mc := GetMinerChain()
	if b != nil {
		b.Hash = "bad hash"
		_, err = mc.VerifyRoundBlock(ctx, mr, b)
	}
	if err == nil {
		t.Error("FAIL: Block with bad hash passed verification")
	} else {
		t.Log("SUCCESS: Block with bad hash failed verifcation")
	}
	common.Done()
}

func TestBlockVerificationTooFewTransactions(t *testing.T) {
	SetUpSingleSelf()
	ctx, clean := getContext()
	defer clean()
	mr := CreateRound(1)
	b, err := generateSingleBlock(ctx, nil, mr)
	if err != nil {
		t.Errorf("Error generating block: %v", err)
		return
	}
	mc := GetMinerChain()
	txnLength := numOfTransactions - 1
	b.Txns = make([]*transaction.Transaction, txnLength)
	if b != nil {
		for idx, txn := range b.Txns {
			if idx < txnLength {
				b.Txns[idx] = txn
			}
		}
		_, err = mc.VerifyRoundBlock(ctx, mr, b)
	}
	if err == nil {
		t.Error("FAIL: Block with too few transactions passed verification")
	} else {
		t.Log("SUCCESS: Block with too few transactions failed verifcation")
	}
	common.Done()
}

func BenchmarkGenerateALotTransactions(b *testing.B) {
	SetUpSingleSelf()
	ctx, clean := getContext()
	defer clean()

	mr := CreateRound(1)
	block, _ := generateSingleBlock(ctx, nil, mr)
	if block != nil {
		b.Logf("Created block with %v transactions", len(block.Txns))
	} else {
		b.Error("Failed to even generate a block... OUCH!")
	}
}

func BenchmarkGenerateAndVerifyALotTransactions(b *testing.B) {
	SetUpSingleSelf()
	ctx, clean := getContext()
	defer clean()
	mr := CreateRound(1)
	block, err := generateSingleBlock(ctx, nil, mr)
	mc := GetMinerChain()
	if block != nil && err == nil {
		_, err = mc.VerifyRoundBlock(ctx, mr, block)
		if err != nil {
			b.Errorf("Block with %v transactions failed verication", len(block.Txns))
		} else {
			b.Logf("Created block with %v transactions", len(block.Txns))
		}
	} else {
		b.Error("Failed to even generate a block... OUCH!")
	}
}

func SetUpSingleSelf() {
	n1 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7071, Status: node.NodeStatusActive}
	n1.ID = "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509"
	node.Self = &node.SelfNode{}
	node.Self.Node = n1
	np := node.NewPool(node.NodeTypeMiner)
	np.AddNode(n1)
	config.SetServerChainID(config.GetMainChainID())
	common.SetupRootContext(node.GetNodeContext())
	transaction.SetupEntity(memorystore.GetStorageProvider())
	block.SetupEntity(memorystore.GetStorageProvider())
	client.SetupEntity(memorystore.GetStorageProvider())
	chain.SetupEntity(memorystore.GetStorageProvider())
	round.SetupEntity(memorystore.GetStorageProvider())

	c := chain.Provider().(*chain.Chain)
	c.ID = datastore.ToKey(config.GetServerChainID())
	c.Miners = np
	chain.SetServerChain(c)
	SetupMinerChain(c)
	mc := GetMinerChain()
	mc.Miners = np
	SetupM2MSenders()
}

func setupSelfNodeKeys() {
	keys := "e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0\naa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"
	breader := bytes.NewBuffer([]byte(keys))
	sigScheme := encryption.NewED25519Scheme()
	sigScheme.ReadKeys(breader)
	node.Self.SetSignatureScheme(sigScheme)
}

func SetupGenesisBlock() *block.Block {
	mc := GetMinerChain()
	mc.BlockSize = int32(numOfTransactions)
	gr, gb := mc.GenerateGenesisBlock("ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae4")
	mr := mc.CreateRound(gr.(*round.Round))
	mc.AddRoundBlock(gr, gb)
	mc.AddRound(mr)
	return gb
}

func setupSelf() {
	n1 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7071, Status: node.NodeStatusActive}
	n1.ID = "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509"
	n2 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7072, Status: node.NodeStatusActive}
	n2.ID = "5fbb6924c222e96df6c491dfc4a542e1bbfc75d821bcca992544899d62121b55"
	n3 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7073, Status: node.NodeStatusActive}
	n3.ID = "103c274502661e78a2b5c470057e57699e372a4382a4b96b29c1bec993b1d19c"

	node.Self = &node.SelfNode{}
	node.Self.Node = n1

	setupSelfNodeKeys()

	np := node.NewPool(node.NodeTypeMiner)
	np.AddNode(n1)
	np.AddNode(n2)
	np.AddNode(n3)
	common.SetupRootContext(node.GetNodeContext())
	config.SetServerChainID(config.GetMainChainID())
	common.SetupRootContext(node.GetNodeContext())
	transaction.SetupEntity(memorystore.GetStorageProvider())
	block.SetupEntity(memorystore.GetStorageProvider())
	client.SetupEntity(memorystore.GetStorageProvider())
	chain.SetupEntity(memorystore.GetStorageProvider())
	round.SetupEntity(memorystore.GetStorageProvider())

	c := chain.Provider().(*chain.Chain)
	c.ID = datastore.ToKey(config.GetServerChainID())
	c.Miners = np
	chain.SetServerChain(c)
	SetupMinerChain(c)
	mc := GetMinerChain()
	mc.Miners = np
	SetupM2MSenders()
}

func TestBlockGeneration(t *testing.T) {
	setupSelf()
	ctx := common.GetRootContext()
	ctx = memorystore.WithConnection(ctx)
	defer memorystore.Close(ctx)
	SetupGenesisBlock()
	r := round.Provider().(*round.Round)
	r.Number = 1
	mc := GetMinerChain()
	mr := mc.CreateRound(r)
	mc.AddRound(mr)
	b := block.Provider().(*block.Block)
	b.ChainID = datastore.ToKey(config.GetServerChainID())
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	blockstore.SetupStore(blockstore.NewFSBlockStore(fmt.Sprintf("%v%s.0chain.net", usr.HomeDir, string(os.PathSeparator))))

	b, err = mc.GenerateRoundBlock(ctx, mr)

	if err != nil {
		t.Errorf("Error generating block: %v\n", err)
	} else {
		/* fmt.Printf("%v\n", datastore.ToJSON(b))
		fmt.Printf("%v\n", datastore.ToMsgpack(b))
		*/
		t.Logf("json length: %v\n", datastore.ToJSON(b).Len())
		t.Logf("msgpack length: %v\n", datastore.ToMsgpack(b).Len())
		err = blockstore.Store.Write(b)
		if err != nil {
			t.Errorf("Error writing the block: %v\n", err)
		} else {
			b2, err := blockstore.Store.Read(b.Hash)
			if err != nil {
				t.Errorf("Error reading the block: %v\n", err)
			} else {
				t.Logf("Block hash is: %v\n", b2.Hash)
			}
		}
	}
	common.Done()
}
