package magmasc

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/0chain/bandwidth_marketplace/code/core/errors"
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"
	"github.com/0chain/bandwidth_marketplace/code/core/time"
	"github.com/rcrowley/go-metrics"

	chain "0chain.net/chaincore/chain/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

// acknowledgment tries to extract Acknowledgment with given id param.
func (m *MagmaSmartContract) acknowledgment(id datastore.Key, sci chain.StateContextI) (*bmp.Acknowledgment, error) {
	data, err := sci.GetTrieNode(nodeUID(m.ID, id, acknowledgment))
	if err != nil {
		return nil, err
	}

	ackn := bmp.Acknowledgment{}
	if err = ackn.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &ackn, nil
}

// acknowledgmentAccepted tries to extract Acknowledgment with given id param.
func (m *MagmaSmartContract) acknowledgmentAccepted(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	ackn, err := m.acknowledgment(vals.Get("id"), sci)
	if err != nil {
		return nil, err
	}

	return ackn, nil
}

// acknowledgmentAcceptedVerify tries to extract Acknowledgment with given id param,
// validate and verifies others IDs from values for equality with extracted acknowledgment.
func (m *MagmaSmartContract) acknowledgmentAcceptedVerify(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	ackn, err := m.acknowledgment(vals.Get("session_id"), sci)
	if err != nil {
		return nil, err
	}

	switch {
	case ackn.AccessPointID != vals.Get("access_point_id"):
		return nil, errInvalidAccessPointID

	case ackn.Consumer.ExtID != vals.Get("consumer_ext_id"):
		return nil, errInvalidConsumerExtID

	case ackn.Provider.ExtID != vals.Get("provider_ext_id"):
		return nil, errInvalidProviderExtID
	}

	return ackn, nil // verified - every think is ok
}

// acknowledgmentExist tries to extract Acknowledgment with given id param
// and returns boolean value of it is existed.
func (m *MagmaSmartContract) acknowledgmentExist(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	got, _ := sci.GetTrieNode(nodeUID(m.ID, vals.Get("id"), acknowledgment))
	return got != nil, nil
}

// activeAcknowledgmentAppend tries to append a new active acknowledgment to list
// and stores the acknowledgment and updated list into provided state.StateContextI.
func (m *MagmaSmartContract) activeAcknowledgmentAppend(ackn *bmp.Acknowledgment, sci chain.StateContextI) error {
	list, err := fetchActiveAcknowledgments(ActiveAcknowledgmentsKey, sci)
	if err != nil {
		return err
	}
	if err = list.append(ackn, sci); err != nil {
		return err
	}
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, ackn.SessionID, acknowledgment), ackn); err != nil {
		return errors.Wrap(errCodeInternal, "insert acknowledgment failed", err)
	}

	return nil
}

// activeAcknowledgmentComplete tries to delete an active acknowledgment from list
// and stores the acknowledgment and updated list into provided state.StateContextI.
func (m *MagmaSmartContract) activeAcknowledgmentComplete(ackn *bmp.Acknowledgment, sci chain.StateContextI) error {
	list, err := fetchActiveAcknowledgments(ActiveAcknowledgmentsKey, sci)
	if err != nil {
		return err
	}
	if err = list.remove(ackn, sci); err != nil {
		return err
	}

	ackn.Billing.CompletedAt = time.Now()
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, ackn.SessionID, acknowledgment), ackn); err != nil {
		return errors.Wrap(errCodeInternal, "insert acknowledgment failed", err)
	}

	return nil
}

// activeAcknowledgments tries to extract Acknowledgments list with current status active
// filtered by given external id param and returns it.
func (m *MagmaSmartContract) activeAcknowledgments(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	acknowledgments, err := fetchActiveAcknowledgments(ActiveAcknowledgmentsKey, sci)
	if err != nil {
		return nil, err
	}

	extID, list := vals.Get("ext_id"), ActiveAcknowledgments{}

	list.mutex.RLock()
	defer list.mutex.RUnlock()

	for _, ackn := range acknowledgments.Nodes {
		if ackn.Consumer.ExtID == extID || ackn.Provider.ExtID == extID {
			list.Nodes = append(list.Nodes, ackn)
		}
	}

	return list.Nodes, nil
}

