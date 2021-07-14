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

// add tries to append consumer to nodes list.
func (m *Consumers) add(scID datastore.Key, cons *Consumer, sci chain.StateContextI) error {
	got := &Consumer{}

	data, err := sci.GetTrieNode(nodeUID(scID, cons.ID, consumerType))
	if err != nil && !errAny(err, util.ErrNodeNotFound, util.ErrValueNotPresent) {
		return errWrap(errCodeFetchData, "fetch consumer failed", err)
	}
	if data != nil { // decode consumer data
		if err = got.Decode(data.Encode()); err != nil {
			return errWrap(errCodeDecode, "decode consumer data failed", err)
		}
	}

	if !cons.Idents(got) {
		m.Nodes.add(cons)
		if _, err = sci.InsertTrieNode(AllConsumersKey, m); err != nil {
			return errWrap(errCodeInternal, "insert consumers list failed", err)
		}
		if _, err = sci.InsertTrieNode(nodeUID(scID, cons.ID, consumerType), cons); err != nil {
			return errWrap(errCodeInternal, "insert consumer failed", err)
		}
	}

	return nil
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
