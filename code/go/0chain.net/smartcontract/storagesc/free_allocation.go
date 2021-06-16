package storagesc

import (
	"0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

const (
	floatToBalance = 10 * 1000 * 1000 * 1000
	YEAR           = time.Duration(time.Hour * 8784)
)

type freeStorageMarker struct {
	Giver      string           `json:"giver"`
	Recipient  string           `json:"recipient"`
	FreeTokens float64          `json:"free_tokens"`
	Timestamp  common.Timestamp `json:"timestamp"`
	Signature  string           `json:"signature"`
}

func (frm *freeStorageMarker) decode(b []byte) error {
	return json.Unmarshal(b, frm)
}

type newFreeStorageAssignerInfo struct {
	ClientId         string
	PublicKey        string
	AnnualTokenLimit float64
}

func (frm *newFreeStorageAssignerInfo) decode(b []byte) error {
	return json.Unmarshal(b, frm)
}

type freeStorageRedeemed struct {
	Amount    state.Balance    `json:"amount"`
	When      common.Timestamp `json:"when"`
	Timestamp common.Timestamp `json:"timestamp"`
}

func freeStorageAssignerKey(sscKey, clientId string) datastore.Key {
	return datastore.Key(sscKey + ":freestorageredeemed:" + clientId)
}

type freeStorageAssigner struct {
	ClientId             string                `json:"client_id"`
	PublicKey            string                `json:"public_key"`
	AnnualLimit          state.Balance         `json:"annual_limit"`
	FreeStoragesRedeemed []freeStorageRedeemed `json:"free_storages_redeemed"`
}

func (fsa *freeStorageAssigner) Encode() []byte {
	var b, err = json.Marshal(fsa)
	if err != nil {
		panic(err)
	}
	return b
}

func (fsa *freeStorageAssigner) Decode(p []byte) error {
	return json.Unmarshal(p, fsa)
}

func (fsa *freeStorageAssigner) save(sscKey string, balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(freeStorageAssignerKey(sscKey, fsa.ClientId), fsa)
	return err
}

func (fsa *freeStorageAssigner) validate(
	marker freeStorageMarker,
	now common.Timestamp,
	annualLimit state.Balance,
) error {
	verified, err := verifyFreeAllocationRequest(marker, fsa.PublicKey)
	if err != nil {
		return err
	}
	if !verified {
		return common.NewErrInternal("failed to verify signature")
	}

	yearStartIndex := len(fsa.FreeStoragesRedeemed)
	for i, redeemed := range fsa.FreeStoragesRedeemed {
		if redeemed.When > now-common.Timestamp(YEAR.Seconds()) {
			yearStartIndex = i
			break
		}
		if marker.Timestamp == redeemed.Timestamp {
			return common.NewErrInternal("marker already redeemed")
		}
	}
	annualTotal := state.Balance(0)
	for i := yearStartIndex; i < len(fsa.FreeStoragesRedeemed); i++ {
		if marker.Timestamp == fsa.FreeStoragesRedeemed[i].Timestamp {
			return common.NewErrInternal("marker already redeemed")
		}
		annualTotal += fsa.FreeStoragesRedeemed[i].Amount
	}
	if annualTotal+state.Balance(marker.FreeTokens)*floatToBalance > annualLimit {
		return common.NewErrInternal("exceeded annual free storage limit")
	}
	return nil
}

func (ssc *StorageSmartContract) addFreeStorageAssigner(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) error {
	if t.ClientID != owner {
		return common.NewError("add_free_storage_assigner",
			"unauthorized access - only the owner can update the variables")
	}

	var info newFreeStorageAssignerInfo
	info.decode(input)

	var conf *scConfig
	var err error
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return common.NewErrorf("add_free_storage_assigner",
			"can't get config: %v", err)
	}
	var newAnnualLimit = state.Balance(info.AnnualTokenLimit * floatToBalance)
	if newAnnualLimit > conf.MaxAnnualFreeAllocation {
		return common.NewErrorf("add_free_storage_assigner",
			"annual limit exceeds maximum permitted: tokens %f", info.AnnualTokenLimit)
	}

	assigner, err := ssc.getFreeStorageAssigner(info.ClientId, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return common.NewError("add_free_storage_assigner", err.Error())
	}
	if err == util.ErrValueNotPresent {
		assigner = &freeStorageAssigner{
			ClientId: info.ClientId,
		}
	}
	assigner.PublicKey = info.PublicKey
	assigner.AnnualLimit = newAnnualLimit
	err = assigner.save(ssc.ID, balances)
	if err != nil {
		return common.NewErrorf("add_free_storage_assigner", "error saving new assigner: %v", err)
	}

	return nil
}

