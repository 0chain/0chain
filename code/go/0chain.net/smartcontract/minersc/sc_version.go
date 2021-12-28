package minersc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"github.com/blang/semver/v4"
)

// SCVersionNode represents the smart contract version node stores in MPT
type SCVersionNode string

func (v SCVersionNode) Encode() []byte {
	return []byte(v)
}

func (v *SCVersionNode) Decode(b []byte) error {
	*v = SCVersionNode(b)
	return nil
}

func (v *SCVersionNode) String() string {
	return string(*v)
}

// GetSCVersion gets the sc_version from MPT
func GetSCVersion(balances cstate.StateContextI) (*SCVersionNode, error) {
	nodesBytes, err := balances.GetTrieNode(SCVersionKey)
	if err != nil {
		return nil, err
	}

	var sv SCVersionNode
	if err = sv.Decode(nodesBytes.Encode()); err != nil {
		return nil, err
	}

	return &sv, nil
}

// updateSCVersion updates the sc_version
func updateSCVersion(state cstate.StateContextI, version string) error {
	vn := SCVersionNode(version)
	if _, err := state.InsertTrieNode(SCVersionKey, &vn); err != nil {
		return common.NewError("update_sc_version", err.Error())
	}
	return nil
}

// NewUpdateSCVersionTxnData creates the transaction data for updating sc version
func NewUpdateSCVersionTxnData(version string) (*sci.SmartContractTransactionData, error) {
	txnInput := &UpdateSCVersionTxnInput{Version: version}
	inputData, err := txnInput.Encode()
	if err != nil {
		return nil, err
	}
	return &sci.SmartContractTransactionData{
		FunctionName: "update_sc_version",
		InputData:    inputData,
	}, nil
}

// UpdateSCVersionTxnInput represents the transaction data struct for
// updating the smart contract version
type UpdateSCVersionTxnInput struct {
	Version string `json:"version"`
}

// Decode implements the mpt node decode interface
func (v *UpdateSCVersionTxnInput) Decode(b []byte) error {
	return json.Unmarshal(b, v)
}

// Encode implements the mpt node encode interface
func (v *UpdateSCVersionTxnInput) Encode() ([]byte, error) {
	b, err := json.Marshal(v)
	return b, err
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

	allowedV, ok := balances.CanUpdateSCVersion()
	if !ok {
		return "", common.NewError("update_sc_version_not_allowed",
			"smart contract version cannot be updated yet")
	}

	var scv UpdateSCVersionTxnInput
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
