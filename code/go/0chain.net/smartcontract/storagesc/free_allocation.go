package storagesc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/smartcontractinterface"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

const (
	floatToBalance = 10 * 1000 * 1000 * 1000
)

//msgp:ignore freeStorageAllocationInput newFreeStorageAssignerInfo
//go:generate msgp -io=false -tests=false -unexported=true -v

type freeStorageMarker struct {
	Assigner   string           `json:"assigner"`
	Recipient  string           `json:"recipient"`
	FreeTokens float64          `json:"free_tokens"`
	Timestamp  common.Timestamp `json:"timestamp"`
	Signature  string           `json:"signature"`
}

func (frm *freeStorageMarker) decode(b []byte) error {
	return json.Unmarshal(b, frm)
}

type freeStorageAllocationInput struct {
	RecipientPublicKey string   `json:"recipient_public_key"`
	Marker             string   `json:"marker"`
	Blobbers           []string `json:"blobbers"`
}

func (frm *freeStorageAllocationInput) decode(b []byte) error {
	return json.Unmarshal(b, frm)
}

type freeStorageUpgradeInput struct {
	AllocationId string `json:"allocation_id"`
	Marker       string `json:"marker"`
}

type newFreeStorageAssignerInfo struct {
	Name            string  `json:"name"`
	PublicKey       string  `json:"public_key"`
	IndividualLimit float64 `json:"individual_limit"`
	TotalLimit      float64 `json:"total_limit"`
}

func (frm *newFreeStorageAssignerInfo) decode(b []byte) error {
	return json.Unmarshal(b, frm)
}

func freeStorageAssignerKey(sscKey, clientId string) datastore.Key {
	return sscKey + ":freestorageredeemed:" + clientId
}

type freeStorageAssigner struct {
	ClientId           string             `json:"client_id"`
	PublicKey          string             `json:"public_key"`
	IndividualLimit    currency.Coin      `json:"individual_limit"`
	TotalLimit         currency.Coin      `json:"total_limit"`
	CurrentRedeemed    currency.Coin      `json:"current_redeemed"`
	RedeemedTimestamps []common.Timestamp `json:"redeemed_timestamps"`
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
	value currency.Coin,
	balances cstate.StateContextI,
) error {
	if marker.Timestamp >= now {
		return fmt.Errorf("marker timestamped in the future: %v", marker.Timestamp)
	}

	verified, err := verifyFreeAllocationRequest(marker, fsa.PublicKey, balances)
	if err != nil {
		return err
	}
	if !verified {
		return fmt.Errorf("failed to verify signature")
	}

	newTotal := fsa.CurrentRedeemed + value
	if newTotal > fsa.TotalLimit {
		return fmt.Errorf("%d exceeded total permitted free storage limit %d", newTotal, fsa.TotalLimit)
	}

	if value > fsa.IndividualLimit {
		return fmt.Errorf("%d exceeded permitted free storage  %d", value, fsa.IndividualLimit)
	}

	for _, timestamp := range fsa.RedeemedTimestamps {
		if marker.Timestamp == timestamp {
			return fmt.Errorf("marker already redeemed, timestamp: %v", marker.Timestamp)
		}
	}

	return nil
}

func (ssc *StorageSmartContract) addFreeStorageAssigner(
	t *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var conf *Config
	var err error
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("add_free_storage_assigner",
			"can't get config: %v", err)
	}

	if err := smartcontractinterface.AuthorizeWithOwner("add_free_storage_assigner", func() bool {
		return conf.OwnerId == t.ClientID
	}); err != nil {
		return "", err
	}

	var assignerInfo newFreeStorageAssignerInfo
	if err := assignerInfo.decode(input); err != nil {
		return "", common.NewErrorf("add_free_storage_assigner",
			"can't unmarshal input: %v", err)
	}

	var newTotalLimit = currency.Coin(assignerInfo.TotalLimit * floatToBalance)
	if newTotalLimit > conf.MaxTotalFreeAllocation {
		return "", common.NewErrorf("add_free_storage_assigner",
			"total tokens limit %d exceeds maximum permitted: %d", newTotalLimit, conf.MaxTotalFreeAllocation)
	}

	var newIndividualLimit = currency.Coin(assignerInfo.IndividualLimit * floatToBalance)
	if newIndividualLimit > conf.MaxIndividualFreeAllocation {
		return "", common.NewErrorf("add_free_storage_assigner",
			"individual allocation token limit %d exceeds maximum permitted: %d", newIndividualLimit, conf.MaxIndividualFreeAllocation)
	}

	assigner, err := ssc.getFreeStorageAssigner(assignerInfo.Name, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError("add_free_storage_assigner", err.Error())
	}
	if err == util.ErrValueNotPresent || assigner == nil {
		assigner = &freeStorageAssigner{
			ClientId: assignerInfo.Name,
		}
	}
	assigner.PublicKey = assignerInfo.PublicKey
	assigner.TotalLimit = newTotalLimit
	assigner.IndividualLimit = newIndividualLimit
	err = assigner.save(ssc.ID, balances)
	if err != nil {
		return "", common.NewErrorf("add_free_storage_assigner", "error saving new assigner: %v", err)
	}

	return "", nil
}

