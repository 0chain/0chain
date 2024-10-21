//go:build !integration_tests
// +build !integration_tests

package minersc

import (
	"0chain.net/core/config"
)

func (msc *MinerSmartContract) initSC() {
	msc.InitSmartContractFunctions()

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
}

func (msc *MinerSmartContract) InitSmartContractFunctions() {
	if msc.smartContractFunctions == nil {
		msc.smartContractFunctions = make(map[string]smartContractFunction)
	}
	msc.smartContractFunctions["add_miner"] = msc.AddMiner
	msc.smartContractFunctions["add_sharder"] = msc.AddSharder
	msc.smartContractFunctions["vc_add"] = msc.VCAdd
	msc.smartContractFunctions["delete_miner"] = msc.DeleteMiner
	msc.smartContractFunctions["delete_sharder"] = msc.DeleteSharder
	msc.smartContractFunctions["collect_reward"] = msc.collectReward

	msc.smartContractFunctions["kill_miner"] = msc.killMiner
	msc.smartContractFunctions["kill_sharder"] = msc.killSharder

	msc.smartContractFunctions["miner_health_check"] = msc.minerHealthCheck
	msc.smartContractFunctions["sharder_health_check"] = msc.sharderHealthCheck

	msc.smartContractFunctions["payFees"] = msc.payFees

	msc.smartContractFunctions["contributeMpk"] = msc.contributeMpk
	msc.smartContractFunctions["shareSignsOrShares"] = msc.shareSignsOrShares
	msc.smartContractFunctions["wait"] = msc.wait
	msc.smartContractFunctions["update_globals"] = msc.updateGlobals
	msc.smartContractFunctions["update_settings"] = msc.updateSettings
	msc.smartContractFunctions["update_miner_settings"] = msc.UpdateMinerSettings
	msc.smartContractFunctions["update_sharder_settings"] = msc.UpdateSharderSettings

	msc.smartContractFunctions["addToDelegatePool"] = msc.addToDelegatePool
	msc.smartContractFunctions["deleteFromDelegatePool"] = msc.deleteFromDelegatePool

	msc.smartContractFunctions["sharder_keep"] = msc.sharderKeep
	msc.smartContractFunctions["add_hardfork"] = msc.addHardFork
}
