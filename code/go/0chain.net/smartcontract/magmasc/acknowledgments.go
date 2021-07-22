package magmasc

import (
	"encoding/json"
	"sync"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// ActiveAcknowledgments represents active acknowledgments list, used to inserting,
	// removing or getting from state.StateContextI with ActiveAcknowledgmentsKey.
	ActiveAcknowledgments struct {
		Nodes []*bmp.Acknowledgment
		mutex sync.RWMutex
	}
)

var (
	// Make sure Consumers implements Serializable interface.
	_ util.Serializable = (*ActiveAcknowledgments)(nil)
)

// Decode implements util.Serializable interface.
func (m *ActiveAcknowledgments) Decode(blob []byte) error {
	var list []*bmp.Acknowledgment
	if err := json.Unmarshal(blob, &list); err != nil {
		return errDecodeData.Wrap(err)
	}
	if list != nil {
		m.mutex.Lock()
		m.Nodes = list
		m.mutex.Unlock()
	}

	return nil
}

// Encode implements util.Serializable interface.
func (m *ActiveAcknowledgments) Encode() []byte {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	blob, _ := json.Marshal(m.Nodes)

	return blob
}

// append tires to append a new acknowledgment to active list.
func (m *ActiveAcknowledgments) append(ackn *bmp.Acknowledgment, sci chain.StateContextI) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Nodes = append(m.Nodes, ackn)
	if _, err := sci.InsertTrieNode(ActiveAcknowledgmentsKey, ackn); err != nil {
		return err
	}

	return nil
}

// remove tires to remove an acknowledgment form active list.
func (m *ActiveAcknowledgments) remove(ackn *bmp.Acknowledgment, sci chain.StateContextI) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for idx, item := range m.Nodes {
		if item.SessionID == ackn.SessionID {
			nodes := make([]*bmp.Acknowledgment, len(m.Nodes)-1)
			copy(nodes[:idx], m.Nodes[:idx])
			copy(nodes[idx:], m.Nodes[idx+1:])

			m.Nodes = nodes
			if _, err := sci.InsertTrieNode(ActiveAcknowledgmentsKey, m); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// fetchActiveAcknowledgments extracts active acknowledgments represented in JSON bytes
// stored in state.StateContextI with provided id.
// fetchConsumers returns error if state.StateContextI does not contain
// active acknowledgments or stored bytes have invalid format.
func fetchActiveAcknowledgments(id datastore.Key, sci chain.StateContextI) (*ActiveAcknowledgments, error) {
	acknowledgments := ActiveAcknowledgments{}
	if list, _ := sci.GetTrieNode(id); list != nil {
		if err := acknowledgments.Decode(list.Encode()); err != nil {
			return nil, errDecodeData.Wrap(err)
		}
	}

	return &acknowledgments, nil
}
