package sharder

import (
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/smartcontract/minersc"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

type ContributeCallbackSetter interface {
	SetCallbackPhase(callback func(phase minersc.Phase))
}

// SetupMinerSmartContract  sets callbacks for shader phases MinerSC
// This solution is due to the fact that in MinerSC the shader did not want to complicate with a state machine with phases
func SetupMinerSmartContract(serverChain *Chain) {
	var (
		scs    = smartcontract.GetSmartContract(minersc.ADDRESS)
		setter = scs.(ContributeCallbackSetter)
	)

	setter.SetCallbackPhase(func(phase minersc.Phase) {
		if !config.DevConfiguration.ViewChange {
			return // no view change, no sharder keep
		}
		println(":::: PHASE CALLBACK", phase)
		if phase == minersc.Contribute {
			go registerSharderKeepOnContributeInCallback(serverChain)
		}
	})
}

func registerSharderKeepOnContributeInCallback(sc *Chain) {
	println("REGISTER SHARDER KEEP IN CONTRIBUTE CALLBACK")
	var txn, err = sc.RegisterSharderKeep()
	if err != nil {
		println("REGISTER SHARDER KEEP IN CONTRIBUTE CALLBACK (ERR 1)")
		Logger.Error("register_sharder_keep", zap.Error(err))
		return
	}
	if txn == nil || sc.ConfirmTransaction(txn) {
		println("REGISTER SHARDER KEEP IN CONTRIBUTE CALLBACK (OK 2)")
		Logger.Info("register_sharder_keep -- registered")
		return
	}
	println("REGISTER SHARDER KEEP IN CONTRIBUTE CALLBACK (ERR 3)")
	Logger.Debug("register_sharder_keep -- failed to confirm transaction",
		zap.Any("txn", txn))
}
