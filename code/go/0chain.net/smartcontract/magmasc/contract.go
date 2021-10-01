package magmasc

import (
	"context"
	"net/url"
	"strconv"

	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/time"
	"github.com/rcrowley/go-metrics"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	store "0chain.net/core/ememorystore"
	"0chain.net/core/util"
)

// session tries to extract Session with given id param.
func (m *MagmaSmartContract) session(id string, sci chain.StateContextI) (*zmc.Session, error) {
	data, err := sci.GetTrieNode(nodeUID(m.ID, session, id))
	if err != nil {
		return nil, errors.Wrap(errCodeFetchData, "fetch session failed", err)
	}

	sess := zmc.Session{}
	if err = sess.Decode(data.Encode()); err != nil {
		return nil, errors.Wrap(errCodeDecode, "decode session failed", err)
	}

	return &sess, nil
}

// sessionAccepted tries to extract Session with given id param.
func (m *MagmaSmartContract) sessionAccepted(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	sess, err := m.session(vals.Get("id"), sci)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

// sessionAcceptedVerify tries to extract Session with given id param,
// validate and verifies others IDs from values for equality with extracted session.
func (m *MagmaSmartContract) sessionAcceptedVerify(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	sess, err := m.session(vals.Get("session_id"), sci)
	if err != nil {
		return nil, err
	}

	switch {
	case sess.AccessPoint.ID != vals.Get("access_point_id"):
		return nil, errInvalidAccessPointID

	case sess.Consumer == nil || sess.Consumer.ExtID != vals.Get("consumer_ext_id"):
		return nil, errInvalidConsumerExtID

	case sess.Provider == nil || sess.Provider.ExtID != vals.Get("provider_ext_id"):
		return nil, errInvalidProviderExtID
	}

	return sess, nil // verified - every think is ok
}

// sessionExist tries to extract Session with given id param
// and returns boolean value of it is exists.
func (m *MagmaSmartContract) sessionExist(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	got, _ := sci.GetTrieNode(nodeUID(m.ID, session, vals.Get("id")))
	return got != nil, nil
}

// allConsumers represents MagmaSmartContract handler.
// Returns all registered Consumer's nodes stores in
// provided state.StateContextI with AllConsumersKey.
func (m *MagmaSmartContract) allConsumers(context.Context, url.Values, chain.StateContextI) (interface{}, error) {
	consumers, err := consumersFetch(AllConsumersKey, m.db)
	if err != nil {
		return nil, errors.Wrap(errCodeFetchData, "fetch consumers list failed", err)
	}

	return consumers.Sorted, nil
}

// allProviders represents MagmaSmartContract handler.
// Returns all registered Provider's nodes stores in
// provided state.StateContextI with AllProvidersKey.
func (m *MagmaSmartContract) allProviders(context.Context, url.Values, chain.StateContextI) (interface{}, error) {
	providers, err := providersFetch(AllProvidersKey, m.db)
	if err != nil {
		return nil, errors.Wrap(errCodeFetchData, "fetch providers list failed", err)
	}

	return providers.Sorted, nil
}

// consumerExist tries to extract registered consumer
// with given external id param and returns boolean value of it is exists.
func (m *MagmaSmartContract) consumerExist(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	got, _ := sci.GetTrieNode(nodeUID(m.ID, consumerType, vals.Get("ext_id")))
	return got != nil, nil
}

// consumerFetch tries to extract registered consumer
// with given external id param and returns raw consumer data.
func (m *MagmaSmartContract) consumerFetch(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	return consumerFetch(m.ID, vals.Get("ext_id"), m.db, sci)
}

// consumerRegister allows registering consumer node in the blockchain
// and then saves results in provided state.StateContextI.
func (m *MagmaSmartContract) consumerRegister(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	consumer := &zmc.Consumer{}
	if err := consumer.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeConsumerReg, "decode consumer data failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := consumersFetch(AllConsumersKey, m.db)
	if err != nil {
		return "", errors.Wrap(errCodeConsumerReg, "fetch consumers list failed", err)
	}

	consumer.ID = txn.ClientID
	if err = list.add(m.ID, consumer, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeConsumerReg, "register consumer failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeConsumerReg, "commit changes failed", err)
	}

	// update consumer register metric
	m.SmartContractExecutionStats[consumerRegister].(metrics.Counter).Inc(1)

	return string(consumer.Encode()), nil
}

// consumerSessionStart checks input for validity then inserts resulted session
// in provided state.StateContextI and starts a new session with lock tokens into
// a new token pool by accepted session data.
func (m *MagmaSmartContract) consumerSessionStart(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var err error

	sess := &zmc.Session{}
	if err = sess.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeSessionStart, "decode session data failed", err)
	}
	if sess, err = m.session(sess.SessionID, sci); err != nil {
		return "", errors.New(errCodeSessionStart, err.Error())
	}
	if sess.Consumer.ID != txn.ClientID {
		return "", errors.Wrap(errCodeSessionStart, "check owner id failed", err)
	}
	if err = sess.AccessPoint.Terms.Validate(); err != nil {
		return "", errors.Wrap(errCodeSessionStart, "invalid provider terms", err)
	}

	db := store.GetTransaction(m.db)
	pools, err := rewardPoolsFetch(allRewardPoolsKey, db)
	if err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeRewardPoolLock, "fetch token pools list failed", err)
	}

	pool := newTokenPool()
	if err = pool.create(txn, sess, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeSessionStart, "create token pool failed", err)
	}

	if err = pools.add(m.ID, pool, db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeSessionStart, "add lock pool to list failed", err)
	}

	sess.TokenPool = &pool.TokenPool
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, session, sess.SessionID), sess); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeSessionStart, "insert session failed", err)
	}

	return string(sess.Encode()), nil
}

