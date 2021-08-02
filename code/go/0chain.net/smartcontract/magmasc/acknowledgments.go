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
	// ActiveAcknowledgments represents active acknowledgments list, used to inserting,
	// removing or getting from state.StateContextI with ActiveAcknowledgmentsKey.
	ActiveAcknowledgments struct {
		Nodes map[string]*bmp.Acknowledgment `json:"nodes"`
	}
)

var (
	// Make sure Consumers implements Serializable interface.
	_ util.Serializable = (*ActiveAcknowledgments)(nil)
)

// Decode implements util.Serializable interface.
func (m *ActiveAcknowledgments) Decode(blob []byte) error {
	list := ActiveAcknowledgments{}
	if err := json.Unmarshal(blob, &list); err != nil {
		return errDecodeData.Wrap(err)
	}
	if list.Nodes == nil {
		list.Nodes = make(map[string]*bmp.Acknowledgment)
	}

	m.Nodes = list.Nodes

	return nil
}

// Encode implements util.Serializable interface.
func (m *ActiveAcknowledgments) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// append tires to append a new acknowledgment to active list.
func (m *ActiveAcknowledgments) append(ackn *bmp.Acknowledgment, sci chain.StateContextI) error {
	if _, found := m.getByID(ackn.SessionID); !found {
		m.Nodes[ackn.SessionID] = ackn
		if _, err := sci.InsertTrieNode(ActiveAcknowledgmentsKey, m); err != nil {
			return errors.Wrap(errCodeInternal, "insert active acknowledgment list failed", err)
		}
	}

	return nil
}

// getByID tires to get an acknowledgment form map by given id.
func (m *ActiveAcknowledgments) getByID(id string) (ackn *bmp.Acknowledgment, found bool) {
	if m.Nodes != nil {
		ackn, found = m.Nodes[id]
	} else {
		m.Nodes = make(map[string]*bmp.Acknowledgment)
	}

	return ackn, found
}

// remove tires to remove an acknowledgment form active list.
func (m *ActiveAcknowledgments) remove(ackn *bmp.Acknowledgment, sci chain.StateContextI) error {
	if ackn == nil {
		return errors.New(errCodeInternal, "acknowledgment invalid value").Wrap(errNilPointerValue)
	}
	if _, found := m.getByID(ackn.SessionID); found {
		delete(m.Nodes, ackn.SessionID)
		if _, err := sci.InsertTrieNode(ActiveAcknowledgmentsKey, m); err != nil {
			return errors.Wrap(errCodeInternal, "insert active acknowledgment list failed", err)
		}
	}

	return nil
}

// fetchActiveAcknowledgments extracts active acknowledgments represented in JSON bytes
// stored in state.StateContextI with provided id.
// fetchConsumers returns error if state.StateContextI does not contain
// active acknowledgments or stored bytes have invalid format.
func fetchActiveAcknowledgments(id datastore.Key, sci chain.StateContextI) (*ActiveAcknowledgments, error) {
	acknowledgments := &ActiveAcknowledgments{}
	if list, _ := sci.GetTrieNode(id); list != nil {
		if err := acknowledgments.Decode(list.Encode()); err != nil {
			return nil, errDecodeData.Wrap(err)
		}
	}

	return acknowledgments, nil
}
