package magmasc

import (
	"encoding/json"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// Provider represents providers node stored in block chain.
	Provider struct {
		ID    datastore.Key `json:"id"`
		ExtID datastore.Key `json:"ext_id,omitempty"`
		Host  datastore.Key `json:"host,omitempty"`
		Terms ProviderTerms `json:"terms"`
	}
)

var (
	// Make sure Provider implements Serializable interface.
	_ util.Serializable = (*Provider)(nil)
)

// Decode implements util.Serializable interface.
func (m *Provider) Decode(blob []byte) error {
	var provider Provider
	if err := json.Unmarshal(blob, &provider); err != nil {
		return errDecodeData.WrapErr(err)

	}
	if err := provider.Terms.validate(); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.ID = provider.ID
	m.ExtID = provider.ExtID
	m.Host = provider.Host
	m.Terms = provider.Terms

	return nil
}

// Encode implements util.Serializable interface.
func (m *Provider) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// GetType returns Provider's type.
func (m *Provider) GetType() string {
	return providerType
}

// Idents reports whether the providers have the same identifiers.
func (m *Provider) Idents(prov *Provider) bool {
	return m.ExtID == prov.ExtID && m.Host == prov.Host
}

// extractProvider extracts Provider stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func extractProvider(scID, id datastore.Key, sci chain.StateContextI) (*Provider, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, id, providerType))
	if err != nil {
		return nil, err
	}

	provider := Provider{}
	if err = provider.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.WrapErr(err)
	}

	return &provider, nil
}
