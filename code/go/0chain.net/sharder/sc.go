package sharder

import (
	"0chain.net/chaincore/smartcontract"
	. "0chain.net/core/logging"
	"0chain.net/smartcontract/minersc"
	"go.uber.org/zap"
)

// SetupMinerSmartContract  sets callbacks for shader phases MinerSC
// This solution is due to the fact that in MinerSC the shader did not want to complicate with a state machine with phases
func SetupMinerSmartContract(serverChain *Chain) {
	scs := smartcontract.GetSmartContract(minersc.ADDRESS)
	setterCallback := scs.(interface{ SetCallbackPhase(func(int)) })
	setterCallback.SetCallbackPhase(func(phase int) {
		if phase == minersc.Contribute {
			go registerSharderKeepOnContributeInCallback(serverChain)
		}
	})
}

func registerSharderKeepOnContributeInCallback(sc *Chain) {
	var txn, err = sc.RegisterSharderKeep()
	if err != nil {
		Logger.Error("register_sharder_keep", zap.Error(err))
		return
	}
	if txn == nil || sc.ConfirmTransaction(txn) {
		Logger.Info("register_sharder_keep -- registered")
		return
	}
	Logger.Debug("register_sharder_keep -- failed to confirm transaction",
		zap.Any("txn", txn))
}
