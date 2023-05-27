//go:build integration_tests
// +build integration_tests

package event

import (
	"0chain.net/chaincore/node"
	"0chain.net/conductor/conductrpc"
)

func (edb *EventDb) addStat(event Event) (err error) {
	err = edb.addStatMain(event)
	if err != nil {
		return
	}

	var (
		client = conductrpc.Client()
		state  = client.State()
	)

	if !state.IsMonitor {
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
				Sender: state.Name(conductrpc.NodeID(node.Self.Underlying().GetKey())),
				Miner:  state.Name(conductrpc.NodeID(miner.ID)),
			}

			if ame.Miner == conductrpc.NodeName("") {
				continue
			}
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