// consumerSessionStop checks input for validity and complete the session with
// stake spent tokens and refunds remaining balance by billing data.
func (m *MagmaSmartContract) consumerSessionStop(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var err error

	sess := &zmc.Session{}
	if err = sess.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeSessionStop, "decode session data failed", err)
	}
	if sess, err = m.session(sess.SessionID, sci); err != nil {
		return "", errors.Wrap(errCodeSessionStop, "fetch consumer failed", err)
	}
	if sess.Consumer.ID != txn.ClientID {
		return "", errors.Wrap(errCodeSessionStop, "check owner id failed", err)
	}
	if sess.TokenPool == nil {
		return "", errors.Wrap(errCodeSessionStop, "session not started yet", err)
	}
	if sess.Billing.CompletedAt == 0 { // must be completed
		pool := newTokenPool()
		if err = pool.Decode(sess.TokenPool.Encode()); err != nil {
			return "", errors.New(errCodeSessionStop, err.Error())
		}

		servCharge, serviceID := m.cfg.GetFloat64(serviceCharge), sess.Provider.ID
		if err = pool.spendWithServiceCharge(txn, state.Balance(sess.Billing.Amount), sci, servCharge, serviceID); err != nil {
			return "", errors.New(errCodeSessionStop, err.Error())
		}

		sess.Billing.CompletedAt = time.Now()
		if _, err = sci.InsertTrieNode(nodeUID(m.ID, session, sess.SessionID), sess); err != nil {
			return "", errors.Wrap(errCodeSessionStop, "update session failed", err)
		}
	}

	return string(sess.Encode()), nil
}

// consumerUpdate updates the consumer data.
func (m *MagmaSmartContract) consumerUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	consumer := &zmc.Consumer{}
	if err := consumer.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeConsumerUpdate, "decode consumer data failed", err)
	}

	got, err := consumerFetch(m.ID, consumer.ExtID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(errCodeConsumerUpdate, "fetch consumer failed", err)
	}
	if got.ID != txn.ClientID {
		return "", errors.Wrap(errCodeConsumerUpdate, "check owner id failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := consumersFetch(AllConsumersKey, m.db)
	if err != nil {
		return "", errors.Wrap(errCodeConsumerUpdate, "fetch consumer list failed", err)
	}

	consumer.ID = txn.ClientID
	if err = list.write(m.ID, consumer, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeConsumerUpdate, "update consumer list failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeConsumerUpdate, "commit changes failed", err)
	}

	return string(consumer.Encode()), nil
}

// providerDataUsage updates the Provider billing session.
func (m *MagmaSmartContract) providerDataUsage(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	dataUsage := zmc.DataUsage{}
	if err := dataUsage.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeDataUsage, "decode data usage failed", err)
	}

	sess, err := m.session(dataUsage.SessionID, sci)
	if err != nil {
		return "", errors.Wrap(errCodeDataUsage, "fetch session failed", err)
	}
	if sess.TokenPool == nil {
		return "", errors.Wrap(errCodeDataUsage, "session not started yet", err)
	}
	if sess.Billing.CompletedAt != 0 { // already completed
		return "", errors.Wrap(errCodeDataUsage, "session already completed", err)
	}

	provider, err := providerFetch(m.ID, sess.Provider.ExtID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(errCodeDataUsage, "fetch provider failed", err)
	}
	if provider.ID != txn.ClientID {
		return "", errors.Wrap(errCodeDataUsage, "check owner id failed", err)
	}

	if err = sess.Billing.Validate(&dataUsage); err != nil {
		return "", errors.Wrap(errCodeDataUsage, "validate data usage failed", err)
	}

	// update billing data
	sess.Billing.DataUsage = dataUsage
	sess.Billing.CalcAmount(sess.AccessPoint.Terms)
	// TODO: make checks:
	//  the billing amount is lower than token poll balance
	//  the session is not expired yet
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, session, sess.SessionID), sess); err != nil {
		return "", errors.Wrap(errCodeDataUsage, "update billing data failed", err)
	}

	return string(sess.Encode()), nil
}

