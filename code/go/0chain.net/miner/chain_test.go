package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	round_mocks "0chain.net/mocks/chaincore/round"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"net/http"
	"testing"
)

type ChainTestSuite struct {
	suite.Suite
}

func TestChainTestSuite(t *testing.T) {
	logging.Logger = zap.NewNop()

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

	round.SetupEntity(&mocks.Store{})

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
	logging.Logger = zap.NewNop()

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

	n := node.Provider()

	s1 := 5
	s2 := 8

	mc.RequestStartChain(n, &s1, &s2)
}

func (s *ChainTestSuite) TestSaveClients() {

	mStore := &mocks.Store{}
	mEmd := &mocks.EntityMetadata{}

	mStore.On("MultiRead",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mStore.On("Get",
		mock.Anything).Maybe().Return(round.NewRoundStartingStorage())

	mEmd.On("GetDB").Maybe().Return("client")
	mEmd.On("GetStore").Return(mStore)

	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()

	common.SetupRootContext(context.Background())
	client.SetupEntity(mStore)

	conn := redigomock.NewConn()
	pool := &redis.Pool{
		Dial:    func() (redis.Conn, error) { return conn, nil },
		MaxIdle: 10,
	}

	memorystore.AddPool("client", pool)

	datastore.RegisterEntityMetadata("client", mEmd)

	err := mc.SaveClients(context.Background(), []*client.Client{})
	s.Require().NoError(err)

	mEmd.AssertExpectations(s.T())
	mStore.AssertExpectations(s.T())
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

	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(0)

	mRound := &round_mocks.RoundStorage{}
	mRound.On("Get",
		mock.Anything).Maybe().Return(mb)
	mRound.On("Put",
		mock.Anything, mock.Anything).
		Maybe().Return(nil)

	mEmd := &mocks.EntityMetadata{}
	mEmd.On("Instance").
		Maybe().Return(round.NewRound(1))

	mBSmd := &mocks.EntityMetadata{}
	mBSmd.On("Instance").
		Maybe().Return(block.BlockSummaryProvider())

	datastore.RegisterEntityMetadata("round", mEmd)
	datastore.RegisterEntityMetadata("block_summary", mBSmd)

	b := block.NewBlock("1", 1)

	c := chain.NewChainFromConfig()
	c.LatestFinalizedBlock = b
	c.LatestFinalizedMagicBlock = b
	SetupMinerChain(c)

	mc := GetMinerChain()
	mc.MagicBlockStorage = mRound
	mc.SetMagicBlock(mb)

	mc.SetLatestFinalizedBlock(context.Background(), b)

	mRound.AssertExpectations(s.T())
	mEmd.AssertExpectations(s.T())
}

func (s *ChainTestSuite) TestSetPreviousBlock() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()
	b := block.NewBlock("1", 1)
	b1 := block.NewBlock("1", 2)
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

	ms := &mocks.Store{}

	ms.On("Get", mock.Anything).Maybe().Return(
		round.NewRoundStartingStorage())

	chain.SetupEntity(ms)

	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(0)
	mb.Sharders = node.NewPool(1)
	mb.StartingRound = 1

	mRound := &round_mocks.RoundStorage{}
	mRound.On("Get",
		mock.Anything).Maybe().Return(mb)

	c.MagicBlockStorage = mRound

	mc := GetMinerChain()
	b := block.NewMagicBlock()

	_, err := mc.SetupGenesisBlock(
		"24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509", b,
		&state.InitStates{
			States: []state.InitState{
				{ID: "", Tokens: state.Balance(10)},
			},
		})
	s.Require().NoError(err)

	ms.AssertExpectations(s.T())
}

func (s *ChainTestSuite) TestViewChange() {
	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	mc := GetMinerChain()
	b := block.NewBlock("24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509", 5)

	mc.ViewChange(context.Background(), b)
}

func (s *ChainTestSuite) TestdeleteTxns() {
	chain.SetupEntity(&mocks.Store{})

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
