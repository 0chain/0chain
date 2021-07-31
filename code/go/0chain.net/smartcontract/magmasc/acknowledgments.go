package magmasc

import (
	"encoding/hex"
	"encoding/json"
	"sync"

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
		Nodes []*bmp.Acknowledgment `json:"nodes"`
		mutex sync.RWMutex
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

	m.mutex.Lock()
	m.Nodes = list.Nodes
	m.mutex.Unlock()

	return nil
}

// Encode implements util.Serializable interface.
func (m *ActiveAcknowledgments) Encode() []byte {
	m.mutex.RLock()
	blob, _ := json.Marshal(m)
	m.mutex.RUnlock()

	return blob
}

// append tires to append a new acknowledgment to active list.
func (m *ActiveAcknowledgments) append(ackn *bmp.Acknowledgment, sci chain.StateContextI) error {
	m.mutex.Lock()
	m.Nodes = append(m.Nodes, ackn)
	m.mutex.Unlock()

	if _, err := sci.InsertTrieNode(ActiveAcknowledgmentsKey, m); err != nil {
		mpt := sci.GetState()
		root := hex.EncodeToString(mpt.GetRoot())
		err = errors.Wrap(errCodeInternal, "mpt root: "+root+" ", err)
		return errors.Wrap(errCodeInternal, "insert active acknowledgment list failed", err)
	}

	return nil
}

// remove tires to remove an acknowledgment form active list.
func (m *ActiveAcknowledgments) remove(ackn *bmp.Acknowledgment, sci chain.StateContextI) error {
	if ackn == nil {
		return errors.New(errCodeInternal, "acknowledgment invalid value").Wrap(errNilPointerValue)
	}

	m.mutex.Lock()
	for idx, item := range m.Nodes {
		if item.SessionID == ackn.SessionID {
			size := len(m.Nodes)-1
			nodes := make([]*bmp.Acknowledgment, size)
			if size > 0 {
				copy(nodes[:idx], m.Nodes[:idx])
				copy(nodes[idx:], m.Nodes[idx+1:])
			}
			m.Nodes = nodes
			break
		}
	}
	m.mutex.Unlock()

	if _, err := sci.InsertTrieNode(ActiveAcknowledgmentsKey, m); err != nil {
		return errors.Wrap(errCodeInternal, "insert active acknowledgment list failed", err)
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