// providerMinStakeFetch tries to extract registered provider
// with given external id param and returns boolean value of it is exists.
func (m *MagmaSmartContract) providerMinStakeFetch(context.Context, url.Values, chain.StateContextI) (interface{}, error) {
	minStake := int64(m.cfg.GetFloat64(providerMinStake) * billion)
	if minStake < 0 {
		minStake = 0
	}

	return minStake, nil
}

// providerExist tries to extract registered provider
// with given external id param and returns boolean value of it is exists.
func (m *MagmaSmartContract) providerExist(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	got, _ := sci.GetTrieNode(nodeUID(m.ID, providerType, vals.Get("ext_id")))
	return got != nil, nil
}

// providerFetch tries to extract registered provider
// with given external id param and returns raw provider data.
func (m *MagmaSmartContract) providerFetch(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	return providerFetch(m.ID, vals.Get("ext_id"), m.db, sci)
}

// providerRegister allows registering provider node in the blockchain
// and saves results in provided state.StateContextI.
func (m *MagmaSmartContract) providerRegister(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	provider := &zmc.Provider{}
	if err := provider.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeProviderReg, "decode provider failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := providersFetch(AllProvidersKey, m.db)
	if err != nil {
		return "", errors.Wrap(errCodeProviderReg, "fetch providers list failed", err)
	}

	provider.ID = txn.ClientID
	if err = list.add(m.ID, provider, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeProviderReg, "register provider failed", err)
	}
	if err = m.providerStakePoolManage(txn, provider, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeProviderReg, "manage stake pool failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeProviderReg, "commit changes failed", err)
	}

	// update provider register metric
	m.SmartContractExecutionStats[providerRegister].(metrics.Counter).Inc(1)

	return string(provider.Encode()), nil
}

// providerSessionInit checks input for validity and inits a new session
// with inserts resulted session in provided state.StateContextI.
func (m *MagmaSmartContract) providerSessionInit(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	req := &zmc.Session{}
	err := req.Decode(blob)
	if err != nil {
		return "", errors.Wrap(errCodeSessionInit, "decode session data failed", err)
	}

	if req.Provider, err = providerFetch(m.ID, req.Provider.ExtID, m.db, sci); err != nil {
		return "", errors.Wrap(errCodeSessionInit, "fetch provider failed", err)
	}
	switch { // validate provider's preconditions
	case req.Provider.ID != txn.ClientID:
		return "", errors.New(errCodeSessionInit, "check owner id failed")

	case req.Provider.MinStake == 0:
		return "", errors.New(errCodeSessionInit, "session can not be started with 0 min-staked provider")
	}

	if req.AccessPoint, err = accessPointFetch(m.ID, req.AccessPoint.ID, m.db, sci); err != nil {
		return "", errors.Wrap(errCodeSessionInit, "fetch access point failed", err)
	}
	switch { // validate access point's preconditions
	case req.AccessPoint.MinStake == 0:
		return "", errors.New(errCodeSessionInit, "session can not be initialized with 0 min-staked access point")

	case req.AccessPoint.ProviderExtID != req.Provider.ExtID:
		return "", errors.New(errCodeSessionInit, "access point is not registered with provider")

	default:
		if err = req.AccessPoint.Terms.Validate(); err != nil {
			return "", errors.Wrap(errCodeSessionInit, "invalid access point terms", err)
		}
	}

	req.Consumer, err = consumerFetch(m.ID, req.Consumer.ExtID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(errCodeSessionInit, "fetch consumer failed", err)
	}

	sess := &zmc.Session{
		SessionID:   req.SessionID,
		AccessPoint: req.AccessPoint,
		Consumer:    req.Consumer,
		Provider:    req.Provider,
	}
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, session, sess.SessionID), sess); err != nil {
		return "", errors.Wrap(errCodeSessionInit, "insert session failed", err)
	}

	return string(sess.Encode()), nil
}

