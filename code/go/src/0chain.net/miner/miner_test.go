package miner

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"testing"
	"time"

	"0chain.net/block"
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

func TestBlockGeneration(t *testing.T) {
	SetUpSelf()
	ctx := common.GetRootContext()
	ctx = memorystore.WithConnection(ctx)
	b := block.Provider().(*block.Block)
	b.ChainID = datastore.ToKey(config.GetServerChainID())
	// pb = ... // TODO: Setup a privious block
	// b.SetPreviousBlock(pb)
	gb := block.Provider().(*block.Block)
	gb.Hash = block.GenesisBlockHash
	mc := GetMinerChain()
	mc.BlockSize = 10000
	r := round.Provider().(*round.Round)
	r.Block = gb
	mc.AddRound(r)
	r = round.Provider().(*round.Round)
	r.Number = 1
	mc.AddRound(r)

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	block.SetupFileBlockStore(fmt.Sprintf("%v%s.0chain.net", usr.HomeDir, string(os.PathSeparator)))

	b, err = mc.GenerateRoundBlock(ctx, 1)

	if err != nil {
		fmt.Printf("Error generating block: %v\n", err)
	} else {
		/* fmt.Printf("%v\n", datastore.ToJSON(b))
		fmt.Printf("%v\n", datastore.ToMsgpack(b))
		*/
		fmt.Printf("json length: %v\n", datastore.ToJSON(b).Len())
		fmt.Printf("msgpack length: %v\n", datastore.ToMsgpack(b).Len())
		err = block.Store.Write(b)
		if err != nil {
			fmt.Printf("Error writing the block: %v\n", err)
		} else {
			b2, err := block.Store.Read(b.Hash, b.Round)
			if err != nil {
				fmt.Printf("Error reading the block: %v\n", err)
			} else {
				fmt.Printf("Block hash is: %v\n", b2.Hash)
			}
		}
		b.ComputeProperties()
		valid, err := mc.VerifyBlock(ctx, b)
		if err != nil {
			fmt.Printf("Error verifying block: %v\n", err)
		} else {
			if !valid {
				fmt.Printf("hash verification is working\n")
			} else {
				fmt.Printf("hash verification problem\n")
			}
		}
	}
	common.Done()
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
	np.AddNode(n2)
	np.AddNode(n3)
	common.SetupRootContext(node.GetNodeContext())
	config.SetServerChainID(config.GetMainChainID())
	common.SetupRootContext(node.GetNodeContext())
	transaction.SetupEntity()
	block.SetupEntity()
	client.SetupEntity()
	chain.SetupEntity()

	c := chain.Provider().(*chain.Chain)
	c.ID = datastore.ToKey(config.GetServerChainID())
	c.Miners = np
	chain.SetServerChain(c)
	SetupMinerChain(c)
	mc := GetMinerChain()
	mc.Miners = np
	SetupM2MSenders()
}

func BenchmarkChainSetupWorker(b *testing.B) {
	common.SetupRootContext(node.GetNodeContext())

	//bookstrapping with a genesis block & main chain as the one being mined
	gb := block.Provider().(*block.Block)
	gb.Hash = block.GenesisBlockHash
	gb.Round = 0
	c := chain.Provider().(*chain.Chain)
	c.ID = datastore.ToKey(config.GetServerChainID())
	chain.SetServerChain(c)
	SetupMinerChain(c)
	gb.ChainID = c.GetKey()
	c.LatestFinalizedBlock = gb
	c.SetupWorkers(common.GetRootContext())
	mc := GetMinerChain()
	mc.BlockSize = 10000
	timer := time.NewTimer(5 * time.Second)
	startTime := time.Now()
	go RoundLogic(common.GetRootContext(), GetMinerChain())
	ts := <-timer.C
	fmt.Printf("reached timeout: %v %v\n", time.Since(startTime), ts)
	common.Done()
}

func RoundLogic(ctx context.Context, c *Chain) {
	ticker := time.NewTicker(100 * time.Millisecond)
	r := &round.Round{}
	r.Number = 0
	r.Role = round.RoleVerifier
	roundsChannel := c.GetRoundsChannel()
	for true {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			fmt.Printf("round: %v\n", r)
			if r.Block != nil && r.Block.Txns != nil {
				for idx, txn := range *r.Block.Txns {
					fmt.Printf("txn(%v): %v\n", idx, txn)
				}
			}
			r.Number++
			b := block.Provider().(*block.Block)
			b.ChainID = datastore.ToKey(config.GetServerChainID())
			r.Block = b
			if r.Role == round.RoleVerifier {
				r.Role = round.RoleGenerator
			} else {
				r.Role = round.RoleVerifier
				txns := make([]*transaction.Transaction, 0)
				b.Txns = &txns
			}
			roundsChannel <- r
		}
	}
}