// allConsumers represents MagmaSmartContract handler.
// Returns all registered Consumer's nodes stores in
// provided state.StateContextI with AllConsumersKey.
func (m *MagmaSmartContract) allConsumers(_ context.Context, _ url.Values, sci chain.StateContextI) (interface{}, error) {
	consumers, err := fetchConsumers(AllConsumersKey, sci)
	if err != nil {
		return nil, errors.Wrap(errCodeFetchData, "fetch consumers list failed", err)
	}

	return consumers.Nodes.Sorted, nil
}

// allProviders represents MagmaSmartContract handler.
// Returns all registered Provider's nodes stores in
// provided state.StateContextI with AllProvidersKey.
func (m *MagmaSmartContract) allProviders(_ context.Context, _ url.Values, sci chain.StateContextI) (interface{}, error) {
	providers, err := fetchProviders(AllProvidersKey, sci)
	if err != nil {
		return nil, errors.Wrap(errCodeFetchData, "fetch providers list failed", err)
	}

	return providers.Nodes.Sorted, nil
}

// consumerExist tries to extract registered consumer
// with given external id param and returns boolean value of it is existed.
func (m *MagmaSmartContract) consumerExist(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	got, _ := sci.GetTrieNode(nodeUID(m.ID, vals.Get("ext_id"), consumerType))
	return got != nil, nil
}

// consumerFetch tries to extract registered consumer
// with given external id param and returns raw consumer data.
func (m *MagmaSmartContract) consumerFetch(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	return consumerFetch(m.ID, vals.Get("ext_id"), sci)
}

// consumerRegister allows registering consumer node in the blockchain
// and then saves results in provided state.StateContextI.
func (m *MagmaSmartContract) consumerRegister(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	consumer := &bmp.Consumer{}
	if err := consumer.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeConsumerReg, "decode consumer data failed", err)
	}

	list, err := fetchConsumers(AllConsumersKey, sci)
	if err != nil {
		return "", errors.Wrap(errCodeConsumerReg, "fetch consumers list failed", err)
	}
	if _, got := list.Nodes.getIndex(consumer.ExtID); got {
		return "", errors.Wrap(errCodeConsumerReg, "consumer already registered", err)
	}

	consumer.ID = txn.ClientID
	if err = list.add(m.ID, consumer, sci); err != nil {
		return "", errors.Wrap(errCodeConsumerReg, "register consumer failed", err)
	}

	// update consumer register metric
	m.SmartContractExecutionStats[consumerRegister].(metrics.Counter).Inc(1)

	return string(consumer.Encode()), nil
}

// consumerSessionStart checks input for validity then inserts resulted acknowledgment
// in provided state.StateContextI and starts a new session with lock tokens into
// a new token pool by accepted acknowledgment data.
func (m *MagmaSmartContract) consumerSessionStart(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	ackn := &bmp.Acknowledgment{}

	err := ackn.Decode(blob)
	if err != nil {
		return "", errors.Wrap(errCodeSessionStart, "decode acknowledgment data failed", err)
	}

	ackn.Consumer, err = consumerFetch(m.ID, ackn.Consumer.ExtID, sci)
	if err != nil || ackn.Consumer.ID != txn.ClientID {
		return "", errors.Wrap(errCodeSessionStart, "fetch consumer failed", err)
	}

	ackn.Provider, err = providerFetch(m.ID, ackn.Provider.ExtID, sci)
	if err != nil {
		return "", errors.Wrap(errCodeSessionStart, "fetch provider failed", err)
	}
	if err = ackn.Provider.Terms.Validate(); err != nil {
		return "", errors.New(errCodeSessionStart, "invalid provider terms")
	}

	if err = m.activeAcknowledgmentAppend(ackn, sci); err != nil {
		return "", errors.Wrap(errCodeSessionStart, "append acknowledgment failed", err)
	}

	resp, err := newTokenPool().create(txn, ackn, sci)
	if err != nil {
		return "", errors.Wrap(errCodeSessionStart, "create token pool failed", err)
	}

	list, err := fetchProviders(AllProvidersKey, sci)
	if err != nil {
		return "", errors.Wrap(errCodeSessionStart, "fetch providers list failed", err)
	}
	if prov, found := list.Nodes.get(ackn.Provider.ExtID); found {
		prov.Terms.Increase()
		if err = list.update(m.ID, prov, sci); err != nil {
			return "", errors.Wrap(errCodeSessionStart, "update providers list failed", err)
		}
	}

	return string(resp.Encode()), nil
}