// providerUpdate updates the current provider terms.
func (m *MagmaSmartContract) providerUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	provider := &zmc.Provider{}
	if err := provider.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeProviderUpdate, "decode provider data failed", err)
	}

	got, err := providerFetch(m.ID, provider.ExtID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(errCodeProviderUpdate, "fetch provider failed", err)
	}
	if got.ID != txn.ClientID {
		return "", errors.Wrap(errCodeProviderUpdate, "check owner id failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := providersFetch(AllProvidersKey, m.db)
	if err != nil {
		return "", errors.Wrap(errCodeProviderUpdate, "fetch providers list failed", err)
	}

	provider.ID = txn.ClientID
	if err = list.write(m.ID, provider, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeProviderUpdate, "update providers list failed", err)
	}
	if err = m.providerStakePoolManage(txn, provider, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeProviderReg, "manage stake pool failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeProviderUpdate, "commit changes failed", err)
	}

	return string(provider.Encode()), nil
}

// providerStakePool tries to create a stake token pool for the given provider.
func (m *MagmaSmartContract) providerStakePoolManage(txn *tx.Transaction, provider *zmc.Provider, sci chain.StateContextI) error {
	if provider.ID != txn.ClientID {
		return errors.New(providerStakeTokenPool, "check owner id failed")
	}

	var pool *tokenPool
	data, err := sci.GetTrieNode(nodeUID(m.ID, providerStakeTokenPool, provider.ID))
	if provider.MinStake > 0 {
		if errors.Is(err, util.ErrValueNotPresent) {
			stake := newProviderStakeReq(provider, m.cfg)
			if err = stake.Validate(); err != nil {
				return errors.Wrap(providerStakeTokenPool, "validate stake request failed", err)
			}
			if stake.MinStake != txn.Value {
				want := strconv.FormatInt(stake.MinStake, 10)
				return errors.Wrap(providerStakeTokenPool, "transaction value must be equal to "+want, err)
			}

			pool = newTokenPool()
			if err = pool.create(txn, stake, sci); err != nil {
				return errors.Wrap(providerStakeTokenPool, "create stake pool failed", err)
			}
		}
	} else {
		if data != nil {
			pool = newTokenPool()
			if err = pool.Decode(data.Encode()); err != nil {
				return errDecodeData.Wrap(err)
			}
			//if pool.Balance != txn.Value {
			//	want := strconv.FormatInt(pool.Balance, 10)
			//	return errors.Wrap(providerStakeTokenPool, "transaction value must be equal to "+want, err)
			//} // murashovven: TODO: check is muted, don't know why it is needed
			if err = pool.spend(txn, 0, sci); err != nil {
				return errors.Wrap(providerStakeTokenPool, "refund stake pool failed", err)
			}

			if _, err := sci.DeleteTrieNode(nodeUID(m.ID, providerStakeTokenPool, provider.ID)); err != nil {
				return errors.Wrap(providerStakeTokenPool, "deleting stake pool failed", err)
			}
		}
	}

	if pool != nil { // insert new data into state context
		if _, err = sci.InsertTrieNode(nodeUID(m.ID, providerStakeTokenPool, pool.ID), pool); err != nil {
			return errors.Wrap(providerStakeTokenPool, "insert stake pool failed", err)
		}
	}

	return nil
}

