package magmasc

import (
	"context"
	"net/url"
	"strconv"

	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/magmasc/pb"
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
		return nil, errors.Wrap(zmc.ErrCodeFetchData, "fetch session failed", err)
	}

	sess := zmc.Session{}
	if err = sess.Decode(data.Encode()); err != nil {
		return nil, errors.Wrap(zmc.ErrCodeFetchData, "decode session failed", err)
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
	case sess.AccessPoint.Id != vals.Get("access_point_id"):
		return nil, zmc.ErrInvalidAccessPointID

	case sess.Consumer == nil || sess.Consumer.ExtID != vals.Get("consumer_ext_id"):
		return nil, zmc.ErrInvalidConsumerExtID

	case sess.Provider == nil || sess.Provider.ExtId != vals.Get("provider_ext_id"):
		return nil, zmc.ErrInvalidProviderExtID
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
// provided state.StateContextI with allConsumersKey.
func (m *MagmaSmartContract) allConsumers(context.Context, url.Values, chain.StateContextI) (interface{}, error) {
	consumers, err := consumersFetch(allConsumersKey, m.db)
	if err != nil {
		return nil, errors.Wrap(zmc.ErrCodeFetchData, "fetch consumers list failed", err)
	}

	return consumers.Sorted, nil
}

// allProviders represents MagmaSmartContract handler.
// Returns all registered Provider's nodes stores in
// provided state.StateContextI with allProvidersKey.
func (m *MagmaSmartContract) allProviders(context.Context, url.Values, chain.StateContextI) (interface{}, error) {
	providers, err := providersFetch(allProvidersKey, m.db)
	if err != nil {
		return nil, errors.Wrap(zmc.ErrCodeFetchData, "fetch providers list failed", err)
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
		return "", errors.Wrap(zmc.ErrCodeConsumerReg, "decode consumer data failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := consumersFetch(allConsumersKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeConsumerReg, "fetch consumers list failed", err)
	}

	consumer.ID = txn.ClientID
	if err = list.add(m.ID, consumer, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeConsumerReg, "register consumer failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeConsumerReg, "commit changes failed", err)
	}

	// update consumer register metric
	m.SmartContractExecutionStats[zmc.ConsumerRegisterFuncName].(metrics.Counter).Inc(1)

	return string(consumer.Encode()), nil
}

// consumerSessionStart checks input for validity and inits a new session
// with inserts resulted session in provided state.StateContextI and starts
// the session with lock tokens into a new token pool by accepted session data.
func (m *MagmaSmartContract) consumerSessionStart(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (resp string, err error) {
	sess := &zmc.Session{}
	if err = sess.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "decode session data failed", err)
	}

	if _, err = sci.GetTrieNode(nodeUID(m.ID, session, sess.SessionID)); !errors.Is(err, util.ErrValueNotPresent) {
		return "", errors.New(zmc.ErrCodeSessionStart, "session with provided ID already exist")
	}

	// flush Billing and set only Ratio field
	sess.Billing = zmc.Billing{Ratio: m.cfg.GetInt64(billingRatio)}

	// fetching, checking and setting Consumer
	if sess.Consumer, err = consumerFetch(m.ID, sess.Consumer.ExtID, m.db, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "fetch consumer failed", err)
	}
	if sess.Consumer.ID != txn.ClientID {
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "check owner id failed", err)
	}

	// fetching, checking and setting Access Point
	ap, err := accessPointFetch(m.ID, sess.AccessPoint.Id, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "fetch access point failed", err)
	}
	if ap.Terms.String() != sess.AccessPoint.Terms.String() {
		return "", errors.New(zmc.ErrCodeSessionStart, "terms should be equal to the terms from the state")
	}
	// checking if access point has provided min-stake
	if _, err = sci.GetTrieNode(nodeUID(m.ID, accessPointStakeTokenPool, sess.AccessPoint.Id)); err != nil {
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "error while fetching access point's stake pool", err)
	}
	sess.AccessPoint = ap

	// fetching, checking and setting Provider
	if sess.Provider, err = providerFetch(m.ID, sess.AccessPoint.ProviderExtId, m.db, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "fetch provider failed", err)
	}
	// checking if provider has provided min-stake
	if _, err = sci.GetTrieNode(nodeUID(m.ID, providerStakeTokenPool, sess.Provider.Id)); err != nil {
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "error while fetching provider's stake pool", err)
	}

	db := store.GetTransaction(m.db)
	pools, err := rewardPoolsFetch(allRewardPoolsKey, db)
	if err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "fetch token pools list failed", err)
	}

	pool := newTokenPool()
	if err = pool.create(txn, sess, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "create token pool failed", err)
	}
	if err = pools.add(m.ID, pool, db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "add lock pool to list failed", err)
	}

	sess.TokenPool = &pool.TokenPool
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, session, sess.SessionID), sess); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeSessionStart, "insert session failed", err)
	}

	return string(sess.Encode()), nil
}

