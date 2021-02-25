package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/datastore"
	"context"
	"github.com/stretchr/testify/suite"
	"net/http"
	"testing"
)

type ChainTestSuite struct {
	suite.Suite
}

func TestChainTestSuite(t *testing.T) {

	//round.SetupEntity()

	suite.Run(t, &ChainTestSuite{})
}

func (s *ChainTestSuite) TestChainStarted() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.ChainStarted(context.Background())

}

func (s *ChainTestSuite) TestCreateRound() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()
	r := round.NewRound(5)

	mc.CreateRound(r)
}

func (s *ChainTestSuite) TestGetBlockMessageChannel() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.GetBlockMessageChannel()
}

func (s *ChainTestSuite) TestGetDKG() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.GetDKG(5)
}

func (s *ChainTestSuite) TestGetMinerRound() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.GetMinerRound(5)
}

func (s *ChainTestSuite) TestRequestStartChain() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()
	n, _ := node.NewNode(map[interface{}]interface{}{})
	s1 := 5
	s2 := 8

	mc.RequestStartChain(n, &s1, &s2)
}

func (s *ChainTestSuite) TestSaveClients() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.SaveClients(context.Background(), []*client.Client{})
}

func (s *ChainTestSuite) TestSaveMagicBlock() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.SaveMagicBlock()
}

func (s *ChainTestSuite) TestSetDKG() {

	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()
	dkg := &bls.DKG{}

	mc.SetDKG(dkg, 5)
}

func (s *ChainTestSuite) TestSetDiscoverClients() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.SetDiscoverClients(true)
}

func (s *ChainTestSuite) TestSetLatestFinalizedBlock() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()
	b := block.Provider().(*block.Block)

	mc.SetLatestFinalizedBlock(context.Background(), b)
}

func (s *ChainTestSuite) TestSetPreviousBlock() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()
	b := block.Provider().(*block.Block)
	b1 := block.Provider().(*block.Block)
	r := round.NewRound(5)

	mc.SetPreviousBlock(r, b, b1)
}

func (s *ChainTestSuite) TestSetStarted() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.SetStarted()
}

func (s *ChainTestSuite) TestSetupGenesisBlock() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()
	b := block.NewMagicBlock()

	mc.SetupGenesisBlock("24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509", b)
}

func (s *ChainTestSuite) TestViewChange() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()
	b := block.NewBlock("24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509", 5)

	mc.ViewChange(context.Background(), b)
}

func (s *ChainTestSuite) TestdeleteTxns() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.deleteTxns([]datastore.Entity{})
}

func (s *ChainTestSuite) TestisStarted() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.isStarted()
}

func (s *ChainTestSuite) TestsendRestartRoundEvent() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	// @todo
	go mc.sendRestartRoundEvent(context.Background())
}

func (s *ChainTestSuite) TeststartPulling() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.startPulling()
}

func (s *ChainTestSuite) TeststopPulling() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.stopPulling()
}

func (s *ChainTestSuite) TestsubRestartRoundEvent() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	// @todo
	go mc.subRestartRoundEvent()
}

func (s *ChainTestSuite) TestunsubRestartRoundEvent() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	ch := make(chan struct{})

	// @todo
	go mc.unsubRestartRoundEvent(ch)
}

func (s *ChainTestSuite) TestGetMinerChain() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	GetMinerChain()
}

func (s *ChainTestSuite) TestMinerRoundFactory_CreateRoundF() {
	mrf := MinerRoundFactory{}
	mrf.CreateRoundF(6)
}

func (s *ChainTestSuite) TestSetupMinerChain() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)
}

func (s *ChainTestSuite) TestSetupStartChainEntity() {
	SetupStartChainEntity()
}

func (s *ChainTestSuite) TestStartChainProvider() {
	StartChainProvider()
}

func (s *ChainTestSuite) TestStartChainRequestHandler() {
	StartChainRequestHandler(context.Background(), &http.Request{})
}

func (s *ChainTestSuite) TestStartChain_GetEntityMetadata() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	mc.GetEntityMetadata()
}

func (s *ChainTestSuite) Test_mbRoundOffset() {
	mbRoundOffset(6)
}