// providerRegister allows registering provider node in the blockchain
// and saves results in provided state.StateContextI.
func (m *MagmaSmartContract) accessPointRegister(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	ap := &zmc.AccessPoint{}
	if err := ap.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeAccessPointReg, "decode access point failed", err)
	}

	// check provider existence
	_, err := providerFetch(m.ID, ap.ProviderExtID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(errCodeAccessPointReg, "provider is not registered", err)
	}

	db := store.GetTransaction(m.db)
	list, err := accessPointsFetch(AllAccessPointsKey, m.db)
	if err != nil {
		return "", errors.Wrap(errCodeAccessPointReg, "fetch access points list failed", err)
	}

	ap.ID = txn.ClientID
	if err = list.add(m.ID, ap, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeAccessPointReg, "register access point failed", err)
	}
	if err = m.accessPointStakePoolManage(txn, ap, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeAccessPointReg, "manage stake pool failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeAccessPointReg, "commit changes failed", err)
	}

	// update access point register metric
	m.SmartContractExecutionStats[accessPointRegister].(metrics.Counter).Inc(1)

	return string(ap.Encode()), nil
}

// accessPointStakePoolManage tries to create a stake token pool for the given access point.
func (m *MagmaSmartContract) accessPointStakePoolManage(txn *tx.Transaction, ap *zmc.AccessPoint, sci chain.StateContextI) error {
	if ap.ID != txn.ClientID {
		return errors.New(accessPointStakeTokenPool, "check owner id failed")
	}

	var pool *tokenPool
	data, err := sci.GetTrieNode(nodeUID(m.ID, accessPointStakeTokenPool, ap.ID))
	if ap.MinStake > 0 {
		if errors.Is(err, util.ErrValueNotPresent) {
			stake := newAccessPointStakeReq(ap, m.cfg)
			if err = stake.Validate(); err != nil {
				return errors.Wrap(accessPointStakeTokenPool, "validate stake request failed", err)
			}
			if stake.MinStake != txn.Value {
				want := strconv.FormatInt(stake.MinStake, 10)
				return errors.Wrap(accessPointStakeTokenPool, "transaction value must be equal to "+want, err)
			}

			pool = newTokenPool()
			if err = pool.create(txn, stake, sci); err != nil {
				return errors.Wrap(accessPointStakeTokenPool, "create stake pool failed", err)
			}
		}
	} else {
		if data != nil {
			pool = newTokenPool()
			if err = pool.Decode(data.Encode()); err != nil {
				return errDecodeData.Wrap(err)
			}
			if err = pool.spend(txn, 0, sci); err != nil {
				return errors.Wrap(accessPointStakeTokenPool, "refund stake pool failed", err)
			}

			if _, err := sci.DeleteTrieNode(nodeUID(m.ID, accessPointStakeTokenPool, ap.ID)); err != nil {
				return errors.Wrap(accessPointStakeTokenPool, "deleting stake pool failed", err)
			}
		}
	}

	if pool != nil { // insert new data into state context
		if _, err = sci.InsertTrieNode(nodeUID(m.ID, accessPointStakeTokenPool, pool.ID), pool); err != nil {
			return errors.Wrap(accessPointStakeTokenPool, "insert stake pool failed", err)
		}
	}

	return nil
}

// accessPointUpdate updates the current provider terms.
func (m *MagmaSmartContract) accessPointUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	ap := &zmc.AccessPoint{}
	if err := ap.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeAccessPointUpdate, "decode access point data failed", err)
	}

	// check provider existence
	_, err := providerFetch(m.ID, ap.ProviderExtID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(errCodeAccessPointUpdate, "provider is not registered", err)
	}

	_, err = accessPointFetch(m.ID, ap.ID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(errCodeAccessPointUpdate, "fetch access point failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := accessPointsFetch(AllAccessPointsKey, m.db)
	if err != nil {
		return "", errors.Wrap(errCodeAccessPointUpdate, "fetch access points list failed", err)
	}

	ap.ID = txn.ClientID
	if err = list.write(m.ID, ap, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeAccessPointUpdate, "update access points list failed", err)
	}
	if err = m.accessPointStakePoolManage(txn, ap, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeAccessPointUpdate, "manage stake pool failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeAccessPointUpdate, "commit changes failed", err)
	}

	return string(ap.Encode()), nil
}

// providerFetch tries to extract registered provider
// with given external id param and returns raw provider data.
func (m *MagmaSmartContract) accessPointFetch(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	return accessPointFetch(m.ID, vals.Get("id"), m.db, sci)
}

