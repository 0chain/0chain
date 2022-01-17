package minersc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"github.com/blang/semver/v4"
)

// getProtoVersion gets the sc_version from MPT
func getProtoVersion(balances cstate.StateContextI) (*VersionNode, error) {
	nodesBytes, err := balances.GetTrieNode(ProtoVersionKey)
	if err != nil {
		return nil, err
	}

	var sv VersionNode
	if err = sv.Decode(nodesBytes.Encode()); err != nil {
		return nil, err
	}

	return &sv, nil
}

// updateProtoVersion updates the sc_version
func updateProtoVersion(state cstate.StateContextI, version string) error {
	vn := VersionNode(version)
	if _, err := state.InsertTrieNode(ProtoVersionKey, &vn); err != nil {
		return common.NewError("update_proto_version", err.Error())
	}
	return nil
}

// updateProtoVersion updates the smart contract version node `sc_version` in MPT
func (msc *MinerSmartContract) updateProtoVersion(
	t *transaction.Transaction,
	inputData []byte,
	_ *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {

	logging.Logger.Debug("update protocol version")

	// TODO: only owner are allowed to update the version
	if t.ClientID != owner {
		return "", common.NewError("update_sc_version_unauthorized_access",
			"only the owner can update the smart contract version")
	}

	allowedV, ok, switchAdapterFunc := balances.CanUpdateProtoVersion()
	if !ok {
		return "", common.NewError("update_proto_version_not_allowed",
			"protocol version cannot be updated yet")
	}

	if switchAdapterFunc != nil {
		if err := switchAdapterFunc(balances.GetState()); err != nil {
			return "", common.NewError("update_proto_version_invalid_adapter", err.Error())
		}
	}

	var pv UpdateVersionTxnInput
	if err = pv.Decode(inputData); err != nil {
		return "", common.NewError("update_proto_version_invalid_txn_input", err.Error())
	}

	// parse the version
	newProtoV, err := semver.Make(pv.Version)
	if err != nil {
		return "", common.NewError("update_proto_version_invalid_version",
			fmt.Sprintf("parse protocol version failed, %v", err.Error()))
	}

	if !newProtoV.Equals(*allowedV) {
		return "", common.NewError("update_proto_version_not_allowed",
			"protocol version is not allowed")
	}

	// switch to the new protocol version
	if err := updateProtoVersion(balances, pv.Version); err != nil {
		return "", common.NewError("update_proto_version_save_error", err.Error())
	}

	return pv.Version, nil
}
