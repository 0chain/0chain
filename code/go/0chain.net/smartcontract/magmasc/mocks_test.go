package magmasc

import (
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/magmasc/pb"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	store "0chain.net/core/ememorystore"
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
		store map[string]util.Serializable
	}

	// mockSmartContract implements mocked smart contract interface.
	mockSmartContract struct {
		mocks.SmartContractInterface
		ID string
		SC *MagmaSmartContract
		sync.Mutex
	}

	// mockInvalidJson implements mocked util.Serializable interface for invalid json.
	mockInvalidJson struct{ ID string }
)

var (
	mutexMockMSC sync.Mutex
)

// Decode implements util.Serializable interface.
func (m *mockInvalidJson) Decode([]byte) error {
	return zmc.ErrDecodeData
}

// Encode implements util.Serializable interface.
func (m *mockInvalidJson) Encode() []byte {
	return []byte(":")
}

func mockSession() *zmc.Session {
	now := time.Now().Format(time.RFC3339Nano)
	prov := mockProvider()

	return &zmc.Session{
		SessionID:   "id:session:" + now,
		AccessPoint: mockAccessPoint(prov.ExtId),
		Billing: zmc.Billing{
			DataMarker: &zmc.DataMarker{
				DataMarker: &pb.DataMarker{
					DataUsage: &pb.DataUsage{
						DownloadBytes: 3 * million,
						UploadBytes:   2 * million,
						SessionId:     "id:session:" + now,
						SessionTime:   1 * 60, // 1 minute
					},
				},
			},
		},
		Consumer: mockConsumer(),
		Provider: prov,
	}
}

func mockConsumer() *zmc.Consumer {
	now := time.Now()
	bin, _ := now.MarshalBinary()
	sum := sha3.Sum256(bin)
	fix := hex.EncodeToString(sum[:])
	return &zmc.Consumer{
		ID:    "id:consumer:" + fix,
		ExtID: "id:consumer:external:" + fix,
		Host:  "host.consumer.local:" + fix,
	}
}

func mockConsumers() *Consumers {
	list := &Consumers{}
	for i := 0; i < 10; i++ {
		list.put(mockConsumer())
	}

	return list
}

func mockAccessPoint(providerID string) *zmc.AccessPoint {
	now := time.Now()
	bin, _ := now.MarshalBinary()
	sum := sha3.Sum256(bin)
	fix := hex.EncodeToString(sum[:])
	return &zmc.AccessPoint{
		AccessPoint: &pb.AccessPoint{
			Id: "id:ap:" + fix,
			Terms: &pb.Terms{
				Price:           0.1,
				PriceAutoUpdate: 0.001,
				MinCost:         0.5,
				Volume:          0,
				Qos: &pb.QoS{
					DownloadMbps: 5.4321,
					UploadMbps:   1.2345,
				},
				QosAutoUpdate: &pb.QoSAutoUpdate{
					DownloadMbps: 0.001,
					UploadMbps:   0.001,
				},
				ProlongDuration: durationpb.New(time.Hour),
				ExpiredAt:       timestamppb.New(time.Now().Add(time.Hour)),
			},
			ProviderExtId: providerID,
		},
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
			if _, err := smartContract.SC.Execute(txn, call, blob, sci); errors.Is(err, zmc.ErrInvalidFuncName) {
				return ""
			}
			return call
		},
		func(txn *tx.Transaction, call string, blob []byte, sci chain.StateContextI) error {
			if _, err := smartContract.SC.Execute(txn, call, blob, sci); errors.Is(err, zmc.ErrInvalidFuncName) {
				return err
			}
			return nil
		},
	)

	return &smartContract
}

func mockMagmaSmartContract() *MagmaSmartContract {
	mutexMockMSC.Lock()
	defer mutexMockMSC.Unlock()

	const prefix = "test."
	msc := &MagmaSmartContract{SmartContract: sci.NewSC(zmc.Address)}
	path := filepath.Join(os.TempDir(), rootPath, prefix+time.Now().Format(time.RFC3339Nano))
	err := os.MkdirAll(path, 0755)
	if err != nil {
		panic(err)
	}

	msc.db, err = store.CreateDB(path)
	if err != nil {
		panic(err)
	}
	store.AddPool(storeName, msc.db)

	file, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	sep := string(os.PathSeparator)
	dirs := strings.Split(file, sep)

	file = filepath.Join(sep, filepath.Join(dirs[:len(dirs)-5]...), "config", "sc.yaml")
	if err = config.SmartContractConfig.ReadConfigFile(file); err != nil {
		config.SetupSmartContractConfig()
	}
	msc.cfg = config.SmartContractConfig.Sub("smart_contracts." + Name)

	msc.SmartContractExecutionStats[zmc.ConsumerRegisterFuncName] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+zmc.ConsumerRegisterFuncName, nil)
	msc.SmartContractExecutionStats[zmc.ProviderRegisterFuncName] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+zmc.ProviderRegisterFuncName, nil)
	msc.SmartContractExecutionStats[zmc.AccessPointRegisterFuncName] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+zmc.AccessPointRegisterFuncName, nil)

	return msc
}

