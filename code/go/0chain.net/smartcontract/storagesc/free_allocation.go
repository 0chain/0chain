package storagesc

import (
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract"
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

type freeRequestRequestMarker struct {
	Recipient  string           `json:"recipient"`
	FreeTokens float64          `json:"free_tokens"`
	Timestamp  common.Timestamp `json:"timestamp"`
	Signature  string           `json:"signature"`
}

func (frm *freeRequestRequestMarker) decode(b []byte) error {
	return json.Unmarshal(b, frm)
}

func (ssc *StorageSmartContract) GetFreeStorageMarker(
	ctx context.Context,
	params url.Values,
	balances state.StateContextI,
) (interface{}, error) {
	var amount, err = strconv.ParseFloat(params.Get("amount"), 64)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "cannot read amount")
	}
	if amount <= 0.0 {
		common.NewErrorf("get_free_storage_marker", "marker amount %f out of range", amount)
	}

	return &freeRequestRequestMarker{
		Recipient:  params.Get("recipient"),
		FreeTokens: float64(amount),
		Timestamp:  common.Now(),
	}, nil
}

func (ssc *StorageSmartContract) GetTopUpStorageMarker(
	ctx context.Context,
	params url.Values,
	balances state.StateContextI,
) (interface{}, error) {

	return nil, nil
}

func verifyFreeAllocationRequest(frm freeRequestRequestMarker) (bool, error) {
	var request = struct {
		Recipient  string           `json:"recipient"`
		FreeTokens float64          `json:"free_tokens"`
		Timestamp  common.Timestamp `json:"timestamp"`
	}{
		frm.Recipient, frm.FreeTokens, frm.Timestamp,
	}
	responseBytes, err := json.Marshal(&request)
	if err != nil {
		return false, err
	}
	signatureHash := string(encryption.RawHash(responseBytes))
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	signatureScheme.SetPublicKey(owner)
	return signatureScheme.Verify(frm.Signature, signatureHash)
}

func (ssc *StorageSmartContract) freeAllocationRequest(
	t *transaction.Transaction,
	input []byte,
	balances state.StateContextI,
) (string, error) {
	var err error
	var frm freeRequestRequestMarker
	if err := frm.decode(input); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"unmarshal request: %v", err)
	}

	verified, err := verifyFreeAllocationRequest(frm)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"error verifying request: %v", err)
	}
	if !verified {
		return "", common.NewError("free_allocation_failed",
			"marker verification failed")
	}

	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"can't get config: %v", err)
	}

	var request = newAllocationRequest{
		DataShards:                 conf.FreeAllocationSettings.DataShards,
		ParityShards:               conf.FreeAllocationSettings.ParityShards,
		Size:                       conf.FreeAllocationSettings.Size,
		Expiration:                 common.Timestamp(common.ToTime(t.CreationDate).Add(conf.FreeAllocationSettings.Duration).Unix()),
		Owner:                      t.ClientID,
		OwnerPublicKey:             t.PublicKey,
		ReadPriceRange:             conf.FreeAllocationSettings.ReadPriceRange,
		WritePriceRange:            conf.FreeAllocationSettings.WritePriceRange,
		MaxChallengeCompletionTime: conf.FreeAllocationSettings.MaxChallengeCompletionTime,
	}

	arBytes, err := request.encode()
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"marshal request: %v", err)
	}

	resp, sa, err := ssc.newAllocationRequestInternal(t, arBytes, balances)

	sa.IsFree = true
	sa.FreeTimestamp = frm.Timestamp

	if resp, err = ssc.addAllocation(sa, balances); err != nil {
		return "", common.NewErrorf("free_allocation_failed", "%v", err)
	}

	return resp, err
}

func (ssc *StorageSmartContract) updateFreeStorageRequest(
	t *transaction.Transaction,
	input []byte,
	balances state.StateContextI,
) (string, error) {
	return "", nil
}
