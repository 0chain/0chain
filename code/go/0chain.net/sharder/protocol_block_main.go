//go:build !integration_tests
// +build !integration_tests

package sharder

import "0chain.net/chaincore/block"

func notifyConductor(block *block.Block) error {
	return nil
}