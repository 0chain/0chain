package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"

	"github.com/spf13/viper"

	"0chain.net/conductor/conductrpc"
)

func isIntegrationTests() bool {
	return viper.GetBool("testing.enabled")
}

func newConductRPCClient() (clinet *conductrpc.Client) {
	return conductrpc.NewClient(viper.GetString("testing.address"))
}

func (msc *MinerSmartContract) AddMinerIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *globalNode,
	balances cstate.StateContextI) (resp string, err error) {

	resp, err = msc.AddMiner(t, inputData, gn, balances)
	if err != nil {
		return
	}
	var mn = NewMinerNode()
	mn.Decode(inputData)

	var ame conductrpc.AddMinerEvent
	ame.Sender = conductrpc.NodeID(node.Self.Underlying().GetKey())
	ame.MinerID = conductrpc.NodeID(mn.ID)
	if err = msc.client.AddMiner(ame); err != nil {
		panic(err)
	}
	return
}

func (msc *MinerSmartContract) AddSharderIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *globalNode,
	balances cstate.StateContextI) (resp string, err error) {

	resp, err = msc.AddSharder(t, inputData, gn, balances)
	if err != nil {
		return
	}
	var sn = NewMinerNode()
	sn.Decode(inputData)
	var ase conductrpc.AddSharderEvent
	ase.Sender = conductrpc.NodeID(node.Self.Underlying().GetKey())
	ase.SharderID = conductrpc.NodeID(sn.ID)
	if err = msc.client.AddSharder(ase); err != nil {
		panic(err)
	}
	return
}

func (msc *MinerSmartContract) payFeesIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *globalNode,
	balances cstate.StateContextI) (resp string, err error) {

	// phase before {
	var pn *PhaseNode
	if pn, err = msc.getPhaseNode(balances); err != nil {
		return
	}
	var phaseBefore = pn.Phase
	// }

	// view change before {
	var isViewChange bool = (balances.GetBlock().Round == gn.ViewChange)
	// }

	if resp, err = msc.payFees(t, inputData, gn, balances); err != nil {
		return
	}

	// phase after {
	if pn, err = msc.getPhaseNode(balances); err != nil {
		return
	}
	if pn.Phase != phaseBefore {
		var pe conductrpc.PhaseEvent
		pe.Phase = conductrpc.Phase(pn.Phase)
		pe.Sender = conductrpc.NodeID(node.Self.Underlying().GetKey())
		if err = msc.client.Phase(pe); err != nil {
			panic(err)
		}
	}
	// }

	// view change after {
	if isViewChange {
		var mb = balances.GetBlock().MagicBlock
		if mb == nil {
			panic("missing magic block on view change")
		}

		var vc conductrpc.ViewChangeEvent
		vc.Round = conductrpc.Round(balances.GetBlock().Round)
		vc.Sender = conductrpc.NodeID(node.Self.Underlying().GetKey())

		for _, sid := range mb.Sharders.Keys() {
			vc.Sharders = append(vc.Sharders, conductrpc.NodeID(sid))
		}

		for _, mid := range mb.Miners.Keys() {
			vc.Miners = append(vc.Miners, conductrpc.NodeID(mid))
		}

		if err = msc.client.ViewChange(vc); err != nil {
			panic(err)
		}
	}
	// }

	return
}