func mockProvider() *zmc.Provider {
	now := time.Now()
	bin, _ := now.MarshalBinary()
	sum := sha3.Sum256(bin)
	fix := hex.EncodeToString(sum[:])
	return &zmc.Provider{
		Provider: &pb.Provider{
			Id:    "id:provider:" + fix,
			ExtId: "id:provider:external:" + fix,
			Host:  "host.provider.local:" + fix,
		},
	}
}

func mockProviders() *Providers {
	list := &Providers{}
	for i := 0; i < 10; i++ {
		list.put(mockProvider())
	}

	return list
}

func mockStateContextI() *mockStateContext {
	argStr := mock.AnythingOfType("string")
	stateContext := mockStateContext{store: make(map[string]util.Serializable)}
	funcInsertID := func(id string, val util.Serializable) string {
		if !strings.Contains(id, "cannot_insert_id") {
			stateContext.Lock()
			stateContext.store[id] = val
			stateContext.Unlock()
		}
		return ""
	}
	errFuncInsertID := func(id string, _ util.Serializable) error {
		if strings.Contains(id, "cannot_insert_id") {
			return errors.New(zmc.ErrCodeInternal, zmc.ErrTextUnexpected)
		}
		return nil
	}
	funcInsertList := func(id string, val util.Serializable) string {
		json := string(val.Encode())
		if !strings.Contains(json, "cannot_insert_list") {
			stateContext.Lock()
			stateContext.store[id] = val
			stateContext.Unlock()
		}
		return ""
	}
	errFuncInsertList := func(_ string, val util.Serializable) error {
		json := string(val.Encode())
		if strings.Contains(json, "cannot_insert_list") {
			return errors.New(zmc.ErrCodeInternal, zmc.ErrTextUnexpected)
		}
		return nil
	}

	funcDeleteID := func(id string) string {
		if !strings.Contains(id, "cannot_delete_id") {
			stateContext.Lock()
			delete(stateContext.store, id)
			stateContext.Unlock()
		}
		return ""
	}
	errFuncDeleteID := func(id string) error {
		if strings.Contains(id, "cannot_delete_id") {
			return errors.New(zmc.ErrCodeInternal, zmc.ErrTextUnexpected)
		}
		return nil
	}

	sess := mockSession()
	sess.SessionID = "cannot_insert_id"
	stateContext.store[nodeUID(zmc.Address, session, sess.SessionID)] = sess

	stateContext.On("AddTransfer", mock.AnythingOfType("*state.Transfer")).Return(
		func(transfer *state.Transfer) error {
			if transfer.ClientID == "not_present_id" || transfer.ToClientID == "not_present_id" {
				return util.ErrValueNotPresent
			}
			return nil
		},
	)
	stateContext.On("GetClientBalance", argStr).Return(
		func(id string) state.Balance {
			if strings.Contains(id, "id:consumer:") || strings.Contains(id, "id:provider:") || strings.Contains(id, "id:ap:") {
				return 10000 * 1e9 // 1000 * 1e9 units equal to ten thousand coins
			}
			return 0
		},
		func(id string) error {
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
			bin, _ := time.Now().MarshalBinary()
			hash := sha3.Sum256(bin)
			txn := tx.Transaction{ToClientID: zmc.Address}
			txn.Hash = hex.EncodeToString(hash[:])
			return &txn
		},
	)
	stateContext.On("GetTrieNode", argStr).Return(
		func(id string) util.Serializable {
			stateContext.Lock()
			defer stateContext.Unlock()
			if val, ok := stateContext.store[id]; ok {
				return val
			}
			return nil
		},
		func(id string) error {
			if strings.Contains(id, "not_present_id") {
				return util.ErrValueNotPresent
			}
			if strings.Contains(id, "unexpected_id") {
				return zmc.ErrInternalUnexpected
			}
			stateContext.Lock()
			defer stateContext.Unlock()
			if _, ok := stateContext.store[id]; ok {
				return nil
			}
			return util.ErrValueNotPresent
		},
	)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Session")).
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
		Return(funcInsertList, errFuncInsertList)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.AccessPoint")).
		Return(funcInsertID, errFuncInsertID)

	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.User")).
		Return(funcInsertID, errFuncInsertID)

	stateContext.On("DeleteTrieNode", argStr).
		Return(funcDeleteID, errFuncDeleteID)

	nodeInvalid := mockInvalidJson{ID: "invalid_json_id"}
	if _, err := stateContext.InsertTrieNode(nodeInvalid.ID, &nodeInvalid); err != nil {
		log.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	return &stateContext
}

func mockTokenPool() *tokenPool {
	now := time.Now().Format(time.RFC3339Nano)

	pool := newTokenPool()
	pool.Id = "id:session:" + now
	pool.PayerId = "id:payer:" + now
	pool.PayeeId = "id:payee:" + now
	pool.Balance = 1000

	return pool
}

func mockUsers() *Users {
	list := &Users{}
	for i := 0; i < 10; i++ {
		list.put(mockUser())
	}

	return list
}

func mockUser() *zmc.User {
	now := time.Now()
	bin, _ := now.MarshalBinary()
	sum := sha3.Sum256(bin)
	fix := hex.EncodeToString(sum[:])
	return &zmc.User{
		User: &pb.User{
			Id:         "id:user:" + fix,
			ConsumerId: "id:consumer:" + fix,
		},
	}
}