// consumerSessionStop checks input for validity and complete the session with
// stake spent tokens and refunds remaining balance by billing data.
func (m *MagmaSmartContract) consumerSessionStop(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var err error

	sess := &zmc.Session{}
	if err = sess.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeSessionStop, "decode session data failed", err)
	}
	if sess, err = m.session(sess.SessionID, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeSessionStop, "fetch consumer failed", err)
	}
	if sess.Consumer.ID != txn.ClientID {
		return "", errors.Wrap(zmc.ErrCodeSessionStop, "check owner id failed", err)
	}
	if sess.TokenPool == nil {
		return "", errors.Wrap(zmc.ErrCodeSessionStop, "session not started yet", err)
	}
	if sess.Billing.CompletedAt == 0 { // must be completed
		if err := m.completeBilling(sess, txn, sci); err != nil {
			return "", err
		}
	}

	return string(sess.Encode()), nil
}

func (m *MagmaSmartContract) completeBilling(sess *zmc.Session, txn *tx.Transaction, sci chain.StateContextI) error {
	pool := newTokenPool()
	if err := pool.Decode(sess.TokenPool.Encode()); err != nil {
		return errors.New(zmc.ErrCodeSessionStop, err.Error())
	}

	amount := sess.Billing.Amount
	if sess.Billing.DataMarker.IsQoSType() {
		amount *= m.cfg.GetInt64(billingRatio)
	}

	servCharge, serviceID := m.cfg.GetFloat64(serviceCharge), sess.Provider.Id
	if err := pool.spendWithServiceCharge(txn, state.Balance(amount), sci, servCharge, serviceID); err != nil {
		return errors.New(zmc.ErrCodeSessionStop, err.Error())
	}

	sess.Billing.CompletedAt = time.Now()
	if _, err := sci.InsertTrieNode(nodeUID(m.ID, session, sess.SessionID), sess); err != nil {
		return errors.Wrap(zmc.ErrCodeSessionStop, "update session failed", err)
	}

	return nil
}

// consumerUpdate updates the consumer data.
func (m *MagmaSmartContract) consumerUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	consumer := &zmc.Consumer{}
	if err := consumer.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeConsumerUpdate, "decode consumer data failed", err)
	}

	got, err := consumerFetch(m.ID, consumer.ExtID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeConsumerUpdate, "fetch consumer failed", err)
	}
	if got.ID != txn.ClientID {
		return "", errors.Wrap(zmc.ErrCodeConsumerUpdate, "check owner id failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := consumersFetch(allConsumersKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeConsumerUpdate, "fetch consumer list failed", err)
	}

	consumer.ID = txn.ClientID
	if err = list.write(m.ID, consumer, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeConsumerUpdate, "update consumer list failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeConsumerUpdate, "commit changes failed", err)
	}

	return string(consumer.Encode()), nil
}

