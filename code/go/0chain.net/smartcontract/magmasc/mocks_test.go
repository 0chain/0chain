package magmasc

import (
	"log"
	"strconv"
	"strings"
	"sync"

	magma "github.com/magma/augmented-networks/accounting/protos"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/mock"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// mockStateContext implements mocked chain state context interface.
	mockStateContext struct {
		mocks.StateContextI
		store map[datastore.Key]util.Serializable
		mutex sync.RWMutex
	}

	// mockSmartContract implements mocked mocked smart contract interface.
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

func mockAcknowledgment() *Acknowledgment {
	return &Acknowledgment{
		SessionID:     "session_id",
		AccessPointID: "access_point_id",
		Consumer:      mockConsumer(),
		Provider:      mockProvider(),
	}
}

func mockBilling() *Billing {
	bill := Billing{
		DataUsage: mockDataUsage(),
		SessionID: "session_id",
	}

	return &bill
}

func mockConsumer() *Consumer {
	return &Consumer{
		ID:    "consumer_id",
		ExtID: "ext_id",
		Host:  "localhost:8010",
	}
}

func mockConsumers() *Consumers {
	list := &Consumers{Nodes: &consumersSorted{}}
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		list.Nodes.add(&Consumer{
			ID:    "consumer_id" + id,
			ExtID: "ext_id" + id,
			Host:  "localhost:801" + id,
		})
	}

	return list
}

func mockDataUsage() *DataUsage {
	return &DataUsage{
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
			if _, err := smartContract.SC.Execute(txn, call, blob, sci); errIs(err, errInvalidFuncName) {
				return ""
			}
			return call
		},
		func(txn *tx.Transaction, call string, blob []byte, sci chain.StateContextI) error {
			if _, err := smartContract.SC.Execute(txn, call, blob, sci); errIs(err, errInvalidFuncName) {
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

func mockProvider() *Provider {
	return &Provider{
		ID:    "provider_id",
		ExtID: "ext_id",
		Host:  "localhost:8020",
		Terms: mockProviderTerms(),
	}
}

func mockProviders() *Providers {
	list := &Providers{Nodes: &providersSorted{}}
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		list.Nodes.add(&Provider{
			ID:    "provider_id" + id,
			ExtID: "ext_id" + id,
			Host:  "localhost:802" + id,
			Terms: ProviderTerms{},
		})
	}

	return list
}

func mockProviderTerms() ProviderTerms {
	return ProviderTerms{
		Terms: mockTerms(),
		QoS:   mockQoS(),
	}
}

func mockStateContextI() *mockStateContext {
	msc := mockMagmaSmartContract()
	argStr := mock.AnythingOfType("string")

	stateContext := mockStateContext{store: make(map[datastore.Key]util.Serializable)}

	ackn := mockAcknowledgment()
	ackn.SessionID = "cannot_insert_id"
	stateContext.store[ackn.uid(msc.ID)] = ackn

	bill := mockBilling()
	bill.SessionID = "cannot_insert_id"
	stateContext.store[bill.uid(msc.ID)] = bill

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
			if _, ok := stateContext.store[id]; ok {
				return id
			}
			return ""
		},
		func(id datastore.Key) error {
			if _, ok := stateContext.store[id]; ok {
				return nil
			}
			return util.ErrValueNotPresent
		},
	)
	stateContext.On("GetClientBalance", argStr).Return(
		func(id datastore.Key) state.Balance {
			if id == "consumer_id" {
				return 1000000000000 // 1000 * 1e9 units equal to one thousand coins
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
		func() *tx.Transaction {
			return &tx.Transaction{ToClientID: msc.ID}
		},
	)
	stateContext.On("GetTrieNode", argStr).Return(
		func(id datastore.Key) util.Serializable {
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
			if _, ok := stateContext.store[id]; ok {
				return nil
			}
			return util.ErrNodeNotFound
		},
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Acknowledgment")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			if !strings.Contains(id, "cannot_insert_id") {
				stateContext.store[id] = val
			}
			return ""
		},
		func(id datastore.Key, _ util.Serializable) error {
			if strings.Contains(id, "cannot_insert_id") {
				return errNew(errCodeInternal, errTextUnexpected)
			}
			return nil
		},
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Billing")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			if !strings.Contains(id, "cannot_insert_id") {
				stateContext.store[id] = val
			}
			return ""
		},
		func(id datastore.Key, _ util.Serializable) error {
			if strings.Contains(id, "cannot_insert_id") {
				return errNew(errCodeInternal, errTextUnexpected)
			}
			return nil
		},
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Consumer")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			if !strings.Contains(id, "cannot_insert_id") {
				stateContext.store[id] = val
			}
			return ""
		},
		func(id datastore.Key, _ util.Serializable) error {
			if strings.Contains(id, "cannot_insert_id") {
				return errNew(errCodeInternal, errTextUnexpected)
			}
			return nil
		},
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Consumers")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			json := string(val.Encode())
			if !strings.Contains(json, "cannot_insert_list") {
				stateContext.store[id] = val
			}
			return ""
		},
		func(_ datastore.Key, val util.Serializable) error {
			json := string(val.Encode())
			if strings.Contains(json, "cannot_insert_list") {
				return errNew(errCodeInternal, errTextUnexpected)
			}
			return nil
		},
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.mockInvalidJson")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			stateContext.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Provider")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			if !strings.Contains(id, "cannot_insert_id") {
				stateContext.store[id] = val
			}
			return ""
		},
		func(id datastore.Key, _ util.Serializable) error {
			if strings.Contains(id, "cannot_insert_id") {
				return errNew(errCodeInternal, errTextUnexpected)
			}
			return nil
		},
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Providers")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			json := string(val.Encode())
			if !strings.Contains(json, "cannot_insert_list") {
				stateContext.store[id] = val
			}
			return ""
		},
		func(_ datastore.Key, val util.Serializable) error {
			json := string(val.Encode())
			if strings.Contains(json, "cannot_insert_list") {
				return errNew(errCodeInternal, errTextUnexpected)
			}
			return nil
		},
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.tokenPool")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			if !strings.Contains(id, "cannot_insert_id") {
				stateContext.store[id] = val
			}
			return ""
		},
		func(id datastore.Key, _ util.Serializable) error {
			if strings.Contains(id, "cannot_insert_id") {
				return errNew(errCodeInternal, errTextUnexpected)
			}
			return nil
		},
	)

	nodeInvalid := mockInvalidJson{ID: "invalid_json_id"}
	if _, err := stateContext.InsertTrieNode(nodeInvalid.ID, &nodeInvalid); err != nil {
		log.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	return &stateContext
}

func mockTerms() Terms {
	return Terms{
		Price:           0.1,
		MinCost:         0.5,
		Volume:          0,
		AutoUpdatePrice: 0.001,
		AutoUpdateQoS: AutoUpdateQoS{
			DownloadMbps: 0.001,
			UploadMbps:   0.001,
		},
		ProlongDuration: 1 * 60 * 60,                  // 1 hour
		ExpiredAt:       common.Now() + (1 * 60 * 60), // 1 hour from now
	}
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
