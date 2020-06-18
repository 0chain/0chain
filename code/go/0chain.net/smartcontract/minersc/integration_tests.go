// +build integration_tests

package minersc

import (
	"sync"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"

	"github.com/spf13/viper"

	. "0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/conductor/conductrpc"
)

func (msc *MinerSmartContract) InitSC() {
	if msc.smartContractFunctions == nil {
		msc.smartContractFunctions = make(map[string]smartContractFunction)
	}
	phaseFuncs[Start] = msc.createDKGMinersForContribute
	phaseFuncs[Contribute] = msc.widdleDKGMinersForShare
	phaseFuncs[Publish] = msc.createMagicBlockForWait

	const pfx = "smart_contracts.minersc."
	var scc = config.SmartContractConfig

	PhaseRounds[Start] = scc.GetInt64(pfx + "start_rounds")
	PhaseRounds[Contribute] = scc.GetInt64(pfx + "contribute_rounds")
	PhaseRounds[Share] = scc.GetInt64(pfx + "share_rounds")
	PhaseRounds[Publish] = scc.GetInt64(pfx + "publish_rounds")
	PhaseRounds[Wait] = scc.GetInt64(pfx + "wait_rounds")

	moveFunctions[Start] = msc.moveToContribute
	moveFunctions[Contribute] = msc.moveToShareOrPublish
	moveFunctions[Share] = msc.moveToShareOrPublish
	moveFunctions[Publish] = msc.moveToWait
	moveFunctions[Wait] = msc.moveToStart

	// wrapped
	msc.smartContractFunctions["add_miner"] = msc.AddMinerIntegrationTests
	msc.smartContractFunctions["add_sharder"] = msc.AddSharderIntegrationTests
	msc.smartContractFunctions["payFees"] = msc.payFeesIntegrationTests
	msc.smartContractFunctions["contributeMpk"] = msc.contributeMpkIntegrationTests
	msc.smartContractFunctions["shareSignsOrShares"] = msc.shareSignsOrSharesIntegrationTests
	// as is
	msc.smartContractFunctions["update_settings"] = msc.UpdateSettings
	msc.smartContractFunctions["addToDelegatePool"] = msc.addToDelegatePool
	msc.smartContractFunctions["deleteFromDelegatePool"] = msc.deleteFromDelegatePool
	msc.smartContractFunctions["sharder_keep"] = msc.sharderKeep
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
	if err = msc.client.AddMiner(&ame); err != nil {
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
	if err = msc.client.AddSharder(&ase); err != nil {
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

	// call the wrapped function {
	if resp, err = msc.payFees(t, inputData, gn, balances); err != nil {
		return
	}
	// }

	// events order
	// - round
	// - view change
	// - phase

	// round {
	var re conductrpc.RoundEvent
	re.Round = conductrpc.Round(balances.GetBlock().Round)
	re.Sender = conductrpc.NodeID(node.Self.Underlying().GetKey())
	if err = msc.client.Round(&re); err != nil {
		panic(err)
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

		if err = msc.client.ViewChange(&vc); err != nil {
			panic(err)
		}
	}
	// }

	// phase after {
	if pn, err = msc.getPhaseNode(balances); err != nil {
		return
	}
	if pn.Phase != phaseBefore {
		var pe conductrpc.PhaseEvent
		pe.Phase = conductrpc.Phase(pn.Phase)
		pe.Sender = conductrpc.NodeID(node.Self.Underlying().GetKey())
		if err = msc.client.Phase(&pe); err != nil {
			panic(err)
		}
	}
	// }

	return
}

func (msc *MinerSmartContract) contributeMpkIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *globalNode,
	balances cstate.StateContextI) (resp string, err error) {

	resp, err = msc.contributeMpk(t, inputData, gn, balances)
	if err != nil {
		return
	}

	var cmpke conductrpc.ContributeMPKEvent
	cmpke.Sender = conductrpc.NodeID(node.Self.Underlying().GetKey())
	cmpke.MinerID = conductrpc.NodeID(t.ClientID)
	if err = msc.client.ContributeMPK(&cmpke); err != nil {
		panic(err)
	}

	return
}

func (msc *MinerSmartContract) shareSignsOrSharesIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *globalNode,
	balances cstate.StateContextI) (resp string, err error) {

	resp, err = msc.shareSignsOrShares(t, inputData, gn, balances)
	if err != nil {
		return
	}

	var ssose conductrpc.ShareOrSignsSharesEvent
	ssose.Sender = conductrpc.NodeID(node.Self.Underlying().GetKey())
	ssose.MinerID = conductrpc.NodeID(t.ClientID)
	if err = msc.client.ShareOrSignsShares(&ssose); err != nil {
		panic(err)
	}

	return
}