// providerDataUsage updates the Provider billing session.
func (m *MagmaSmartContract) providerDataUsage(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	dataMarker := zmc.DataMarker{}
	if err := dataMarker.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeDataUsage, "decode data usage failed", err)
	}

	sess, err := m.session(dataMarker.DataUsage.SessionId, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeDataUsage, "fetch session failed", err)
	}
	if sess.TokenPool == nil {
		return "", errors.Wrap(zmc.ErrCodeDataUsage, "session not started yet", err)
	}
	if sess.Billing.CompletedAt != 0 { // already completed
		return "", errors.Wrap(zmc.ErrCodeDataUsage, "session already completed", err)
	}

	provider, err := providerFetch(m.ID, sess.Provider.ExtId, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeDataUsage, "fetch provider failed", err)
	}
	if provider.Id != txn.ClientID {
		return "", errors.Wrap(zmc.ErrCodeDataUsage, "check owner id failed", err)
	}

	if err = sess.Billing.Validate(dataMarker.DataUsage); err != nil {
		return "", errors.Wrap(zmc.ErrCodeDataUsage, "validate data usage failed", err)
	}

	if dataMarker.IsQoSType() {
		verified, err := dataMarker.Verify()
		if err != nil {
			return "", errors.Wrap(zmc.ErrCodeDataUsage, "error while verifying signature", err)
		}
		if !verified {
			return "", errors.New(zmc.ErrCodeDataUsage, "signature is unverified")
		}
	}

	// update billing data
	sess.Billing.DataMarker = &dataMarker
	sess.Billing.CalcAmount(sess.AccessPoint)
	// TODO: make checks:
	//  the billing amount is lower than token poll balance
	//  the session is not expired yet
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, session, sess.SessionID), sess); err != nil {
		return "", errors.Wrap(zmc.ErrCodeDataUsage, "update billing data failed", err)
	}

	return string(sess.Encode()), nil
}

// providerMinStakeFetch returns configured providerMinStake.
func (m *MagmaSmartContract) providerMinStakeFetch(context.Context, url.Values, chain.StateContextI) (interface{}, error) {
	minStake := int64(m.cfg.GetFloat64(providerMinStake) * zmc.Billion)
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
		return "", errors.Wrap(zmc.ErrCodeProviderReg, "decode provider failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := providersFetch(allProvidersKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeProviderReg, "fetch providers list failed", err)
	}

	provider.Id = txn.ClientID
	if err = list.add(m.ID, provider, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeProviderReg, "register provider failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeProviderReg, "commit changes failed", err)
	}

	// update provider register metric
	m.SmartContractExecutionStats[zmc.ProviderRegisterFuncName].(metrics.Counter).Inc(1)

	return string(provider.Encode()), nil
}

func (m *MagmaSmartContract) providerStake(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	provider := &zmc.Provider{}
	if err := provider.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeProviderStake, "decode provider failed", err)
	}

	var pool *tokenPool
	_, err := sci.GetTrieNode(nodeUID(m.ID, providerStakeTokenPool, provider.Id))
	if errors.Is(err, util.ErrValueNotPresent) {
		stake := newProviderStakeReq(provider, m.cfg)
		if err := stake.Validate(); err != nil {
			return "", errors.Wrap(zmc.ErrCodeProviderStake, "validate stake request failed", err)
		}
		if stake.MinStake != txn.Value {
			want := strconv.FormatInt(stake.MinStake, 10)
			return "", errors.New(zmc.ErrCodeProviderStake, "transaction value must be equal to "+want)
		}

		pool = newTokenPool()
		if err = pool.create(txn, stake, sci); err != nil {
			return "", errors.Wrap(zmc.ErrCodeProviderStake, "create stake pool failed", err)
		}
	}

	if pool != nil { // insert new data into state context
		if _, err = sci.InsertTrieNode(nodeUID(m.ID, providerStakeTokenPool, pool.ID), pool); err != nil {
			return "", errors.Wrap(zmc.ErrCodeProviderStake, "insert stake pool failed", err)
		}
	}

	return string(provider.Encode()), nil
}

func (m *MagmaSmartContract) providerUnstake(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	provider := &zmc.Provider{}
	if err := provider.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeProviderUnstake, "decode provider failed", err)
	}

	var pool *tokenPool
	data, err := sci.GetTrieNode(nodeUID(m.ID, providerStakeTokenPool, provider.Id))
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeProviderUnstake, "data not found", err)
	}
	pool = newTokenPool()
	if err = pool.Decode(data.Encode()); err != nil {
		return "", zmc.ErrDecodeData.Wrap(err)
	}
	if pool.Balance != txn.Value {
		want := strconv.FormatInt(pool.Balance, 10)
		return "", errors.Wrap(zmc.ErrCodeProviderUnstake, "transaction value must be equal to "+want, err)
	}
	if err = pool.spend(txn, 0, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeProviderUnstake, "refund stake pool failed", err)
	}
	if _, err := sci.DeleteTrieNode(nodeUID(m.ID, providerStakeTokenPool, provider.Id)); err != nil {
		return "", errors.Wrap(zmc.ErrCodeProviderUnstake, "deleting stake pool failed", err)
	}

	return string(provider.Encode()), nil
}