func verifyFreeAllocationRequest(frm freeStorageMarker, publicKey string) (bool, error) {
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
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	if err := signatureScheme.SetPublicKey(publicKey); err != nil {
		return false, err
	}
	return signatureScheme.Verify(frm.Signature, hex.EncodeToString(responseBytes))
}

func (ssc *StorageSmartContract) freeAllocationRequest(
	txn *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var err error
	var marker freeStorageMarker
	if err := marker.decode(input); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"unmarshal request: %v", err)
	}

	if marker.Timestamp >= txn.CreationDate {
		return "", common.NewErrorf("free_allocation_failed",
			"marker timestamped in the past: %v", marker.Timestamp)
	}

	var conf *scConfig
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"can't get config: %v", err)
	}

	assigner, err := ssc.getFreeStorageAssigner(marker.Giver, balances)
	if err := assigner.validate(marker, txn.CreationDate, conf.MaxAnnualFreeAllocation); err != nil {
		return "", common.NewError("free_allocation_failed",
			"marker verification failed")
	}

	var request = newAllocationRequest{
		DataShards:                 conf.FreeAllocationSettings.DataShards,
		ParityShards:               conf.FreeAllocationSettings.ParityShards,
		Size:                       conf.FreeAllocationSettings.Size,
		Expiration:                 common.Timestamp(common.ToTime(txn.CreationDate).Add(conf.FreeAllocationSettings.Duration).Unix()),
		Owner:                      txn.ClientID,
		OwnerPublicKey:             txn.PublicKey,
		ReadPriceRange:             conf.FreeAllocationSettings.ReadPriceRange,
		WritePriceRange:            conf.FreeAllocationSettings.WritePriceRange,
		MaxChallengeCompletionTime: conf.FreeAllocationSettings.MaxChallengeCompletionTime,
	}

	arBytes, err := request.encode()
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"marshal request: %v", err)
	}

	resp, sa, err := ssc.newAllocationRequestInternal(txn, arBytes, conf, balances)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed", ": %v", err)
	}
	if sa == nil {
		return "", common.NewError("free_allocation_failed", "nil allocation storage object")
	}

	sa.IsFree = true
	sa.FreeTimestamp = marker.Timestamp

	redeemed := freeStorageRedeemed{
		Timestamp: marker.Timestamp,
		When:      txn.CreationDate,
		Amount:    state.Balance(marker.FreeTokens * floatToBalance),
	}
	assigner.FreeStoragesRedeemed = append(assigner.FreeStoragesRedeemed, redeemed)
	if err := assigner.save(ssc.ID, balances); err != nil {
		return "", common.NewErrorf("free_allocation_failed", "assigner save failed: %v", err)
	}

	if resp, err = ssc.addAllocation(sa, balances); err != nil {
		return "", common.NewErrorf("free_allocation_failed", "%v", err)
	}

	// create write pool and lock tokens
	if err = ssc.createWritePool(txn, sa, balances); err != nil {
		return "", common.NewError("allocation_creation_failed", err.Error())
	}

	return resp, err
}

func (ssc *StorageSmartContract) updateFreeStorageRequest(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	return "", nil
}

func (ssc *StorageSmartContract) getFreeStorageAssignerBytes(
	clientID datastore.Key,
	balances cstate.StateContextI,
) ([]byte, error) {
	var val util.Serializable
	val, err := balances.GetTrieNode(freeStorageAssignerKey(ssc.ID, clientID))
	if err != nil {
		return nil, err
	}
	return val.Encode(), nil
}

// getWritePool of current client
func (ssc *StorageSmartContract) getFreeStorageAssigner(
	clientID datastore.Key,
	balances cstate.StateContextI,
) (*freeStorageAssigner, error) {
	var err error
	var aBytes []byte
	if aBytes, err = ssc.getFreeStorageAssignerBytes(clientID, balances); err != nil {
		return nil, err
	}
	fsa := new(freeStorageAssigner)
	err = fsa.Decode(aBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return fsa, nil
}
