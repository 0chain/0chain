package storagesc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"0chain.net/chaincore/smartcontractinterface"
	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

const (
	floatToBalance = 10 * 1000 * 1000 * 1000
)

//msgp:ignore freeStorageAllocationInput newFreeStorageAssignerInfo concurrentReader
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
	return encryption.Hash(sscKey + ":freestorageredeemed:" + clientId)
	//return "a"
}

type freeStorageAssigner struct {
	IndividualLimit    uint64  `json:"individual_limit" msg:"i"`
	TotalLimit         uint64  `json:"total_limit" msg:"t"`
	CurrentRedeemed    uint64  `json:"current_redeemed" msg:"r"`
	RedeemedTimestamps []int64 `json:"redeemed_timestamps" msg:"rt"`
	ClientId           string  `json:"client_id" msg:"c"`
	PublicKey          string  `json:"public_key" msg:"p"`
}

func (fsa *freeStorageAssigner) Copy() *freeStorageAssigner {
	f := &freeStorageAssigner{
		IndividualLimit:    fsa.IndividualLimit,
		TotalLimit:         fsa.TotalLimit,
		CurrentRedeemed:    fsa.CurrentRedeemed,
		ClientId:           fsa.ClientId,
		PublicKey:          fsa.PublicKey,
		RedeemedTimestamps: make([]int64, len(fsa.RedeemedTimestamps)),
	}
	copy(f.RedeemedTimestamps[:], fsa.RedeemedTimestamps[:])

	return f
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

// TODO test that we really send some value here
func (fsa *freeStorageAssigner) validate(
	marker freeStorageMarker,
	now common.Timestamp,
	value currency.Coin,
	balances cstate.StateContextI,
) error {
	if marker.Timestamp > now {
		return fmt.Errorf("marker timestamped in the future: %v", marker.Timestamp)
	}

	verified, err := verifyFreeAllocationRequest(marker, fsa.PublicKey, balances)
	if err != nil {
		return err
	}
	if !verified {
		return fmt.Errorf("failed to verify signature")
	}

	newTotal, err := currency.AddCoin(currency.Coin(fsa.CurrentRedeemed), value)
	if err != nil {
		return err
	}

	if newTotal > currency.Coin(fsa.TotalLimit) {
		return fmt.Errorf("%d exceeded total permitted free storage limit %d", newTotal, fsa.TotalLimit)
	}

	if value > currency.Coin(fsa.IndividualLimit) {
		return fmt.Errorf("%d exceeded permitted free storage  %d", value, fsa.IndividualLimit)
	}

	for _, timestamp := range fsa.RedeemedTimestamps {
		if int64(marker.Timestamp) == timestamp {
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

	newTotalLimit, err := currency.Float64ToCoin(assignerInfo.TotalLimit * floatToBalance)
	if err != nil {
		return "", common.NewErrorf("add_free_storage_assigner", "can't convert total limit to coin: %v", err)
	}

	if newTotalLimit > conf.MaxTotalFreeAllocation {
		return "", common.NewErrorf("add_free_storage_assigner",
			"total tokens limit %d exceeds maximum permitted: %d", newTotalLimit, conf.MaxTotalFreeAllocation)
	}

	newIndividualLimit, err := currency.Float64ToCoin(assignerInfo.IndividualLimit * floatToBalance)
	if err != nil {
		return "", common.NewErrorf("add_free_storage_assigner", "can't convert individual limit to coin: %v", err)
	}

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
	assigner.TotalLimit = uint64(newTotalLimit)
	assigner.IndividualLimit = uint64(newIndividualLimit)
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
	timings map[string]time.Duration,
) (string, error) {
	m := Timings{timings: timings, start: common.ToTime(common.Now())}
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

	if txn.ClientID != marker.Recipient {
		return "", common.NewErrorf("free_allocation_failed",
			"marker can be used only by its recipient")
	}

	coin, err := currency.ParseZCN(marker.FreeTokens)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"marker verification failed: %v", err)
	}
	m.tick("prepare")

	conf, err := ssc.getConfig(balances, true)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed", "can't get config: %v", err)
	}

	m.tick("load config")
	totalMint, err := currency.ParseZCN(marker.FreeTokens)
	if err != nil {
		return "", err
	}
	readPoolTokens, err := currency.Float64ToCoin(float64(totalMint) * conf.FreeAllocationSettings.ReadPoolFraction)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed", "converting read pool tokens to Coin: %v", err)
	}
	writePoolTokens, err := currency.MinusCoin(totalMint, readPoolTokens)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"subtracting read pool token from transaction value: %v", err)
	}
	m.tick("calculate tokens")

	var (
		assigner *freeStorageAssigner
		rp       *readPool
		resp     string
		rc       = concurrentReader{}
	)
	rc.add(func(state cstate.StateContextI) error {
		var err error
		assigner, err = ssc.getFreeStorageAssigner(marker.Assigner, balances)
		if err != nil {
			return common.NewErrorf("free_allocation_failed", "error getting assigner details: %v", err)
		}

		//todo query sharder on 0box to get the price of allocation
		if err := assigner.validate(marker, txn.CreationDate, coin, balances); err != nil {
			return common.NewErrorf("free_allocation_failed", "marker verification failed: %v", err)
		}
		return nil
	})

	rc.add(func(state cstate.StateContextI) error {
		var err error
		rp, err = ssc.getReadPool(marker.Recipient, balances)
		if err != nil {
			if err != util.ErrValueNotPresent {
				return common.NewError("read_pool_lock_failed", err.Error())
			} else {
				rp = new(readPool)
			}
		}
		return nil
	})

	rc.add(func(state cstate.StateContextI) error {
		var request = newAllocationRequest{
			DataShards:      conf.FreeAllocationSettings.DataShards,
			ParityShards:    conf.FreeAllocationSettings.ParityShards,
			Size:            conf.FreeAllocationSettings.Size,
			Expiration:      common.Timestamp(common.ToTime(txn.CreationDate).Add(conf.TimeUnit).Unix()),
			Owner:           marker.Recipient,
			OwnerPublicKey:  inputObj.RecipientPublicKey,
			ReadPriceRange:  conf.FreeAllocationSettings.ReadPriceRange,
			WritePriceRange: conf.FreeAllocationSettings.WritePriceRange,
			Blobbers:        inputObj.Blobbers,
		}

		sa, err := ssc.newAllocationRequestInternal(txn, &request, conf, writePoolTokens, balances, nil)
		if err != nil {
			return common.NewErrorf("free_allocation_failed", "creating new allocation: %v", err)
		}

		resp = string(sa.Encode())
		return nil
	})

	if err := rc.do(balances); err != nil {
		return "", err
	}
	m.tick("concurrent run")

	txn.Value = readPoolTokens
	_, err = ssc.readPoolLockInternal(txn, rp, readPoolTokens, true, marker.Recipient, balances)
	if err != nil {
		return "", common.NewErrorf("free_allocation_failed", "locking tokens in read pool: %v", err)
	}

	m.tick("lock read pool")

	free, err := currency.ParseZCN(marker.FreeTokens)
	if err != nil {
		return "", err
	}
	newRedeemed, err := currency.AddCoin(currency.Coin(assigner.CurrentRedeemed), free)
	if err != nil {
		return "", err
	}
	assigner.CurrentRedeemed = uint64(newRedeemed)
	assigner.RedeemedTimestamps = append(assigner.RedeemedTimestamps, int64(marker.Timestamp))
	if err := assigner.save(ssc.ID, balances); err != nil {
		return "", common.NewErrorf("free_allocation_failed", "assigner Save failed: %v", err)
	}
	m.tick("save assigner")

	return resp, err
}

