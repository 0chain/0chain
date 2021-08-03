package magmasc

import (
	"encoding/json"

	"github.com/0chain/bandwidth_marketplace/code/core/errors"
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	"0chain.net/core/datastore"
	store "0chain.net/core/ememorystore"
)

type (
	// ActiveAcknowledgments represents active acknowledgments list, used to inserting,
	// removing or getting from state.StateContextI with ActiveAcknowledgmentsKey.
	ActiveAcknowledgments struct {
		Nodes []*bmp.Acknowledgment `json:"nodes"`
	}
)

// append tires to append a new acknowledgment to active list.
func (m *ActiveAcknowledgments) append(ackn *bmp.Acknowledgment, db *store.Connection) error {
	if _, found := m.getIndex(ackn.SessionID); !found {
		nodes := append(m.Nodes, ackn)
		blob, err := json.Marshal(nodes)
		if err != nil {
			return errors.Wrap(errCodeInternal, "encode active acknowledgments list failed", err)
		}
		if err = db.Conn.Put([]byte(ActiveAcknowledgmentsKey), blob); err != nil {
			return errors.Wrap(errCodeInternal, "insert active acknowledgment list failed", err)
		}
		m.Nodes = nodes
	}

	return db.Commit()
}

// getIndex tires to get an acknowledgment form map by given id.
func (m *ActiveAcknowledgments) getIndex(id string) (int, bool) {
	for idx, item := range m.Nodes {
		if item.SessionID == id {
			return idx, true
		}
	}

	return -1, false
}

// remove tires to remove an acknowledgment form active list.
func (m *ActiveAcknowledgments) remove(ackn *bmp.Acknowledgment, db *store.Connection) error {
	if ackn == nil {
		return errors.New(errCodeInternal, "acknowledgment invalid value").Wrap(errNilPointerValue)
	}
	if idx, found := m.getIndex(ackn.SessionID); found {
		nodes := append(m.Nodes[:idx], m.Nodes[idx+1:]...)
		blob, err := json.Marshal(nodes)
		if err != nil {
			return errors.Wrap(errCodeInternal, "encode active acknowledgments list failed", err)
		}
		if err = db.Conn.Put([]byte(ActiveAcknowledgmentsKey), blob); err != nil {
			return errors.Wrap(errCodeInternal, "put active acknowledgments list failed", err)
		}
		m.Nodes = nodes
	}

	return db.Commit()
}

// fetchActiveAcknowledgments extracts active acknowledgments represented in JSON bytes
// stored in state.StateContextI with provided id.
// fetchConsumers returns error if state.StateContextI does not contain
// active acknowledgments or stored bytes have invalid format.
func fetchActiveAcknowledgments(id datastore.Key, db *store.Connection) (*ActiveAcknowledgments, error) {
	list := &ActiveAcknowledgments{}
	buf, err := db.Conn.Get(db.ReadOptions, []byte(id))
	if err != nil {
		return list, errors.Wrap(errCodeInternal, "get active acknowledgments list failed", err)
	}
	defer buf.Free()

	if err = json.Unmarshal(buf.Data(), &list.Nodes); err != nil {
		return list, errors.Wrap(errCodeInternal, "decode active acknowledgments list failed", err)
	}

	return list, nil
}