// providerUpdate updates the current provider terms.
func (m *MagmaSmartContract) providerUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	provider := &zmc.Provider{}
	if err := provider.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeProviderUpdate, "decode provider data failed", err)
	}

	got, err := providerFetch(m.ID, provider.ExtId, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeProviderUpdate, "fetch provider failed", err)
	}
	if got.Id != txn.ClientID {
		return "", errors.Wrap(zmc.ErrCodeProviderUpdate, "check owner id failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := providersFetch(allProvidersKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeProviderUpdate, "fetch providers list failed", err)
	}

	provider.Id = txn.ClientID
	if err = list.write(m.ID, provider, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeProviderUpdate, "update providers list failed", err)
	}

	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeProviderUpdate, "commit changes failed", err)
	}

	return string(provider.Encode()), nil
}

// providerRegister allows registering provider node in the blockchain
// and saves results in provided state.StateContextI.
func (m *MagmaSmartContract) accessPointRegister(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	req := zmc.NewAccessPoint()
	if err := req.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointReg, "decode access point failed", err)
	}

	provList, err := providersFetch(allProvidersKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointReg, "fetch access point failed", err)
	}
	prov, err := provList.random(int64(txn.CreationDate)) // pseudo-random provider, because the provided seed is always same for txn
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointReg, "error while picking provider", err)
	}

	ap := &zmc.AccessPoint{
		AccessPoint: &pb.AccessPoint{
			Id:            txn.ClientID,
			ProviderExtId: prov.ExtId,
			Terms:         req.Terms,
		},
	}
	list, err := accessPointsFetch(allAccessPointsKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointReg, "fetch access points list failed", err)
	}
	if err = list.add(m.ID, ap, m.db, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointReg, "register access point failed", err)
	}

	// update access point register metric
	m.SmartContractExecutionStats[zmc.AccessPointRegisterFuncName].(metrics.Counter).Inc(1)

	return string(ap.Encode()), nil
}

// providerRegister allows updating registered access point's terms.
func (m *MagmaSmartContract) accessPointUpdateTerms(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	req := zmc.NewAccessPoint()
	if err := req.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointUpdTerms, "decode terms failed", err)
	}

	ap, err := accessPointFetch(m.ID, txn.ClientID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointUpdTerms, "fetch access point failed", err)
	}
	ap.Terms = req.Terms

	list, err := accessPointsFetch(allAccessPointsKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointUpdTerms, "fetch access points list failed", err)
	}

	if err = list.write(m.ID, ap, m.db, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointUpdTerms, "update access points list failed", err)
	}

	return string(ap.Encode()), nil
}

func (m *MagmaSmartContract) accessPointStake(txn *tx.Transaction, _ []byte, sci chain.StateContextI) (string, error) {
	ap, err := accessPointFetch(m.ID, txn.ClientID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointUnstake, "error while fetching access point", err)
	}

	var pool *tokenPool
	_, err = sci.GetTrieNode(nodeUID(m.ID, accessPointStakeTokenPool, txn.ClientID))
	if errors.Is(err, util.ErrValueNotPresent) {
		stake := newAccessPointStakeReq(ap, m.cfg)
		if err := stake.Validate(); err != nil {
			return "", errors.Wrap(zmc.ErrCodeAccessPointStake, "validate stake request failed", err)
		}
		if stake.MinStake != txn.Value {
			want := strconv.FormatInt(stake.MinStake, 10)
			return "", errors.New(zmc.ErrCodeAccessPointStake, "transaction value must be equal to "+want)
		}

		pool = newTokenPool()
		if err = pool.create(txn, stake, sci); err != nil {
			return "", errors.Wrap(zmc.ErrCodeAccessPointStake, "create stake pool failed", err)
		}
	}

	if pool != nil { // insert new data into state context
		if _, err = sci.InsertTrieNode(nodeUID(m.ID, accessPointStakeTokenPool, pool.ID), pool); err != nil {
			return "", errors.Wrap(zmc.ErrCodeAccessPointStake, "insert stake pool failed", err)
		}
	}

	return string(ap.Encode()), nil
}