// consumerSessionStop checks input for validity and complete the session with
// stake spent tokens and refunds remaining balance by billing data.
func (m *MagmaSmartContract) consumerSessionStop(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var req bmp.Acknowledgment
	if err := json.Unmarshal(blob, &req); err != nil {
		return "", errors.Wrap(errCodeSessionStop, "decode request failed", err)
	}

	ackn, err := m.acknowledgment(req.SessionID, sci)
	if err != nil || ackn.Consumer.ID != txn.ClientID {
		return "", errors.Wrap(errCodeSessionStop, "fetch acknowledgment failed", err)
	}

	pool, err := m.tokenPollFetch(ackn, sci)
	if err != nil {
		return "", errors.New(errCodeSessionStop, err.Error())
	}

	resp, err := pool.spend(txn, &ackn.Billing, sci)
	if err != nil { // spend token pool to provider
		return "", errors.New(errCodeSessionStop, err.Error())
	}
	if err = m.activeAcknowledgmentComplete(ackn, sci); err != nil {
		return "", errors.Wrap(errCodeSessionStop, "append acknowledgment failed", err)
	}

	list, err := fetchProviders(AllProvidersKey, sci)
	if err != nil {
		return "", errors.Wrap(errCodeSessionStop, "fetch providers list failed", err)
	}
	if provider, found := list.Nodes.get(ackn.Provider.ExtID); found {
		provider.Terms.Decrease()
		if err = list.update(m.ID, provider, sci); err != nil {
			return "", errors.Wrap(errCodeSessionStop, "update providers list failed", err)
		}
	}

	return string(resp.Encode()), nil
}

// consumerUpdate updates the consumer data.
func (m *MagmaSmartContract) consumerUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	consumer := &bmp.Consumer{}
	if err := consumer.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeConsumerUpdate, "decode consumer data failed", err)
	}
	if got, err := consumerFetch(m.ID, consumer.ExtID, sci); err != nil || got.ID != txn.ClientID {
		return "", errors.Wrap(errCodeConsumerUpdate, "fetch consumer failed", err)
	}

	list, err := fetchConsumers(AllConsumersKey, sci)
	if err != nil {
		return "", errors.Wrap(errCodeConsumerUpdate, "fetch consumer list failed", err)
	}

	consumer.ID = txn.ClientID
	if err = list.update(m.ID, consumer, sci); err != nil {
		return "", errors.Wrap(errCodeConsumerUpdate, "update consumer list failed", err)
	}

	return string(consumer.Encode()), nil
}

// providerDataUsage updates the Provider billing session.
func (m *MagmaSmartContract) providerDataUsage(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	dataUsage := bmp.DataUsage{}
	if err := dataUsage.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeDataUsage, "decode data usage failed", err)
	}

	ackn, err := m.acknowledgment(dataUsage.SessionID, sci)
	if err != nil {
		return "", errors.Wrap(errCodeDataUsage, "fetch acknowledgment failed", err)
	}

	provider, err := providerFetch(m.ID, ackn.Provider.ExtID, sci)
	if err != nil || provider.ID != txn.ClientID {
		return "", errors.Wrap(errCodeDataUsage, "fetch provider failed", err)
	}

	if err = ackn.Billing.Validate(&dataUsage); err != nil {
		return "", errors.Wrap(errCodeDataUsage, "validate data usage failed", err)
	}

	ackn.Billing.DataUsage = dataUsage
	ackn.Billing.CalcAmount(ackn.Provider.Terms)
	if _, err = sci.InsertTrieNode(nodeUID(m.ID, ackn.SessionID, acknowledgment), ackn); err != nil { // update billing data
		return "", errors.Wrap(errCodeDataUsage, "insert billing data failed", err)
	}

	return string(ackn.Encode()), nil
}

// providerExist tries to extract registered provider
// with given external id param and returns boolean value of it is existed.
func (m *MagmaSmartContract) providerExist(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	got, _ := sci.GetTrieNode(nodeUID(m.ID, vals.Get("ext_id"), providerType))
	return got != nil, nil
}

