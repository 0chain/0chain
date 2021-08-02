package magmasc

import (
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/0chain/bandwidth_marketplace/code/core/errors"
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"
	"github.com/0chain/bandwidth_marketplace/code/core/time"
	magma "github.com/magma/augmented-networks/accounting/protos"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/mock"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

const (
	// One million (Mega) is a unit prefix in metric systems
	// of units denoting a factor of one million (1e6 or 1_000_000).
	million = 1e6
)

type (
	// mockStateContext implements mocked chain state context interface.
	mockStateContext struct {
		mocks.StateContextI
		sync.Mutex
		store map[datastore.Key]util.Serializable
	}

	// mockSmartContract implements mocked smart contract interface.
	mockSmartContract struct {
		mocks.SmartContractInterface
		ID string
		SC *MagmaSmartContract
	}

	// mockInvalidJson implements mocked util.Serializable interface for invalid json.
	mockInvalidJson struct{ ID datastore.Key }
)

// Decode implements util.Serializable interface.
func (m *mockInvalidJson) Decode([]byte) error {
	return errDecodeData
}

// Encode implements util.Serializable interface.
func (m *mockInvalidJson) Encode() []byte {
	return []byte(":")
}

func mockAcknowledgment() *bmp.Acknowledgment {
	return &bmp.Acknowledgment{
		SessionID:     "session_id",
		AccessPointID: "access_point_id",
		Billing:       mockBilling(),
		Consumer:      mockConsumer(),
		Provider:      mockProvider(),
	}
}

func mockActiveAcknowledgments(size int) *ActiveAcknowledgments {
	list := &ActiveAcknowledgments{Nodes: make(map[string]*bmp.Acknowledgment, size)}
	for i := 0; i < size; i++ {
		id := strconv.Itoa(i)
		ackn := mockAcknowledgment()
		ackn.SessionID += id
		ackn.AccessPointID += id
		ackn.Billing.DataUsage.SessionID += id
		ackn.Provider.ID += id
		ackn.Provider.ExtID += id
		ackn.Provider.Host += id
		ackn.Consumer.ID += id
		ackn.Consumer.ExtID += id
		ackn.Consumer.Host += id
		list.Nodes[ackn.SessionID] = ackn
	}

	return list
}

func mockBilling() bmp.Billing {
	return bmp.Billing{
		DataUsage: mockDataUsage(),
	}
}

func mockConsumer() *bmp.Consumer {
	return &bmp.Consumer{
		ID:    "consumer_id",
		ExtID: "ext_id",
		Host:  "localhost:8010",
	}
}

func mockConsumers() *Consumers {
	list := &Consumers{}
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		list.Nodes.add(&bmp.Consumer{
			ID:    "consumer_id" + id,
			ExtID: "ext_id" + id,
			Host:  "localhost:8010" + id,
		})
	}

	return list
}

func mockDataUsage() bmp.DataUsage {
	return bmp.DataUsage{
		DownloadBytes: 3 * million,
		UploadBytes:   2 * million,
		SessionID:     "session_id",
		SessionTime:   1 * 60, // 1 minute
	}
}

func mockSmartContractI() *mockSmartContract {
	msc := mockMagmaSmartContract()

	argBlob := mock.AnythingOfType("[]uint8")
	argSci := mock.AnythingOfType("*magmasc.mockStateContext")
	argStr := mock.AnythingOfType("string")
	argTxn := mock.AnythingOfType("*transaction.Transaction")

	smartContract := mockSmartContract{ID: msc.ID, SC: msc}
	smartContract.On("Execute", argTxn, argStr, argBlob, argSci).Return(
		func(txn *tx.Transaction, call string, blob []byte, sci chain.StateContextI) string {
			if _, err := smartContract.SC.Execute(txn, call, blob, sci); errors.Is(err, errInvalidFuncName) {
				return ""
			}
			return call
		},
		func(txn *tx.Transaction, call string, blob []byte, sci chain.StateContextI) error {
			if _, err := smartContract.SC.Execute(txn, call, blob, sci); errors.Is(err, errInvalidFuncName) {
				return err
			}
			return nil
		},
	)

	return &smartContract
}

func mockMagmaSmartContract() *MagmaSmartContract {
	msc := MagmaSmartContract{SmartContract: sci.NewSC("sc_id")}

	msc.SmartContractExecutionStats[consumerRegister] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+consumerRegister, nil)
	msc.SmartContractExecutionStats[providerRegister] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+providerRegister, nil)

	return &msc
}

func mockProvider() *bmp.Provider {
	return &bmp.Provider{
		ID:    "provider_id",
		ExtID: "ext_id",
		Host:  "localhost:8020",
		Terms: mockProviderTerms(),
	}
}

func mockProviders() *Providers {
	list := &Providers{}
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		list.Nodes.add(&bmp.Provider{
			ID:    "provider_id" + id,
			ExtID: "ext_id" + id,
			Host:  "localhost:8020" + id,
			Terms: mockProviderTerms(),
		})
	}

	return list
}

