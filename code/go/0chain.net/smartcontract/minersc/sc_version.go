package minersc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"github.com/blang/semver/v4"
)

// getSCVersion gets the sc_version from MPT
func getSCVersion(balances cstate.StateContextI) (*VersionNode, error) {
	nodesBytes, err := balances.GetTrieNode(SCVersionKey)
	if err != nil {
		return nil, err
	}

	var sv VersionNode
	if err = sv.Decode(nodesBytes.Encode()); err != nil {
		return nil, err
	}

	return &sv, nil
}

// updateSCVersion updates the sc_version
func updateSCVersion(state cstate.StateContextI, version string) error {
	vn := VersionNode(version)
	if _, err := state.InsertTrieNode(SCVersionKey, &vn); err != nil {
		return common.NewError("update_sc_version", err.Error())
	}
	return nil
}

// NewUpdateSCVersionTxnData creates the transaction data for updating sc version
func NewUpdateSCVersionTxnData(version string) (*sci.SmartContractTransactionData, error) {
	txnInput := &UpdateVersionTxnInput{Version: version}
	inputData, err := txnInput.Encode()
	if err != nil {
		return nil, err
	}
	return &sci.SmartContractTransactionData{
		FunctionName: "update_sc_version",
		InputData:    inputData,
	}, nil
}

// updateSCVersion updates the smart contract version node `sc_version` in MPT
func (msc *MinerSmartContract) updateSCVersion(
	t *transaction.Transaction,
	inputData []byte,
	_ *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {

	logging.Logger.Debug("update sc version")

	if t.ClientID != owner {
		return "", common.NewError("update_sc_version_unauthorized_access",
			"only the owner can update the smart contract version")
	}

	allowedV, ok, switchAdapterFunc := balances.CanUpdateSCVersion()
	if !ok {
		return "", common.NewError("update_sc_version_not_allowed",
			"smart contract version cannot be updated yet")
	}

	if switchAdapterFunc != nil {
		if err := switchAdapterFunc(balances.GetState()); err != nil {
			return "", common.NewError("update_sc_version_invalid_adapter", err.Error())
		}
	}

	var scv UpdateVersionTxnInput
	if err = scv.Decode(inputData); err != nil {
		return "", common.NewError("update_sc_version_invalid_txn_input", err.Error())
	}

	// parse the version
	newSCV, err := semver.Make(scv.Version)
	if err != nil {
		return "", common.NewError("update_sc_version_invalid_version",
			fmt.Sprintf("parse smart contract version failed, %v", err.Error()))
	}

	if !newSCV.Equals(*allowedV) {
		return "", common.NewError("update_sc_version_not_allowed",
			"smart contract version is not allowed")
	}

	// switch to the new smart contract version
	if err := updateSCVersion(balances, scv.Version); err != nil {
		return "", common.NewError("update_sc_version_save_error", err.Error())
	}

	return scv.Version, nil
}