// providerFetch tries to extract registered provider
// with given external id param and returns raw provider data.
func (m *MagmaSmartContract) providerFetch(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	return providerFetch(m.ID, vals.Get("ext_id"), sci)
}

// providerRegister allows registering provider node in the blockchain
// and saves results in provided state.StateContextI.
func (m *MagmaSmartContract) providerRegister(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	provider := &bmp.Provider{}
	if err := provider.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeProviderReg, "decode provider data failed", err)
	}
	if err := provider.Terms.Validate(); err != nil {
		return "", errors.Wrap(errCodeProviderReg, "validate provider failed", err)
	}

	list, err := fetchProviders(AllProvidersKey, sci)
	if err != nil {
		return "", errors.Wrap(errCodeProviderReg, "fetch providers list failed", err)
	}
	if _, got := list.Nodes.getIndex(provider.ExtID); got {
		return "", errors.Wrap(errCodeProviderReg, "provider already registered", err)
	}

	provider.ID = txn.ClientID
	if err = list.add(m.ID, provider, sci); err != nil {
		return "", errors.Wrap(errCodeProviderReg, "register provider failed", err)
	}

	// update provider register metric
	m.SmartContractExecutionStats[providerRegister].(metrics.Counter).Inc(1)

	return string(provider.Encode()), nil
}

// providerTerms represents MagmaSmartContract handler.
// providerTerms looks for Provider with id, passed in params url.Values,
// in provided state.StateContextI and returns Provider.Terms.
func (m *MagmaSmartContract) providerTerms(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	provider, err := providerFetch(m.ID, vals.Get("ext_id"), sci)
	if err != nil {
		return nil, errors.Wrap(errCodeFetchData, "fetch provider failed", err)
	}

	return provider.Terms, nil
}

// providerUpdate updates the current provider terms.
func (m *MagmaSmartContract) providerUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	provider := &bmp.Provider{}
	if err := provider.Decode(blob); err != nil {
		return "", errors.Wrap(errCodeProviderUpdate, "decode provider data failed", err)
	}
	if err := provider.Terms.Validate(); err != nil {
		return "", errors.New(errCodeProviderUpdate, "invalid provider terms")
	}
	if got, err := providerFetch(m.ID, provider.ExtID, sci); err != nil || got.ID != txn.ClientID {
		return "", errors.Wrap(errCodeProviderUpdate, "fetch provider failed", err)
	}

	list, err := fetchProviders(AllProvidersKey, sci)
	if err != nil {
		return "", errors.Wrap(errCodeProviderUpdate, "fetch providers list failed", err)
	}
	if err = list.update(m.ID, provider, sci); err != nil {
		return "", errors.Wrap(errCodeProviderUpdate, "update providers list failed", err)
	}

	return string(provider.Encode()), nil
}

// tokenPollFetch fetches token pool form provided state.StateContextI.
func (m *MagmaSmartContract) tokenPollFetch(ackn *bmp.Acknowledgment, sci chain.StateContextI) (*tokenPool, error) {
	var pool tokenPool

	pool.ID = ackn.SessionID
	data, err := sci.GetTrieNode(pool.uid(m.ID))
	if err != nil {
		return nil, errors.Wrap(errCodeFetchData, "fetch token pool failed", err)
	}
	if err = pool.Decode(data.Encode()); err != nil {
		return nil, errors.Wrap(errCodeFetchData, "decode token pool failed", err)
	}

	if pool.ID != ackn.SessionID {
		return nil, errors.New(errCodeFetchData, "malformed token pool: "+ackn.SessionID)
	}
	if pool.PayerID != ackn.Consumer.ID {
		return nil, errors.New(errCodeFetchData, "not a payer owned token pool: "+ackn.Consumer.ID)
	}
	if pool.PayeeID != ackn.Provider.ID {
		return nil, errors.New(errCodeFetchData, "not a payee owned token pool: "+ackn.Provider.ID)
	}

	return &pool, nil
}

// nodeUID returns a uniq id for Node interacting with magma smart contract.
// Should be used while inserting, removing or getting Node in state.StateContextI
func nodeUID(scID, nodeID, nodeType datastore.Key) datastore.Key {
	return "sc:" + scID + colon + nodeType + colon + nodeID
}
