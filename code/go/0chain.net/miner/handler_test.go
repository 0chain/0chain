package miner

import (
	"0chain.net/chaincore/chain"
	"0chain.net/core/common"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"testing"
)

type HandlerSuite struct {
	suite.Suite
}

func TestHandlerSuiteSuite(t *testing.T) {
	suite.Run(t, &HandlerSuite{})
}

func (s *HandlerSuite) TestChainStatsHandler() {
	req := httptest.NewRequest(http.MethodGet, "/_chain_stats", nil)

	chain.SetupEntity(&mocks.Store{})

	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	_, err := ChainStatsHandler(context.Background(), req)

	s.Require().NoError(err)
}

func (s *HandlerSuite) TestChainStatsWriter() {

	req := httptest.NewRequest(http.MethodGet, "/_chain_stats", nil)
	res := httptest.NewRecorder()

	ChainStatsWriter(res, req)

	s.Require().Equal(http.StatusOK, res.Code)
}

func (s *HandlerSuite) TestGetWalletStats() {
	req := httptest.NewRequest(http.MethodGet, "/_diagnostics/wallet_stats", nil)
	res := httptest.NewRecorder()

	GetWalletStats(res, req)

	s.Require().Equal(http.StatusOK, res.Code)
}

func (s *HandlerSuite) TestGetWalletTable() {
	GetWalletTable(true)
}

func (s *HandlerSuite) TestMinerStatsHandler() {

	chain.SetupEntity(&mocks.Store{})

	c := chain.NewChainFromConfig()
	SetupMinerChain(c)

	req := httptest.NewRequest(http.MethodGet, "/v1/miner/get/stats", nil)

	MinerStatsHandler(context.Background(), req)
}

func TestSetupHandlers(t *testing.T) {
	common.ConfigRateLimits()
	SetupHandlers()
}
