package magmasc

import (
	"encoding/json"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// Consumers represents sorted Consumer nodes, used to inserting,
	// removing or getting from state.StateContextI with AllConsumersKey.
	Consumers struct {
		Nodes *consumersSorted `json:"nodes"`
	}
)

var (
	// Make sure Consumers implements Serializable interface.
	_ util.Serializable = (*Consumers)(nil)
)

// Decode implements util.Serializable interface.
func (m *Consumers) Decode(blob []byte) error {
	var sorted []*Consumer
	if err := json.Unmarshal(blob, &sorted); err != nil {
		return errDecodeData.WrapErr(err)
	}

	if sorted != nil {
		m.Nodes = &consumersSorted{Sorted: sorted}
	}

	return nil
}

// Encode implements util.Serializable interface.
func (m *Consumers) Encode() []byte {
	blob, _ := json.Marshal(m.Nodes.Sorted)
	return blob
}

// contains looks for given Consumer by provided smart contract ID into state.StateContextI.
func (m *Consumers) contains(scID string, consumer *Consumer, sci chain.StateContextI) bool {
	if _, found := m.Nodes.getIndex(consumer.ID); found {
		return true
	}

	uid := nodeUID(scID, consumer.ID, consumerType)
	if _, err := sci.GetTrieNode(uid); err == nil {
		return m.Nodes.add(consumer)
	}

	return false
}

// extractConsumers extracts all consumers represented in JSON bytes
// stored in state.StateContextI with AllConsumersKey.
// extractConsumers returns error if state.StateContextI does not contain
// consumers or stored bytes have invalid format.
func extractConsumers(id datastore.Key, sci chain.StateContextI) (*Consumers, error) {
	consumers := Consumers{Nodes: &consumersSorted{}}
	if list, _ := sci.GetTrieNode(id); list != nil {
		if err := json.Unmarshal(list.Encode(), &consumers.Nodes.Sorted); err != nil {
			return nil, errDecodeData.WrapErr(err)
		}
	}

	return &consumers, nil
}
