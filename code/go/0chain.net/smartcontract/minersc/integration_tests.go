//go:build integration_tests
// +build integration_tests

package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"

	crpc "0chain.net/conductor/conductrpc"
)

func (msc *MinerSmartContract) initSC() {
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
	msc.smartContractFunctions["sharder_keep"] = msc.sharderKeepIntegrationTests
	// as is
	msc.smartContractFunctions["wait"] = msc.wait
	msc.smartContractFunctions["update_globals"] = msc.updateGlobals
	msc.smartContractFunctions["update_miner_settings"] = msc.UpdateMinerSettings
	msc.smartContractFunctions["update_sharder_settings"] = msc.UpdateSharderSettings
	msc.smartContractFunctions["update_settings"] = msc.updateSettings
	msc.smartContractFunctions["addToDelegatePool"] = msc.addToDelegatePool
	msc.smartContractFunctions["deleteFromDelegatePool"] = msc.deleteFromDelegatePool
	msc.smartContractFunctions["sharder_keep"] = msc.sharderKeep
}

func (msc *MinerSmartContract) AddMinerIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	resp, err = msc.AddMiner(t, inputData, gn, balances)
	if err != nil {
		return
	}
	var mn = NewMinerNode()
	mn.Decode(inputData)

	var (
		client = crpc.Client()
		state  = client.State()
		ame    crpc.AddMinerEvent
	)
	ame.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
	ame.Miner = state.Name(crpc.NodeID(mn.ID))
	if err = client.AddMiner(&ame); err != nil {
		panic(err)
	}
	return
}

func (msc *MinerSmartContract) AddSharderIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	resp, err = msc.AddSharder(t, inputData, gn, balances)
	if err != nil {
		return
	}
	var sn = NewMinerNode()
	sn.Decode(inputData)
	var (
		client = crpc.Client()
		state  = client.State()
		ase    crpc.AddSharderEvent
	)
	ase.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
	ase.Sharder = state.Name(crpc.NodeID(sn.ID))
	if err = client.AddSharder(&ase); err != nil {
		panic(err)
	}
	return
}

func (msc *MinerSmartContract) payFeesIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	// phase before {
	var pn *PhaseNode
	if pn, err = GetPhaseNode(balances); err != nil {
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
	var (
		client = crpc.Client()
		state  = client.State()
		re     crpc.RoundEvent
	)
	re.Round = crpc.Round(balances.GetBlock().Round)
	re.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
	if err = client.Round(&re); err != nil {
		panic(err)
	}
	// }

	// view change after {
	if isViewChange {
		var mb = balances.GetBlock().MagicBlock
		if mb != nil {
			var vc crpc.ViewChangeEvent
			vc.Round = crpc.Round(balances.GetBlock().Round)
			vc.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
			vc.Number = crpc.Number(mb.MagicBlockNumber)

			for _, sid := range mb.Sharders.Keys() {
				vc.Sharders = append(vc.Sharders, state.Name(crpc.NodeID(sid)))
			}

			for _, mid := range mb.Miners.Keys() {
				vc.Miners = append(vc.Miners, state.Name(crpc.NodeID(mid)))
			}

			if err = client.ViewChange(&vc); err != nil {
				panic(err)
			}
		}
	}
	// }

	// phase after {
	if pn, err = GetPhaseNode(balances); err != nil {
		return
	}
	if pn.Phase != phaseBefore {
		var pe crpc.PhaseEvent
		pe.Phase = crpc.Phase(pn.Phase)
		pe.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
		if err = client.Phase(&pe); err != nil {
			panic(err)
		}
	}
	// }

	return
}

func (msc *MinerSmartContract) contributeMpkIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	resp, err = msc.contributeMpk(t, inputData, gn, balances)
	if err != nil {
		return
	}

	var (
		client = crpc.Client()
		state  = client.State()
		cmpke  crpc.ContributeMPKEvent
	)
	cmpke.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
	cmpke.Miner = state.Name(crpc.NodeID(t.ClientID))
	if err = client.ContributeMPK(&cmpke); err != nil {
		panic(err)
	}

	return
}

func (msc *MinerSmartContract) shareSignsOrSharesIntegrationTests(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	resp, err = msc.shareSignsOrShares(t, inputData, gn, balances)
	if err != nil {
		return
	}

	var (
		client = crpc.Client()
		state  = client.State()
		ssose  crpc.ShareOrSignsSharesEvent
	)
	ssose.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
	ssose.Miner = state.Name(crpc.NodeID(t.ClientID))
	if err = client.ShareOrSignsShares(&ssose); err != nil {
		panic(err)
	}

	return
}

func (msc *MinerSmartContract) sharderKeepIntegrationTests(
	t *transaction.Transaction, input []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	if resp, err = msc.sharderKeep(t, input, gn, balances); err != nil {
		return // error
	}

	var mn = NewMinerNode()
	if err = mn.Decode(input); err != nil {
		panic(err) // must not happen, because of the successful call above
	}

	var (
		client = crpc.Client()
		state  = client.State()
		ske    crpc.SharderKeepEvent
	)
	ske.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
	ske.Sharder = state.Name(crpc.NodeID(mn.ID))
	if err = client.SharderKeep(&ske); err != nil {
		panic(err)
	}

	return
}
