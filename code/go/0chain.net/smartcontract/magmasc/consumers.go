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
	var sorted []*bmp.Consumer
	if err := json.Unmarshal(blob, &sorted); err != nil {
		return errDecodeData.Wrap(err)
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
func (m *Consumers) add(scID datastore.Key, cons *bmp.Consumer, sci chain.StateContextI) error {
	if _, err := sci.InsertTrieNode(nodeUID(scID, cons.ExtID, consumerType), cons); err != nil {
		return errors.Wrap(errCodeInternal, "insert consumer failed", err)
	}
	m.Nodes.add(cons)
	if _, err := sci.InsertTrieNode(AllConsumersKey, m); err != nil {
		return errors.Wrap(errCodeInternal, "insert consumers list failed", err)
	}

	return nil
}

// fetchConsumers extracts all consumers represented in JSON bytes
// stored in state.StateContextI with AllConsumersKey.
// fetchConsumers returns error if state.StateContextI does not contain
// consumers or stored bytes have invalid format.
func fetchConsumers(id datastore.Key, sci chain.StateContextI) (*Consumers, error) {
	consumers := Consumers{Nodes: &consumersSorted{}}
	if list, _ := sci.GetTrieNode(id); list != nil {
		if err := consumers.Decode(list.Encode()); err != nil {
			return nil, errDecodeData.Wrap(err)
		}
	}

	return &consumers, nil
}
