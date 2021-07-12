package magmasc

import (
	"encoding/json"

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
	var sorted []*Provider
	if err := json.Unmarshal(blob, &sorted); err != nil {
		return errDecodeData.WrapErr(err)
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

// contains looks for given Provider by provided smart contract ID into state.StateContextI.
func (m *Providers) contains(scID datastore.Key, provider *Provider, sci chain.StateContextI) bool {
	if _, found := m.Nodes.getIndex(provider.ID); found {
		return true
	}

	uid := nodeUID(scID, provider.ID, providerType)
	if _, err := sci.GetTrieNode(uid); err == nil {
		return m.Nodes.add(provider)
	}

	return false
}

// extractProviders extracts all providers represented in JSON bytes
// stored in state.StateContextI with given id.
// extractProviders returns error if state.StateContextI does not contain
// providers or stored bytes have invalid format.
func extractProviders(id datastore.Key, sci chain.StateContextI) (*Providers, error) {
	providers := Providers{Nodes: &providersSorted{}}
	if list, _ := sci.GetTrieNode(id); list != nil {
		if err := json.Unmarshal(list.Encode(), &providers.Nodes.Sorted); err != nil {
			return nil, errDecodeData.WrapErr(err)
		}
	}

	return &providers, nil
}
