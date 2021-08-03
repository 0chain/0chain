package magmasc

import (
	"context"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"github.com/0chain/bandwidth_marketplace/code/core/errors"
	"github.com/0chain/gorocksdb"
	"github.com/rcrowley/go-metrics"

	chain "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	tx "0chain.net/chaincore/transaction"
	store "0chain.net/core/ememorystore"
)

type (
	// MagmaSmartContract represents smartcontractinterface.SmartContractInterface
	// implementation allows interacting with Magma.
	MagmaSmartContract struct {
		*sci.SmartContract
		db *gorocksdb.TransactionDB
	}
)

var (
	// Ensure MagmaSmartContract implements smartcontractinterface.SmartContractInterface.
	_ sci.SmartContractInterface = (*MagmaSmartContract)(nil)
)

// NewMagmaSmartContract creates smartcontractinterface.SmartContractInterface
// and sets provided smartcontractinterface.SmartContract to corresponding
// MagmaSmartContract field and configures RestHandlers and SmartContractExecutionStats.
func NewMagmaSmartContract() *MagmaSmartContract {
	msc := MagmaSmartContract{SmartContract: sci.NewSC(Address)}

	// Magma smart contract REST handlers
	msc.RestHandlers["/acknowledgmentAccepted"] = msc.acknowledgmentAccepted
	msc.RestHandlers["/acknowledgmentAcceptedVerify"] = msc.acknowledgmentAcceptedVerify
	msc.RestHandlers["/acknowledgmentExist"] = msc.acknowledgmentExist
	msc.RestHandlers["/activeAcknowledgments"] = msc.activeAcknowledgments
	msc.RestHandlers["/allConsumers"] = msc.allConsumers
	msc.RestHandlers["/allProviders"] = msc.allProviders
	msc.RestHandlers["/consumerExist"] = msc.consumerExist
	msc.RestHandlers["/consumerFetch"] = msc.consumerFetch
	msc.RestHandlers["/providerExist"] = msc.providerExist
	msc.RestHandlers["/providerFetch"] = msc.providerFetch
	msc.RestHandlers["/providerTerms"] = msc.providerTerms

	// metrics setup section
	msc.SmartContractExecutionStats[consumerRegister] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+consumerRegister, nil)
	msc.SmartContractExecutionStats[providerRegister] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+providerRegister, nil)

	return &msc
}

// Execute implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) Execute(txn *tx.Transaction, call string, blob []byte, sci chain.StateContextI) (string, error) {
	switch call {
	// consumer's function list
	case consumerRegister:
		return m.consumerRegister(txn, blob, sci)
	case consumerSessionStart:
		return m.consumerSessionStart(txn, blob, sci)
	case consumerSessionStop:
		return m.consumerSessionStop(txn, blob, sci)
	case consumerUpdate:
		return m.consumerUpdate(txn, blob, sci)

	// provider's function list
	case providerDataUsage:
		return m.providerDataUsage(txn, blob, sci)
	case providerRegister:
		return m.providerRegister(txn, blob, sci)
	case providerUpdate:
		return m.providerUpdate(txn, blob, sci)
	}

	return "", errInvalidFuncName
}

// GetAddress implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetAddress() string {
	return Address
}

// GetExecutionStats implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetExecutionStats() map[string]interface{} {
	return m.SmartContractExecutionStats
}

// GetHandlerStats implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return m.SmartContract.HandlerStats(ctx, params)
}

// GetName implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetName() string {
	return Name
}

// GetRestPoints implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetRestPoints() map[string]sci.SmartContractRestHandler {
	return m.RestHandlers
}

// InitStore inits and configures the magma smart contract environment.
func (m *MagmaSmartContract) InitStore() error {
	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(errCodeInternal, "init magma smart contract store failed", err)
	}

	path := filepath.Join(usr.HomeDir, rootPath, storePath)
	if err = os.MkdirAll(path, 0644); err != nil {
		return errors.Wrap(errCodeInternal, "create magma smart contract store failed", err)
	}

	m.db, err = store.CreateDB(path)
	if err != nil {
		return errors.Wrap(errCodeInternal, "open magma smart contract store failed", err)
	}

	store.AddPool(storeName, m.db)

	return nil
}
