package miner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/transaction"
)

func TestBlockGeneration(t *testing.T) {
	SetUpSelf()
	common.SetupRootContext(node.GetNodeContext())
	transaction.SetupEntity()
	block.SetupEntity()
	client.SetupEntity()
	ctx := common.GetRootContext()
	ctx = datastore.WithConnection(ctx)
	block.BLOCK_SIZE = 1
	b := block.Provider().(*block.Block)
	b.ChainID = datastore.ToKey(config.GetServerChainID())
	// pb = ... // TODO: Setup a privious block
	// b.SetPreviousBlock(pb)
	b.GenerateBlock(ctx)
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(b)
	fmt.Printf("%v\n", buf)
	common.Done()
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

func BenchmarkChainSetupWorker(b *testing.B) {
	SetUpSelf()
	common.SetupRootContext(node.GetNodeContext())
	block.SetupEntity()
	chain.SetupEntity()
	client.SetupEntity()
	transaction.SetupEntity()
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

	block.BLOCK_SIZE = 1 // Just for testing
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
