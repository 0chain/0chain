// +build !integration_tests

package minersc

import (
	"0chain.net/chaincore/config"
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

	msc.smartContractFunctions["add_miner"] = msc.AddMiner
	msc.smartContractFunctions["add_sharder"] = msc.AddSharder
	msc.smartContractFunctions["update_miner_settings"] = msc.UpdateMinerSettings
	msc.smartContractFunctions["update_sharder_settings"] = msc.UpdateSharderSettings
	msc.smartContractFunctions["delete_miner"] = msc.DeleteMiner
	msc.smartContractFunctions["delete_sharder"] = msc.DeleteSharder

	msc.smartContractFunctions["miner_health_check"] = msc.minerHealthCheck
	msc.smartContractFunctions["sharder_health_check"] = msc.sharderHealthCheck

	msc.smartContractFunctions["payFees"] = msc.payFees

	msc.smartContractFunctions["contributeMpk"] = msc.contributeMpk
	msc.smartContractFunctions["shareSignsOrShares"] = msc.shareSignsOrShares
	msc.smartContractFunctions["wait"] = msc.wait

	msc.smartContractFunctions["update_settings"] = msc.UpdateSettings

	msc.smartContractFunctions["addToDelegatePool"] = msc.addToDelegatePool
	msc.smartContractFunctions["deleteFromDelegatePool"] = msc.deleteFromDelegatePool

	msc.smartContractFunctions["sharder_keep"] = msc.sharderKeep
}
