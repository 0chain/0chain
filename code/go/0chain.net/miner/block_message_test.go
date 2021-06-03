package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"github.com/stretchr/testify/suite"
	"testing"
)

type BlockMessageTestSuite struct {
	suite.Suite
}

func TestBlockMessageTestSuite(t *testing.T) {
	suite.Run(t, &BlockMessageTestSuite{})
}

func (s *BlockMessageTestSuite) TestBlockMessageRetry() {

	n := node.Provider()
	r := &Round{}
	b := block.Provider().(*block.Block)
	c := make(chan *BlockMessage, 128)

	bc := NewBlockMessage(5, n, r, b)

	bc.Retry(c)
}

func (s *BlockMessageTestSuite) TestBlockMessageShouldRetry() {
	n := node.Provider()
	r := &Round{}
	b := block.Provider().(*block.Block)

	bc := NewBlockMessage(5, n, r, b)

	bc.ShouldRetry()
}

func (s *BlockMessageTestSuite) TestGetMessageLookup() {
	GetMessageLookup(5)
}
