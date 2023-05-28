//go:build integration_tests
// +build integration_tests

package event

import (
	"fmt"

	"0chain.net/chaincore/node"
	"0chain.net/conductor/conductrpc"
	"github.com/0chain/common/core/logging"
)

func (edb *EventDb) addStat(event Event) (err error) {
	logging.Logger.Info(fmt.Sprintf("Adding stat: %v", event))
	err = edb.addStatMain(event)
	if err != nil {
		return
	}

	var (
		client = conductrpc.Client()
		state  = client.State()
		sender = state.Name(conductrpc.NodeID(node.Self.Underlying().GetKey()))
	)

	if !state.IsMonitor {
		logging.Logger.Info(fmt.Sprintf("skipping as %s is not monitor", sender))
		return
	}

	switch event.Tag {
	case TagAddMiner:
		miners, ok := fromEvent[[]Miner](event.Data)
		if !ok {
			return
		}

		for _, miner := range *miners {
			ame := conductrpc.AddMinerEvent{
				Sender: sender,
				Miner:  state.Name(conductrpc.NodeID(miner.ID)),
			}

			if ame.Miner == conductrpc.NodeName("") {
				continue
			}
			logging.Logger.Info(fmt.Sprintf("Sending %s to conductor server", ame.Miner))
			if err := client.AddMiner(&ame); err != nil {
				panic(err)
			}
		}
		//
	case TagAddBlobber:
		//
	case TagAddSharder:
		//
	case TagAddOrOverwiteValidator:
		//
	case TagAddAuthorizer:
		//
	}
	return nil
}