func verifyFreeAllocationRequest(
	frm freeStorageMarker,
	publicKey string,
	balances cstate.StateContextI,
) (bool, error) {
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
	signatureScheme := balances.GetSignatureScheme()
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
	var inputObj freeStorageAllocationInput
	if err := inputObj.decode(input); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"unmarshal input: %v", err)
	}

	var marker freeStorageMarker
	if err := marker.decode([]byte(inputObj.Marker)); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"unmarshal request: %v", err)
	}

	var conf *Config
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"can't get config: %v", err)
	}

	if txn.ClientID != marker.Recipient {
		return "", common.NewErrorf("free_allocation_failed",
			"marker can be used only by its recipient")
	}

	assigner, err := ssc.getFreeStorageAssigner(marker.Assigner, balances)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"error getting assigner details: %v", err)
	}

	if err := assigner.validate(marker, txn.CreationDate, txn.Value, balances); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"marker verification failed: %v", err)
	}

	var request = newAllocationRequest{
		DataShards:      conf.FreeAllocationSettings.DataShards,
		ParityShards:    conf.FreeAllocationSettings.ParityShards,
		Size:            conf.FreeAllocationSettings.Size,
		Expiration:      common.Timestamp(common.ToTime(txn.CreationDate).Add(conf.FreeAllocationSettings.Duration).Unix()),
		Owner:           marker.Recipient,
		OwnerPublicKey:  inputObj.RecipientPublicKey,
		ReadPriceRange:  conf.FreeAllocationSettings.ReadPriceRange,
		WritePriceRange: conf.FreeAllocationSettings.WritePriceRange,
		Blobbers:        inputObj.Blobbers,
	}

	arBytes, err := request.encode()
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"marshal request: %v", err)
	}

	assigner.CurrentRedeemed += txn.Value
	fTxnVal, err := txn.Value.Float64()
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed", "converting transaction value to float: %v", err)
	}
	readPoolTokens, err := currency.Float64ToCoin(fTxnVal * conf.FreeAllocationSettings.ReadPoolFraction)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed", "converting read pool tokens to Coin: %v", err)
	}
	txn.Value, err = currency.MinusCoin(txn.Value, readPoolTokens)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"subtracting read pool token from transaction value: %v", err)
	}

	resp, err := ssc.newAllocationRequestInternal(txn, arBytes, conf, true, balances, nil)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed", "creating new allocation: %v", err)
	}

	var sa StorageAllocation
	if err := sa.Decode([]byte(resp)); err != nil {
		return "", common.NewErrorf("free_allocation_failed", "unmarshalling allocation: %v", err)
	}

	assigner.RedeemedTimestamps = append(assigner.RedeemedTimestamps, marker.Timestamp)
	if err := assigner.save(ssc.ID, balances); err != nil {
		return "", common.NewErrorf("free_allocation_failed", "assigner save failed: %v", err)
	}

	var lr = readPoolLockRequest{
		TargetId:   marker.Recipient,
		MintTokens: true,
	}
	input, err = json.Marshal(lr)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed", "marshal read lock request: %v", err)
	}

	txn.Value = readPoolTokens
	_, err = ssc.readPoolLock(txn, input, balances)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed", "locking tokens in read pool: %v", err)
	}

	return resp, err
}

func (ssc *StorageSmartContract) updateFreeStorageRequest(
	txn *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
	var err error
	var inputObj freeStorageUpgradeInput
	if err := json.Unmarshal(input, &inputObj); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"unmarshal input: %v", err)
	}

	var marker freeStorageMarker
	if err := marker.decode([]byte(inputObj.Marker)); err != nil {
		return "", common.NewErrorf("update_free_storage_request",
			"unmarshal request: %v", err)
	}

	var conf *Config
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("update_free_storage_request",
			"can't get config: %v", err)
	}

	assigner, err := ssc.getFreeStorageAssigner(marker.Assigner, balances)
	if err != nil {
		return "", common.NewErrorf("update_free_storage_request",
			"error getting assigner details: %v", err)
	}

	if err := assigner.validate(marker, txn.CreationDate, txn.Value, balances); err != nil {
		return "", common.NewErrorf("update_free_storage_request",
			"marker verification failed: %v", err)
	}

	var request = updateAllocationRequest{
		ID:         inputObj.AllocationId,
		OwnerID:    marker.Recipient,
		Size:       conf.FreeAllocationSettings.Size,
		Expiration: common.Timestamp(conf.FreeAllocationSettings.Duration.Seconds()),
	}
	input, err = json.Marshal(request)
	if err != nil {
		return "", common.NewErrorf("update_free_storage_request",
			"marshal marker: %v", err)
	}

	resp, err := ssc.updateAllocationRequestInternal(txn, input, conf, true, balances)
	if err != nil {
		return "", common.NewErrorf("update_free_storage_request", err.Error())
	}

	assigner.CurrentRedeemed += txn.Value
	assigner.RedeemedTimestamps = append(assigner.RedeemedTimestamps, marker.Timestamp)
	if err := assigner.save(ssc.ID, balances); err != nil {
		return "", common.NewErrorf("update_free_storage_request", "assigner save failed: %v", err)
	}

	return resp, nil
}

func (ssc *StorageSmartContract) getFreeStorageAssigner(
	clientID datastore.Key,
	balances cstate.StateContextI,
) (*freeStorageAssigner, error) {
	fsa := new(freeStorageAssigner)
	err := balances.GetTrieNode(freeStorageAssignerKey(ssc.ID, clientID), fsa)
	if err != nil {
		return nil, err
	}

	return fsa, nil
}
