package miner

import (
	"0chain.net/chaincore/block"
	mocks "0chain.net/mocks/core/datastore"
	"github.com/stretchr/testify/suite"
	"testing"
)

type BlockNotarizationEntityTestSuite struct {
	suite.Suite
}

func TestBlockNotarizationEntityTestSuiteSuite(t *testing.T) {
	suite.Run(t, &BlockNotarizationEntityTestSuite{})
}

func (s *BlockNotarizationEntityTestSuite) TestNotarizationProvider() {
	NotarizationProvider()
}

func (s *BlockNotarizationEntityTestSuite) TestNotarizationDoReadLock() {
	n := NotarizationProvider().(*Notarization)
	store := &mocks.Store{}

	block.SetupEntity(store)

	b := block.NewBlock("1", 1)
	n.Block = b
	n.DoReadLock()
}

func (s *BlockNotarizationEntityTestSuite) TestNotarizationDoReadUnlock() {
	n := NotarizationProvider().(*Notarization)
	store := &mocks.Store{}

	block.SetupEntity(store)

	b := block.NewBlock("1", 1)
	n.Block = b
	//n.DoReadUnlock()
}

func (s *BlockNotarizationEntityTestSuite) TestNotarizationGetEntityMetadata() {
	n := NotarizationProvider().(*Notarization)
	n.GetEntityMetadata()
}

func (s *BlockNotarizationEntityTestSuite) TestNotarizationGetKey() {
	n := NotarizationProvider().(*Notarization)
	n.GetKey()
}

func (s *BlockNotarizationEntityTestSuite) TestSetupNotarizationEntity() {
	SetupNotarizationEntity()
}
