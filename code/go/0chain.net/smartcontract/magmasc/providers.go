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
		Nodes *providersSorted `json:"nodes"`
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
	if sorted != nil {
		m.Nodes = &providersSorted{Sorted: sorted}
	}

	return nil
}

// Encode implements util.Serializable interface.
func (m *Providers) Encode() []byte {
	blob, _ := json.Marshal(m.Nodes.Sorted)
	return blob
}

// add tries to append consumer to nodes list.
func (m *Providers) add(scID datastore.Key, prov *bmp.Provider, sci chain.StateContextI) error {
	if _, err := sci.InsertTrieNode(nodeUID(scID, prov.ExtID, providerType), prov); err != nil {
		return errors.Wrap(errCodeInternal, "insert provider failed", err)
	}
	m.Nodes.add(prov)
	if _, err := sci.InsertTrieNode(AllProvidersKey, m); err != nil {
		return errors.Wrap(errCodeInternal, "insert providers list failed", err)
	}

	return nil
}

// fetchProviders extracts all providers represented in JSON bytes
// stored in state.StateContextI with given id.
// fetchProviders returns error if state.StateContextI does not contain
// providers or stored bytes have invalid format.
func fetchProviders(id datastore.Key, sci chain.StateContextI) (*Providers, error) {
	providers := Providers{Nodes: &providersSorted{}}
	if list, _ := sci.GetTrieNode(id); list != nil {
		if err := providers.Decode(list.Encode()); err != nil {
			return nil, errDecodeData.Wrap(err)
		}
	}

	return &providers, nil
}