type concurrentReader struct {
	funcs []func(cstate.StateContextI) error
}

func (rc *concurrentReader) add(f func(cstate.StateContextI) error) {
	rc.funcs = append(rc.funcs, f)
}

func (rc *concurrentReader) do(state cstate.StateContextI) error {
	wg := sync.WaitGroup{}
	errs := make([]error, len(rc.funcs))
	for i, f := range rc.funcs {
		wg.Add(1)
		go func(idx int, f func(s cstate.StateContextI) error) {
			defer wg.Done()
			errs[idx] = f(state)
		}(i, f)
	}
	wg.Wait()
	for _, err := range errs {
		// return the first encountered error to ensure the SC is in a consistent state
		if err != nil {
			return err
		}
	}

	return nil
}

func (ssc *StorageSmartContract) updateFreeStorageRequest(
	txn *transaction.Transaction,
	input []byte,
	balances cstate.StateContextI,
) (string, error) {
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

	var assigner *freeStorageAssigner
	//var conf *Config
	assignErrC := make(chan error, 1)
	cr := concurrentReader{}
	cr.add(func(state cstate.StateContextI) error {
		var err error
		assigner, err = ssc.getFreeStorageAssigner(marker.Assigner, balances)
		if err != nil {
			return common.NewErrorf("update_free_storage_request",
				"error getting assigner details: %v", err)
		}

		ac := assigner.Copy()
		go func() {
			if err := ac.validate(marker, txn.CreationDate, txn.Value, balances); err != nil {
				assignErrC <- common.NewErrorf("update_free_storage_request",
					"marker verification failed: %v", err)
			} else {
				assignErrC <- nil
			}
		}()
		return nil
	})

	var resp string
	cr.add(func(state cstate.StateContextI) error {
		alloc, bil, conf, err := ssc.preloadUpdateAllocation(inputObj.AllocationId, balances)
		if err != nil {
			return err
		}

		var request = updateAllocationRequest{
			ID:         inputObj.AllocationId,
			OwnerID:    marker.Recipient,
			Size:       conf.FreeAllocationSettings.Size,
			Expiration: common.Timestamp(conf.TimeUnit.Seconds()),
		}

		resp, err = ssc.updateAllocationRequestInternal(txn, request, alloc, bil, conf, balances)
		if err != nil {
			return common.NewErrorf("update_free_storage_request", err.Error())
		}
		return nil
	})

	if err := cr.do(balances); err != nil {
		return "", err
	}

	newRedeemed, err := currency.AddCoin(currency.Coin(assigner.CurrentRedeemed), txn.Value)
	if err != nil {
		return "", common.NewErrorf("update_free_storage_request",
			"can't add redeemed tokens: %v", err)
	}
	assigner.CurrentRedeemed = uint64(newRedeemed)
	assigner.RedeemedTimestamps = append(assigner.RedeemedTimestamps, int64(marker.Timestamp))

	if err := assigner.save(ssc.ID, balances); err != nil {
		return "", common.NewErrorf("update_free_storage_request", "assigner Save failed: %v", err)
	}

	err = <-assignErrC
	if err != nil {
		return "", err
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
