package magmasc

import (
	"encoding/json"

	"github.com/0chain/bandwidth_marketplace/code/core/errors"
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	store "0chain.net/core/ememorystore"
)

type (
	// ActiveSessions represents active acknowledgments list, used to inserting,
	// removing or getting from state.StateContextI with ActiveAcknowledgmentsKey.
	ActiveSessions struct {
		Items []*bmp.Acknowledgment
	}
)

// append tires to append a new acknowledgment to active list.
func (m *ActiveSessions) append(item *bmp.Acknowledgment, db *store.Connection) error {
	if item == nil {
		return errors.New(errCodeInternal, "acknowledgment invalid value").Wrap(errNilPointerValue)
	}
	if _, found := m.getIndex(item.SessionID); found {
		return nil // already exists
	}

	items := append(m.Items, item)
	blob, err := json.Marshal(items)
	if err != nil {
		return errors.Wrap(errCodeInternal, "encode active acknowledgments list failed", err)
	}
	if err = db.Conn.Put([]byte(ActiveAcknowledgmentsKey), blob); err != nil {
		return errors.Wrap(errCodeInternal, "insert active acknowledgment list failed", err)
	}
	if err = db.Commit(); err != nil {
		return errors.Wrap(errCodeInternal, "commit changes failed", err)
	}

	m.Items = items

	return nil
}

// getIndex tires to get an acknowledgment form map by given id.
func (m *ActiveSessions) getIndex(id string) (int, bool) {
	for idx, item := range m.Items {
		if item.SessionID == id {
			return idx, true
		}
	}

	return -1, false
}

// remove tires to remove an acknowledgment form active list.
func (m *ActiveSessions) remove(item *bmp.Acknowledgment, db *store.Connection) error {
	if item == nil {
		return errors.New(errCodeInternal, "acknowledgment invalid value").Wrap(errNilPointerValue)
	}

	idx, found := m.getIndex(item.SessionID)
	if !found {
		return nil // does not exist
	}

	nodes := append(m.Items[:idx], m.Items[idx+1:]...)
	blob, err := json.Marshal(nodes)
	if err != nil {
		return errors.Wrap(errCodeInternal, "encode active acknowledgments list failed", err)
	}
	if err = db.Conn.Put([]byte(ActiveAcknowledgmentsKey), blob); err != nil {
		return errors.Wrap(errCodeInternal, "put active acknowledgments list failed", err)
	}
	if err = db.Commit(); err != nil {
		return errors.Wrap(errCodeInternal, "commit changes failed", err)
	}

	m.Items = nodes

	return nil
}

// fetchActiveAcknowledgments extracts active acknowledgments represented in JSON bytes
// stored in state.StateContextI with provided id.
// fetchConsumers returns error if state.StateContextI does not contain
// active acknowledgments or stored bytes have invalid format.
func fetchActiveAcknowledgments(id string, db *store.Connection) (*ActiveSessions, error) {
	list := &ActiveSessions{}

	buf, err := db.Conn.Get(db.ReadOptions, []byte(id))
	if err != nil {
		return list, errors.Wrap(errCodeInternal, "get active acknowledgments list failed", err)
	}
	defer buf.Free()

	blob := buf.Data()
	if blob != nil {
		if err = json.Unmarshal(blob, &list.Items); err != nil {
			return list, errors.Wrap(errCodeInternal, "decode active acknowledgments list failed", err)
		}
	}

	return list, nil
}