func (m *MagmaSmartContract) accessPointUnstake(txn *tx.Transaction, _ []byte, sci chain.StateContextI) (string, error) {
	ap, err := accessPointFetch(m.ID, txn.ClientID, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointUnstake, "error while fetching access point", err)
	}

	var pool *tokenPool
	data, err := sci.GetTrieNode(nodeUID(m.ID, accessPointStakeTokenPool, ap.Id))
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointUnstake, "data not found", err)
	}
	pool = newTokenPool()
	if err = pool.Decode(data.Encode()); err != nil {
		return "", zmc.ErrDecodeData.Wrap(err)
	}
	if pool.Balance != txn.Value {
		want := strconv.FormatInt(pool.Balance, 10)
		return "", errors.Wrap(zmc.ErrCodeAccessPointUnstake, "transaction value must be equal to "+want, err)
	}
	if err = pool.spend(txn, 0, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointUnstake, "refund stake pool failed", err)
	}
	if _, err := sci.DeleteTrieNode(nodeUID(m.ID, providerStakeTokenPool, ap.Id)); err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointUnstake, "deleting stake pool failed", err)
	}

	return string(ap.Encode()), nil
}

// accessPointChangeProvider changes provider for access point by picking random from registered.
func (m *MagmaSmartContract) accessPointChangeProvider(txn *tx.Transaction, _ []byte, sci chain.StateContextI) (string, error) {
	ap, err := accessPointFetch(m.ID, txn.ClientID, m.db, sci)
	if err != nil {
		return "", err
	}

	provList, err := providersFetch(allProvidersKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointChangeProvider, "fetch access point failed", err)
	}
	prov, err := provList.random(int64(txn.CreationDate)) // pseudo-random provider, because the provided seed is always same for txn
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointChangeProvider, "error while picking provider", err)
	}
	ap.ProviderExtId = prov.ExtId

	list, err := accessPointsFetch(allAccessPointsKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointChangeProvider, "fetch access points list failed", err)
	}
	if err = list.write(m.ID, ap, m.db, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeAccessPointChangeProvider, "update access points list failed", err)
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

// accessPointMinStakeFetch returns configured accessPointMinStake.
func (m *MagmaSmartContract) accessPointMinStakeFetch(_ context.Context, _ url.Values, _ chain.StateContextI) (interface{}, error) {
	minStake := int64(m.cfg.GetFloat64(accessPointMinStake) * zmc.Billion)
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
		return nil, zmc.ErrDecodeData.Wrap(err)
	}

	return &pool, nil
}

// rewardPoolLock checks input for validity and creates
// a new reward pool intended for the payee by provided data.
func (m *MagmaSmartContract) rewardPoolLock(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var err error

	req := &tokenPoolReq{txn: txn}
	if err = req.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolLock, "decode lock request failed", err)
	}

	req.Provider, err = providerFetch(m.ID, req.Provider.ExtId, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolLock, "fetch provider failed", err)
	}

	db := store.GetTransaction(m.db)
	pools, err := rewardPoolsFetch(allRewardPoolsKey, db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolLock, "fetch token pools list failed", err)
	}

	pool := newTokenPool()
	if err = pool.create(txn, req, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolLock, "create lock pool failed", err)
	}
	if err = pools.add(m.ID, pool, db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeRewardPoolLock, "add lock pool to list failed", err)
	}

	return string(pool.Encode()), nil
}

