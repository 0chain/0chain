package miner

import (
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
	n.DoReadLock()
}

func (s *BlockNotarizationEntityTestSuite) TestNotarizationDoReadUnlock() {
	n := NotarizationProvider().(*Notarization)
	n.DoReadUnlock()
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
