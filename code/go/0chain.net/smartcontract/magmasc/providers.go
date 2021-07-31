package magmasc

import (
	"encoding/json"

	"github.com/0chain/bandwidth_marketplace/code/core/errors"
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// Providers represents sorted Provider nodes, used to inserting,
	// removing or getting from state.StateContextI with AllProvidersKey.
	Providers struct {
		Nodes providersSorted `json:"nodes"`
	}
)

var (
	// Make sure Providers implements Serializable interface.
	_ util.Serializable = (*Providers)(nil)
)

// Decode implements util.Serializable interface.
func (m *Providers) Decode(blob []byte) error {
	var sorted []*bmp.Provider
	if err := json.Unmarshal(blob, &sorted); err != nil {
		return errDecodeData.Wrap(err)
	}

	m.Nodes.setSorted(sorted)

	return nil
}

// Encode implements util.Serializable interface.
func (m *Providers) Encode() []byte {
	blob, _ := json.Marshal(m.Nodes.getSorted())
	return blob
}

// add tries to add a new provider to nodes list.
func (m *Providers) add(scID datastore.Key, prov *bmp.Provider, sci chain.StateContextI) error {
	if _, found := m.Nodes.getByHost(prov.Host); found {
		return errors.New(errCodeInternal, "provider host already registered: "+prov.Host)
	}

	return m.update(scID, prov, sci)
}

// update tries to update the provider into nodes list.
func (m *Providers) update(scID datastore.Key, prov *bmp.Provider, sci chain.StateContextI) error {
	if _, err := sci.InsertTrieNode(nodeUID(scID, prov.ExtID, providerType), prov); err != nil {
		return errors.Wrap(errCodeInternal, "insert provider failed", err)
	}

	list := &Providers{Nodes: providersSorted{Sorted: m.Nodes.getSorted()}}
	list.Nodes.add(prov)
	if _, err := sci.InsertTrieNode(AllProvidersKey, list); err != nil {
		return errors.Wrap(errCodeInternal, "insert providers list failed", err)
	}

	m.Nodes.setSorted(list.Nodes.Sorted)

	return nil
}

// fetchProviders extracts all providers represented in JSON bytes
// stored in state.StateContextI with given id.
// fetchProviders returns error if state.StateContextI does not contain
// providers or stored bytes have invalid format.
func fetchProviders(id datastore.Key, sci chain.StateContextI) (*Providers, error) {
	providers := &Providers{}
	if list, _ := sci.GetTrieNode(id); list != nil {
		if err := providers.Decode(list.Encode()); err != nil {
			return nil, errDecodeData.Wrap(err)
		}
	}

	return providers, nil
}