func mockProviderTerms() bmp.ProviderTerms {
	return bmp.ProviderTerms{
		Price:           0.1,
		PriceAutoUpdate: 0.001,
		MinCost:         0.5,
		Volume:          0,
		QoS:             mockQoS(),
		QoSAutoUpdate: bmp.AutoUpdateQoS{
			DownloadMbps: 0.001,
			UploadMbps:   0.001,
		},
		ProlongDuration: 1 * 60 * 60,                // 1 hour
		ExpiredAt:       time.Now() + (1 * 60 * 60), // 1 hour from now
	}
}

func mockStateContextI() *mockStateContext {
	argStr := mock.AnythingOfType("string")
	stateContext := mockStateContext{store: make(map[datastore.Key]util.Serializable)}
	funcInsertID := func(id datastore.Key, val util.Serializable) datastore.Key {
		if !strings.Contains(id, "cannot_insert_id") {
			stateContext.Lock()
			stateContext.store[id] = val
			stateContext.Unlock()
		}
		return ""
	}
	errFuncInsertID := func(id datastore.Key, _ util.Serializable) error {
		if strings.Contains(id, "cannot_insert_id") {
			return errors.New(errCodeInternal, errTextUnexpected)
		}
		return nil
	}
	funcInsertList := func(id datastore.Key, val util.Serializable) datastore.Key {
		json := string(val.Encode())
		if !strings.Contains(json, "cannot_insert_list") {
			stateContext.Lock()
			stateContext.store[id] = val
			stateContext.Unlock()
		}
		return ""
	}
	errFuncInsertList := func(_ datastore.Key, val util.Serializable) error {
		json := string(val.Encode())
		if strings.Contains(json, "cannot_insert_list") {
			return errors.New(errCodeInternal, errTextUnexpected)
		}
		return nil
	}

	msc := mockMagmaSmartContract()

	ackn := mockAcknowledgment()
	ackn.SessionID = "cannot_insert_id"
	stateContext.store[nodeUID(msc.ID, ackn.SessionID, acknowledgment)] = ackn

	stateContext.On("AddTransfer", mock.AnythingOfType("*state.Transfer")).Return(
		func(transfer *state.Transfer) error {
			if transfer.ClientID == "not_present_id" || transfer.ToClientID == "not_present_id" {
				return util.ErrValueNotPresent
			}
			return nil
		},
	)
	stateContext.On("DeleteTrieNode", argStr).Return(
		func(id datastore.Key) datastore.Key {
			stateContext.Lock()
			defer stateContext.Unlock()
			if _, ok := stateContext.store[id]; ok {
				return id
			}
			return ""
		},
		func(id datastore.Key) error {
			stateContext.Lock()
			defer stateContext.Unlock()
			defer delete(stateContext.store, id)
			if _, ok := stateContext.store[id]; ok {
				return nil
			}
			return util.ErrValueNotPresent
		},
	)
	stateContext.On("GetClientBalance", argStr).Return(
		func(id datastore.Key) state.Balance {
			if id == "consumer_id" {
				return 1000 * 1e9 // 1000 * 1e9 units equal to one thousand coins
			}
			return 0
		},
		func(id datastore.Key) error {
			if id == "" {
				return util.ErrNodeNotFound
			}
			if id == "not_present_id" {
				return util.ErrValueNotPresent
			}
			return nil
		},
	)
	stateContext.On("GetTransaction").Return(
		func() *tx.Transaction { return &tx.Transaction{ToClientID: msc.ID} },
	)
	stateContext.On("GetTrieNode", argStr).Return(
		func(id datastore.Key) util.Serializable {
			stateContext.Lock()
			defer stateContext.Unlock()
			if val, ok := stateContext.store[id]; ok {
				return val
			}
			return nil
		},
		func(id datastore.Key) error {
			if strings.Contains(id, "not_present_id") {
				return util.ErrValueNotPresent
			}
			if strings.Contains(id, "unexpected_id") {
				return errInternalUnexpected
			}
			stateContext.Lock()
			defer stateContext.Unlock()
			if _, ok := stateContext.store[id]; ok {
				return nil
			}
			return util.ErrValueNotPresent
		},
	)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Acknowledgment")).
		Return(funcInsertID, errFuncInsertID)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.ActiveAcknowledgments")).
		Return(funcInsertList, errFuncInsertList)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Billing")).
		Return(funcInsertID, errFuncInsertID)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Consumer")).
		Return(funcInsertID, errFuncInsertID)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Consumers")).
		Return(funcInsertList, errFuncInsertList)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.mockInvalidJson")).
		Return(funcInsertID, errFuncInsertID)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Provider")).
		Return(funcInsertID, errFuncInsertID)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Providers")).
		Return(funcInsertList, errFuncInsertList)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.tokenPool")).
		Return(funcInsertID, errFuncInsertID)

	nodeInvalid := mockInvalidJson{ID: "invalid_json_id"}
	if _, err := stateContext.InsertTrieNode(nodeInvalid.ID, &nodeInvalid); err != nil {
		log.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	return &stateContext
}

func mockTokenPool() *tokenPool {
	pool := tokenPool{
		PayerID: "payer_id",
		PayeeID: "payee_id",
	}

	pool.ID = "session_id"
	pool.Balance = 1000

	return &pool
}

func mockQoS() magma.QoS {
	return magma.QoS{
		DownloadMbps: 5.4321,
		UploadMbps:   1.2345,
	}
}
