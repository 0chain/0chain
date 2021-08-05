package magmasc

import (
	"encoding/json"

	"github.com/0chain/bandwidth_marketplace/code/core/errors"
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	store "0chain.net/core/ememorystore"
)

type (
	// ActiveSessions represents active sessions list
	// for every registered provider and consumer.
	ActiveSessions struct {
		Items []*bmp.Acknowledgment
	}
)

func (m *ActiveSessions) add(scID string, item *bmp.Acknowledgment, db *store.Connection, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "acknowledgment invalid value").Wrap(errNilPointerValue)
	}
	if _, found := m.getIndex(item.SessionID); found {
		return errors.New(errCodeInternal, "active session already exists: "+item.SessionID)
	}

	return m.write(scID, item, db, sci)
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

func (m *ActiveSessions) del(item *bmp.Acknowledgment, db *store.Connection) error {
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
	if err = db.Conn.Put([]byte(ActiveSessionsKey), blob); err != nil {
		return errors.Wrap(errCodeInternal, "put active acknowledgments list failed", err)
	}
	if err = db.Commit(); err != nil {
		return errors.Wrap(errCodeInternal, "commit changes failed", err)
	}

	m.Items = nodes

	return nil
}

func (m *ActiveSessions) write(scID string, item *bmp.Acknowledgment, db *store.Connection, sci chain.StateContextI) error {
	if item == nil {
		return errors.New(errCodeInternal, "acknowledgment invalid value").Wrap(errNilPointerValue)
	}

	var items []*bmp.Acknowledgment
	if idx, found := m.getIndex(item.SessionID); !found { // new item
		items = append(m.Items, item)
	} else { // replace item
		items = make([]*bmp.Acknowledgment, len(m.Items))
		copy(items, m.Items)
		items[idx] = item
	}

	blob, err := json.Marshal(items)
	if err != nil {
		return errors.Wrap(errCodeInternal, "encode active sessions failed", err)
	}
	if err = db.Conn.Put([]byte(ActiveSessionsKey), blob); err != nil {
		return errors.Wrap(errCodeInternal, "insert active sessions failed", err)
	}
	if _, err = sci.InsertTrieNode(nodeUID(scID, item.SessionID, acknowledgment), item); err != nil {
		_ = db.Conn.Rollback()
		return errors.Wrap(errCodeInternal, "insert acknowledgment failed", err)
	}
	if err = db.Commit(); err != nil {
		return errors.Wrap(errCodeInternal, "commit active sessions failed", err)
	}

	m.Items = items

	return nil
}

// fetchActiveSessions extracts active sessions list
// stored in memory data store with given id.
func fetchActiveSessions(id string, db *store.Connection) (*ActiveSessions, error) {
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
