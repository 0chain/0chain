package magmasc

import (
	"context"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"github.com/0chain/gorocksdb"
	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/rcrowley/go-metrics"

	chain "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	tx "0chain.net/chaincore/transaction"
	store "0chain.net/core/ememorystore"
	"0chain.net/core/viper"
)

type (
	// MagmaSmartContract represents smartcontractinterface.SmartContractInterface
	// implementation allows interacting with Magma.
	MagmaSmartContract struct {
		*sci.SmartContract
		db  *gorocksdb.TransactionDB
		cfg *viper.Viper
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
	msc := MagmaSmartContract{SmartContract: sci.NewSC(zmc.Address)}

	// Magma smart contract REST handlers
	msc.RestHandlers[zmc.AccessPointFetchRP] = msc.accessPointFetch
	msc.RestHandlers[zmc.AccessPointMinStakeFetchRP] = msc.accessPointMinStakeFetch
	msc.RestHandlers[zmc.AccessPointRegisteredRP] = msc.accessPointExist
	msc.RestHandlers[zmc.ConsumerFetchRP] = msc.consumerFetch
	msc.RestHandlers[zmc.ConsumerRegisteredRP] = msc.consumerExist
	msc.RestHandlers[zmc.FetchBillingRatioRP] = msc.billingRatioFetch
	msc.RestHandlers[zmc.GetAllConsumersRP] = msc.allConsumers
	msc.RestHandlers[zmc.GetAllProvidersRP] = msc.allProviders
	msc.RestHandlers[zmc.IsSessionExistRP] = msc.sessionExist
	msc.RestHandlers[zmc.SessionRP] = msc.sessionAccepted
	msc.RestHandlers[zmc.VerifySessionAcceptedRP] = msc.sessionAcceptedVerify
	msc.RestHandlers[zmc.ProviderFetchRP] = msc.providerFetch
	msc.RestHandlers[zmc.ProviderMinStakeFetchRP] = msc.providerMinStakeFetch
	msc.RestHandlers[zmc.ProviderRegisteredRP] = msc.providerExist
	msc.RestHandlers[zmc.RewardPoolExistRP] = msc.rewardPoolExist
	msc.RestHandlers[zmc.RewardPoolFetchRP] = msc.rewardPoolFetch
	msc.RestHandlers[zmc.UserFetchRP] = msc.userFetch
	msc.RestHandlers[zmc.UserRegisteredRP] = msc.userExist

	// metrics setup section
	msc.SmartContractExecutionStats[zmc.AccessPointRegisterFuncName] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+zmc.AccessPointRegisterFuncName, nil)
	msc.SmartContractExecutionStats[zmc.ConsumerRegisterFuncName] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+zmc.ConsumerRegisterFuncName, nil)
	msc.SmartContractExecutionStats[zmc.ProviderRegisterFuncName] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+zmc.ProviderRegisterFuncName, nil)

	return &msc
}

// Execute implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) Execute(txn *tx.Transaction, call string, blob []byte, sci chain.StateContextI) (string, error) {
	switch call {
	// consumer's functions list
	case zmc.ConsumerRegisterFuncName:
		return m.consumerRegister(txn, blob, sci)
	case zmc.ConsumerSessionStartFuncName:
		return m.consumerSessionStart(txn, blob, sci)
	case zmc.ConsumerSessionStopFuncName:
		return m.consumerSessionStop(txn, blob, sci)
	case zmc.ConsumerUpdateFuncName:
		return m.consumerUpdate(txn, blob, sci)

	// provider's functions list
	case zmc.ProviderDataUsageFuncName:
		return m.providerDataUsage(txn, blob, sci)
	case zmc.ProviderRegisterFuncName:
		return m.providerRegister(txn, blob, sci)
	case zmc.ProviderStakeFuncName:
		return m.providerStake(txn, blob, sci)
	case zmc.ProviderUnStakeFuncName:
		return m.providerUnstake(txn, blob, sci)
	case zmc.ProviderUpdateFuncName:
		return m.providerUpdate(txn, blob, sci)

	// access-point's functions list
	case zmc.AccessPointRegisterFuncName:
		return m.accessPointRegister(txn, blob, sci)
	case zmc.AccessPointUpdateTermsFuncName:
		return m.accessPointTermsUpdate(txn, blob, sci)
	case zmc.AccessPointStakeFuncName:
		return m.accessPointStake(txn, blob, sci)
	case zmc.AccessPointUnStakeFuncName:
		return m.accessPointUnstake(txn, blob, sci)
	case zmc.AccessPointChangeProviderFuncName:
		return m.accessPointProviderChange(txn, blob, sci)

	// reward token pools functions list
	case zmc.RewardPoolLockFuncName:
		return m.rewardPoolLock(txn, blob, sci)
	case zmc.RewardPoolUnlockFuncName:
		return m.rewardPoolUnlock(txn, blob, sci)

	// user's functions list
	case zmc.UserRegisterFuncName:
		return m.userRegister(txn, blob, sci)
	case zmc.UserUpdateFuncName:
		return m.userUpdate(txn, blob, sci)
	}

	return "", zmc.ErrInvalidFuncName
}

// GetAddress implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetAddress() string {
	return zmc.Address
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

// Setup inits and configures the magma smart contract environment.
func (m *MagmaSmartContract) Setup(cfg *viper.Viper) error {
	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(zmc.ErrCodeInternal, "init magma smart contract store failed", err)
	}

	path := filepath.Join(usr.HomeDir, rootPath, storePath)
	if err = os.MkdirAll(path, 0644); err != nil {
		return errors.Wrap(zmc.ErrCodeInternal, "create magma smart contract store failed", err)
	}

	m.db, err = store.CreateDB(path)
	if err != nil {
		return errors.Wrap(zmc.ErrCodeInternal, "open magma smart contract store failed", err)
	}

	if err := validateCfg(cfg); err != nil {
		return errors.Wrap(zmc.ErrCodeInternal, "configuration is invalid", err)
	}
	m.cfg = cfg
	store.AddPool(storeName, m.db)

	return nil
}

// validateCfg validates provided config.
func validateCfg(cfg *viper.Viper) error {
	var (
		billRatio    = cfg.GetInt64(billingRatio)
		servCharge   = cfg.GetFloat64(serviceCharge)
		apMinStake   = cfg.GetFloat64(accessPointMinStake)
		provMinStake = cfg.GetFloat64(providerMinStake)
	)
	switch {
	case billRatio < 1:
		return errors.New(zmc.ErrCodeInvalidConfig, "billing ratio can not be less than 1")

	case !(servCharge >= 0 && servCharge < 1):
		return errors.New(zmc.ErrCodeInvalidConfig, "service charge must be in [0;1) interval")

	case apMinStake < 0:
		return errors.New(zmc.ErrCodeInvalidConfig, "access point's min stake must be greater or equal than 0")

	case provMinStake < 0:
		return errors.New(zmc.ErrCodeInvalidConfig, "provider's min stake must be greater or equal than 0")

	default:
		return nil
	}
}
