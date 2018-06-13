package miner

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"testing"

	"0chain.net/block"
	"0chain.net/blockstore"
	"0chain.net/chain"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/transaction"
)

var numOfTransactions int

func init() {
	flag.IntVar(&numOfTransactions, "num_txns", 4000, "number of transactions per block")
}

/*
func getContext() context.Context {
	ctx := common.GetRootContext()
	ctx = memorystore.WithConnection(ctx)
	return ctx
}

func generateSingleBlock(ctx context.Context, prevBlock *block.Block, r *round.Round) (*block.Block, error) {
	b := block.Provider().(*block.Block)
	r := round.Provider().(*round.Round)
	mc := GetMinerChain()
	if prevBlock == nil {
		fmt.Println("...")
		prevBlock = block.Provider().(*block.Block)
		prevBlock.Hash = chain.GenesisBlockHash
		prevBlock.ChainID = datastore.ToKey(config.GetServerChainID())
		r.Block = prevBlock
		r.AddBlock(prevBlock)
		mc.AddRound(r)
	}
	b.ChainID = prevBlock.ChainID
	// pb = ... // TODO: Setup a privious block
	// b.SetPreviousBlock(pb)
	mc.BlockSize = int32(numOfTransactions)
	r = round.Provider().(*round.Round)
	r.Number = roundNum
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	block.SetupFileBlockStore(fmt.Sprintf("%v%s.0chain.net", usr.HomeDir, string(os.PathSeparator)))
	b, err = mc.GenerateRoundBlock(ctx, r)
	if err != nil {
		return nil, err
	}
	b.ComputeProperties()
	r.AddBlock(b)
	mc.AddRound(r)
	block.Store.Write(b)
	return b, nil
}

func TestBlockVerification(t *testing.T) {
	SetUpSingleSelf()
	ctx := getContext()
	b, err := generateSingleBlock(ctx, nil, 1)
	mc := GetMinerChain()
	if b != nil {
		_, err = mc.VerifyRoundBlock(ctx, b)
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
	ctx := getContext()
	b0, err := generateSingleBlock(ctx, nil, 1)
	mc := GetMinerChain()
	if b0 != nil {
		var b1 *block.Block
		b1, err = generateSingleBlock(ctx, b0, 2)
		_, err = mc.VerifyRoundBlock(ctx, b1)
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
	ctx := getContext()
	b0, err := generateSingleBlock(ctx, nil, 1)
	//mc := GetMinerChain()
	if b0 != nil {
		//var b1 *block.Block
		_, err = generateSingleBlock(ctx, b0, 3)
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
	ctx := getContext()
	b, err := generateSingleBlock(ctx, nil, 1)
	mc := GetMinerChain()
	if b != nil {
		b.Hash = "bad hash"
		_, err = mc.VerifyRoundBlock(ctx, b)
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
	ctx := getContext()
	b, err := generateSingleBlock(ctx, nil, 1)
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
		_, err = mc.VerifyRoundBlock(ctx, b)
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
	ctx := getContext()

	block, _ := generateSingleBlock(ctx, nil, 1)
	if block != nil {
		b.Logf("Created block with %v transactions", len(block.Txns))
	} else {
		b.Error("Failed to even generate a block... OUCH!")
	}
}

func BenchmarkGenerateAndVerifyALotTransactions(b *testing.B) {
	SetUpSingleSelf()
	ctx := getContext()
	block, err := generateSingleBlock(ctx, nil, 1)
	mc := GetMinerChain()
	if block != nil && err == nil {
		_, err = mc.VerifyRoundBlock(ctx, block)
		if err != nil {
			b.Errorf("Block with %v transactions failed verication", len(block.Txns))
		} else {
			b.Logf("Created block with %v transactions", len(block.Txns))
		}
	} else {
		b.Error("Failed to even generate a block... OUCH!")
	}
} */

func SetUpSingleSelf() {
	n1 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7071, Status: node.NodeStatusActive}
	n1.ID = "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509"
	n1.PublicKey = "e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"
	node.Self = &node.SelfNode{}
	node.Self.Node = n1
	node.Self.SetPrivateKey("aa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0")
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

func SetUpSelf() {
	n1 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7071, Status: node.NodeStatusActive}
	n1.ID = "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509"
	n2 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7072, Status: node.NodeStatusActive}
	n2.ID = "5fbb6924c222e96df6c491dfc4a542e1bbfc75d821bcca992544899d62121b55"
	n3 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7073, Status: node.NodeStatusActive}
	n3.ID = "103c274502661e78a2b5c470057e57699e372a4382a4b96b29c1bec993b1d19c"

	node.Self = &node.SelfNode{}
	node.Self.Node = n1
	node.Self.SetPrivateKey("aa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0")
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
	SetUpSelf()
	ctx := common.GetRootContext()
	ctx = memorystore.WithConnection(ctx)
	b := block.Provider().(*block.Block)
	b.ChainID = datastore.ToKey(config.GetServerChainID())
	// pb = ... // TODO: Setup a privious block
	// b.SetPreviousBlock(pb)
	gb := block.Provider().(*block.Block)
	gb.Hash = chain.GenesisBlockHash
	mc := GetMinerChain()
	mc.BlockSize = int32(numOfTransactions)
	r := round.Provider().(*round.Round)
	r.Block = gb
	mr := mc.CreateRound(r)
	mc.AddRound(mr)
	r = round.Provider().(*round.Round)
	r.Number = 1
	mr = mc.CreateRound(r)
	mc.AddRound(mr)

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	blockstore.SetupFSBlockStore(fmt.Sprintf("%v%s.0chain.net", usr.HomeDir, string(os.PathSeparator)))

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
			b2, err := blockstore.Store.Read(b.Hash, b.Round)
			if err != nil {
				t.Errorf("Error reading the block: %v\n", err)
			} else {
				t.Logf("Block hash is: %v\n", b2.Hash)
			}
		}
	}
	common.Done()
}
