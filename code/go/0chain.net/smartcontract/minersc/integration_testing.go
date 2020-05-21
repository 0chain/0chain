package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
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

	if err = msc.client.AddMiner(conductrpc.MinerID(mn.ID)); err != nil {
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
	var mn = NewMinerNode()
	mn.Decode(inputData)
	if err = msc.client.AddSharder(conductrpc.SharderID(mn.ID)); err != nil {
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
		if err = msc.client.Phase(conductrpc.Phase(pn.Phase)); err != nil {
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

		var vc conductrpc.ViewChange
		vc.Round = balances.GetBlock().Round

		for _, sid := range mb.Sharders.Keys() {
			vc.Sharders = append(vc.Sharders, conductrpc.SharderID(sid))
		}

		for _, mid := range mb.Miners.Keys() {
			vc.Miners = append(vc.Miners, conductrpc.MinerID(mid))
		}

		if err = msc.client.ViewChange(vc); err != nil {
			panic(err)
		}
	}
	// }

	return
}
