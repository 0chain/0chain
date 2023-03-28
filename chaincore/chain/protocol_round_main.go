//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"0chain.net/chaincore/round"
)

func (c *Chain) FinalizeRound(r round.RoundI) {
	c.FinalizeRoundImpl(r)
}