// providerFetch tries to extract registered provider
// with given external id param and returns raw provider data.
func (m *MagmaSmartContract) accessPointExist(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	got, _ := sci.GetTrieNode(nodeUID(m.ID, accessPointType, vals.Get("id")))
	return got != nil, nil
}

// accessPointMinStakeFetch tries to extract registered provider
// with given external id param and returns boolean value of it is exists.
func (m *MagmaSmartContract) accessPointMinStakeFetch(_ context.Context, _ url.Values, _ chain.StateContextI) (interface{}, error) {
	minStake := int64(m.cfg.GetFloat64(accessPointMinStake) * billion)
	if minStake < 0 {
		minStake = 0
	}

	return minStake, nil
}

// rewardPoolExist tries to extract registered reward token pool
// with given id param and returns boolean value of it is exists.
func (m *MagmaSmartContract) rewardPoolExist(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	got, _ := sci.GetTrieNode(nodeUID(m.ID, rewardTokenPool, vals.Get("id")))
	return got != nil, nil
}

// rewardPoolFetch tries to extract registered reward token pool
// with given pool id params and returns it as raw data.
func (m *MagmaSmartContract) rewardPoolFetch(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	data, err := sci.GetTrieNode(nodeUID(m.ID, rewardTokenPool, vals.Get("id")))
	if err != nil {
		return nil, err
	}

	pool := newTokenPool()
	if err = pool.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &pool, nil
}

// rewardPoolLock checks input for validity and creates
// a new reward pool intended for the payee by provided data.
func (m *MagmaSmartContract) rewardPoolLock(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var err error

	req := &tokenPoolReq{txn: txn}
	if err = req.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeRewardPoolLock, "decode lock request failed", err)
	}

	req.Provider, err = providerFetch(m.ID, req.Provider.ExtID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(errCodeRewardPoolLock, "fetch provider failed", err)
	}

	db := store.GetTransaction(m.db)
	pools, err := rewardPoolsFetch(allRewardPoolsKey, db)
	if err != nil {
		return "", errors.Wrap(errCodeRewardPoolLock, "fetch token pools list failed", err)
	}

	pool := newTokenPool()
	if err = pool.create(txn, req, sci); err != nil {
		return "", errors.Wrap(errCodeRewardPoolLock, "create lock pool failed", err)
	}
	if err = pools.add(m.ID, pool, db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeRewardPoolLock, "add lock pool to list failed", err)
	}

	return string(pool.Encode()), nil
}

// rewardPoolUnlock checks input for validity and unlocks
// the reward pool intended for the payee by provided data.
func (m *MagmaSmartContract) rewardPoolUnlock(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var err error

	req := &tokenPoolReq{txn: txn}
	if err = req.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeRewardPoolUnlock, "decode unlock request failed", err)
	}

	req.Provider, err = providerFetch(m.ID, req.Provider.ExtID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(errCodeRewardPoolUnlock, "fetch provider failed", err)
	}

	db := store.GetTransaction(m.db)
	pools, err := rewardPoolsFetch(allRewardPoolsKey, db)
	if err != nil {
		return "", errors.Wrap(errCodeRewardPoolUnlock, "fetch reward pools list failed", err)
	}

	payeeID, poolID := req.PoolPayeeID(), req.PoolID()
	pool := pools.List[payeeID][poolID]
	if pool == nil { // found
		return "", errors.Wrap(errCodeRewardPoolUnlock, "fetch reward pool failed", err)
	}
	if pool.PayerID != txn.ClientID {
		return "", errors.Wrap(errCodeRewardPoolUnlock, "check owner id failed", err)
	}
	if err = pool.spend(txn, 0, sci); err != nil {
		return "", errors.Wrap(errCodeRewardPoolUnlock, "refund reward pool failed", err)
	}
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, rewardTokenPool, pool.ID), pool); err != nil {
		return "", errors.Wrap(errCodeRewardPoolUnlock, "update reward pool failed", err)
	}
	if _, err = pools.del(payeeID, poolID, db); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(errCodeRewardPoolUnlock, "delete reward pool failed", err)
	}

	return string(pool.Encode()), nil
}

// nodeUID returns an uniq id for Node interacting with magma smart contract.
// Should be used while inserting, removing or getting nodes into state.StateContextI.
func nodeUID(scID, prefix, key string) string {
	return "sc:" + scID + colon + prefix + colon + key
}