// rewardPoolUnlock checks input for validity and unlocks
// the reward pool intended for the payee by provided data.
func (m *MagmaSmartContract) rewardPoolUnlock(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var err error

	req := &tokenPoolReq{txn: txn}
	if err = req.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolUnlock, "decode unlock request failed", err)
	}

	req.Provider, err = providerFetch(m.ID, req.Provider.ExtId, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolUnlock, "fetch provider failed", err)
	}

	db := store.GetTransaction(m.db)
	pools, err := rewardPoolsFetch(allRewardPoolsKey, db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolUnlock, "fetch reward pools list failed", err)
	}

	payeeID, poolID := req.PoolPayeeID(), req.PoolID()
	pool := pools.List[payeeID][poolID]
	if pool == nil { // found
		return "", errors.Wrap(zmc.ErrCodeRewardPoolUnlock, "fetch reward pool failed", err)
	}
	if pool.PayerID != txn.ClientID {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolUnlock, "check owner id failed", err)
	}
	if err = pool.spend(txn, 0, sci); err != nil {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolUnlock, "refund reward pool failed", err)
	}
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, rewardTokenPool, pool.ID), pool); err != nil {
		return "", errors.Wrap(zmc.ErrCodeRewardPoolUnlock, "update reward pool failed", err)
	}
	if _, err = pools.del(payeeID, poolID, db); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeRewardPoolUnlock, "delete reward pool failed", err)
	}

	return string(pool.Encode()), nil
}

// fetchBillingRatio returns configured billingRatio.
func (m *MagmaSmartContract) fetchBillingRatio(_ context.Context, _ url.Values, _ chain.StateContextI) (interface{}, error) {
	br := m.cfg.GetInt64(billingRatio)
	if br < 1 {
		br = 1
	}

	return br, nil
}

// nodeUID returns an uniq id for Node interacting with magma smart contract.
// Should be used while inserting, removing or getting nodes into state.StateContextI.
func nodeUID(scID, prefix, key string) string {
	return "sc:" + scID + colon + prefix + colon + key
}

// userRegister allows registering user in the blockchain
// and saves results in provided state.StateContextI.
func (m *MagmaSmartContract) userRegister(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	user := &zmc.User{}
	if err := user.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeUserReg, "decode user failed", err)
	}

	// check consumer existence
	_, err := consumerFetch(m.ID, user.ConsumerId, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeUserReg, "consumer is not registered", err)
	}

	db := store.GetTransaction(m.db)
	list, err := usersFetch(allUsersKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeUserReg, "fetch users list failed", err)
	}

	user.Id = txn.ClientID
	if err = list.add(m.ID, user, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeUserReg, "register user failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeUserReg, "commit changes failed", err)
	}

	// update user register metric
	m.SmartContractExecutionStats[zmc.UserRegisterFuncName].(metrics.Counter).Inc(1)

	return string(user.Encode()), nil
}

// userUpdate updates the current user.
func (m *MagmaSmartContract) userUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	user := &zmc.User{}
	if err := user.Decode(blob); err != nil {
		return "", errors.Wrap(zmc.ErrCodeUserUpdate, "decode access point data failed", err)
	}

	// check consumer existence
	_, err := consumerFetch(m.ID, user.ConsumerId, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeUserUpdate, "consumer is not registered", err)
	}

	_, err = userFetch(m.ID, user.Id, m.db, sci)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeUserUpdate, "fetch user failed", err)
	}

	db := store.GetTransaction(m.db)
	list, err := usersFetch(allUsersKey, m.db)
	if err != nil {
		return "", errors.Wrap(zmc.ErrCodeUserUpdate, "fetch users list failed", err)
	}

	user.Id = txn.ClientID
	if err = list.write(m.ID, user, m.db, sci); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeUserUpdate, "update users list failed", err)
	}
	if err = db.Commit(); err != nil {
		_ = db.Conn.Rollback()
		return "", errors.Wrap(zmc.ErrCodeUserUpdate, "commit changes failed", err)
	}

	return string(user.Encode()), nil
}

// userFetch tries to extract registered user
// with given external id param and returns raw user data.
func (m *MagmaSmartContract) userFetch(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	return userFetch(m.ID, vals.Get("id"), m.db, sci)
}

// userExist tries to extract registered user
// with given external id param and returns raw user data.
func (m *MagmaSmartContract) userExist(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	got, _ := sci.GetTrieNode(nodeUID(m.ID, userType, vals.Get("id")))
	return got != nil, nil
}
