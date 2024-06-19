//go:build integration_tests
// +build integration_tests

package chain

import (
	"context"
	"log"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/config"
	"0chain.net/core/common"
	"github.com/0chain/common/core/logging"
)

var myFailingRound int64 // once set, we ignore all restarts for that round

func (c *Chain) IsRoundGenerator(r round.RoundI, nd *node.Node) bool {

	var (
		rank          = r.GetMinerRank(nd)
		state         = crpc.Client().State()
		comp          bool
		numGenerators = c.GetGeneratorsNumOfRound(r.GetRoundNumber())
		is            = rank != -1 && rank < numGenerators
	)

	if is {
		// test if we have request to skip this round
		if r.GetRoundNumber() == myFailingRound {
			logging.Logger.Info("we're still pretending to be not a generator for round", zap.Int64("round", r.GetRoundNumber()))
			return false
		}
		if config.Round(r.GetRoundNumber()) == state.GeneratorsFailureRoundNumber && r.GetTimeoutCount() == 0 {
			logging.Logger.Info("we're a failing generator for round", zap.Int64("round", r.GetRoundNumber()))
			// remember this round as failing
			myFailingRound = r.GetRoundNumber()
			return false
		}
		return true // regular round generator
	}

	var competingBlock = state.CompetingBlock
	comp = competingBlock.IsCompetingRoundGenerator(state, nd.GetKey(),
		r.GetRoundNumber())

	if comp {
		return true // competing generator
	}

	return false // is not
}

func (c *Chain) DeleteRound(ctx context.Context, r round.RoundI) {} // disable deleting rounds

func (c *Chain) DeleteRoundsBelow(roundNumber int64) {} // disable deleting rounds

func (c *Chain) ChainHasTransaction(ctx context.Context, b *block.Block, txn *transaction.Transaction) (bool, error) {
	state := crpc.Client().State()
	if state.DoubleSpendTransactionHash == txn.Hash {
		return false, nil
	}
	return c.chainHasTransaction(ctx, b, txn)
}

func initialStateCT(balances cstate.CommonStateContextI) error {
	// Set initial hardfork from client state
	if err := setInitialHardfork(balances); err != nil {
		return err
	}

	return nil
}

func setInitialHardfork(balances cstate.CommonStateContextI) error {
	DefaultHardforkConfig := crpc.Client().State().Hardfork
	if DefaultHardforkConfig != nil {
		h := cstate.NewHardFork(DefaultHardforkConfig.Name, DefaultHardforkConfig.Round)
		if _, err := balances.InsertTrieNode(h.GetKey(), h); err != nil {
			return common.NewError("setInitialHardfork", err.Error())
		}
	} else {
		log.Panicf("Conductor: Hardfork is nil")
	}

	return nil
}
