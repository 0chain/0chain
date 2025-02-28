package minersc

import (
	"sort"
	"sync"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

// Constants for MPT keys
const (
	GSoSIndexKey     = "gsos_index" // Key for storing the index of all GSoS IDs
	GSoSPartitionKey = "gsos_part"  // Prefix for individual ShareOrSign partitions
)

//go:generate msgp -io=false -tests=false -v

// GroupSharesIndex is a lightweight structure that stores just the IDs of all ShareOrSigns
// without the actual data
type GroupSharesIndex struct {
	mutex sync.RWMutex `json:"-" msgpack:"-" msg:"-"`
	IDs   []string     `json:"ids" msg:"ids"`
}

// NewGroupSharesIndex creates a new GroupSharesIndex instance
func NewGroupSharesIndex() *GroupSharesIndex {
	return &GroupSharesIndex{
		IDs: make([]string, 0),
	}
}

// AddID adds an ID to the index if it doesn't already exist
func (gsi *GroupSharesIndex) AddID(id string) bool {
	gsi.mutex.Lock()
	defer gsi.mutex.Unlock()

	// Check if ID already exists
	for _, existingID := range gsi.IDs {
		if existingID == id {
			return false
		}
	}

	gsi.IDs = append(gsi.IDs, id)
	return true
}

// RemoveID removes an ID from the index
func (gsi *GroupSharesIndex) RemoveID(id string) bool {
	gsi.mutex.Lock()
	defer gsi.mutex.Unlock()

	for i, existingID := range gsi.IDs {
		if existingID == id {
			// Remove the ID by replacing it with the last element and truncating
			gsi.IDs[i] = gsi.IDs[len(gsi.IDs)-1]
			gsi.IDs = gsi.IDs[:len(gsi.IDs)-1]
			return true
		}
	}
	return false
}

// GetIDs returns a copy of all IDs in the index
func (gsi *GroupSharesIndex) GetIDs() []string {
	gsi.mutex.RLock()
	defer gsi.mutex.RUnlock()

	result := make([]string, len(gsi.IDs))
	copy(result, gsi.IDs)
	return result
}

// ContainsID checks if the ID exists in the index
func (gsi *GroupSharesIndex) ContainsID(id string) bool {
	gsi.mutex.RLock()
	defer gsi.mutex.RUnlock()

	for _, existingID := range gsi.IDs {
		if existingID == id {
			return true
		}
	}
	return false
}

// GetHash returns the hash of the index
func (gsi *GroupSharesIndex) GetHash() string {
	return util.ToHex(gsi.GetHashBytes())
}

// GetHashBytes returns the hash bytes of the index
func (gsi *GroupSharesIndex) GetHashBytes() []byte {
	gsi.mutex.RLock()
	defer gsi.mutex.RUnlock()

	sortedIDs := make([]string, len(gsi.IDs))
	copy(sortedIDs, gsi.IDs)
	sort.Strings(sortedIDs)

	var data []byte
	for _, id := range sortedIDs {
		data = append(data, []byte(id)...)
	}
	return encryption.RawHash(data)
}

// GetPartitionKey generates the MPT key for a specific ShareOrSign partition
func GetPartitionKey(id string) string {
	return GSoSPartitionKey + "_" + id
}

// PartitionedGroupSharesManager provides methods to manage ShareOrSigns in a distributed way
type PartitionedGroupSharesManager struct{}

// NewPartitionedGroupSharesManager creates a new manager instance
func NewPartitionedGroupSharesManager() *PartitionedGroupSharesManager {
	return &PartitionedGroupSharesManager{}
}

// AddShareOrSigns adds a new ShareOrSigns to the MPT
// It updates the index and creates a new partition node for the share
func (m *PartitionedGroupSharesManager) AddShareOrSigns(state state.StateContextI, id string, sos *block.ShareOrSigns) error {
	// 1. Get the current index
	gsi := NewGroupSharesIndex()
	err := state.GetTrieNode(GSoSIndexKey, gsi)
	if err != nil && err != util.ErrValueNotPresent {
		return err
	}

	// 2. Add the ID to the index if it doesn't exist
	if added := gsi.AddID(id); !added {
		return nil // ID already exists, nothing to do
	}

	// 3. Update the index in the MPT
	_, err = state.InsertTrieNode(GSoSIndexKey, gsi)
	if err != nil {
		return err
	}

	// 4. Set the ID in the ShareOrSigns
	sos.ID = id

	// 5. Store the individual ShareOrSigns in its own partition
	partKey := GetPartitionKey(id)
	_, err = state.InsertTrieNode(partKey, sos)

	return err
}

// GetShareOrSigns retrieves a ShareOrSigns by ID from the MPT
func (m *PartitionedGroupSharesManager) GetShareOrSigns(state state.StateContextI, id string) (*block.ShareOrSigns, error) {
	// 1. Get the current index
	gsi := NewGroupSharesIndex()
	err := state.GetTrieNode(GSoSIndexKey, gsi)
	if err != nil {
		return nil, err
	}

	// 2. Check if the ID exists in the index
	if !gsi.ContainsID(id) {
		return nil, util.ErrValueNotPresent
	}

	// 3. Get the ShareOrSigns from its partition
	partKey := GetPartitionKey(id)
	sos := block.NewShareOrSigns()
	err = state.GetTrieNode(partKey, sos)
	if err != nil {
		return nil, err
	}

	return sos, nil
}

// GetAllShareOrSigns retrieves all ShareOrSigns from the MPT
func (m *PartitionedGroupSharesManager) GetAllShareOrSigns(state state.StateContextI) (*block.GroupSharesOrSigns, error) {
	// 1. Get the current index
	gsi := NewGroupSharesIndex()
	err := state.GetTrieNode(GSoSIndexKey, gsi)
	if err != nil {
		return nil, err
	}

	// 2. Create a new GroupSharesOrSigns to hold the result
	gsos := block.NewGroupSharesOrSigns()

	// 3. Retrieve each ShareOrSigns from its partition
	ids := gsi.GetIDs()
	for _, id := range ids {
		partKey := GetPartitionKey(id)
		sos := block.NewShareOrSigns()
		err = state.GetTrieNode(partKey, sos)
		if err != nil {
			// Skip entries that can't be retrieved
			continue
		}
		gsos.Shares[id] = sos
	}

	return gsos, nil
}

// DeleteShareOrSigns deletes a ShareOrSigns by ID from the MPT
func (m *PartitionedGroupSharesManager) DeleteShareOrSigns(state state.StateContextI, id string) error {
	// 1. Get the current index
	gsi := NewGroupSharesIndex()
	err := state.GetTrieNode(GSoSIndexKey, gsi)
	if err != nil {
		return err
	}

	// 2. Remove the ID from the index
	if removed := gsi.RemoveID(id); !removed {
		return nil // ID doesn't exist, nothing to do
	}

	// 3. Update the index in the MPT
	_, err = state.InsertTrieNode(GSoSIndexKey, gsi)
	if err != nil {
		return err
	}

	// 4. Delete the partition for this ID
	partKey := GetPartitionKey(id)
	_, err = state.DeleteTrieNode(partKey)
	return err
}

// DeleteAllShareOrSigns deletes all ShareOrSigns from the MPT
func (m *PartitionedGroupSharesManager) DeleteAllShareOrSigns(state state.StateContextI) error {
	// 1. Get the current index
	gsi := NewGroupSharesIndex()
	err := state.GetTrieNode(GSoSIndexKey, gsi)
	if err != nil && err != util.ErrValueNotPresent {
		return err
	}

	// 2. Delete each partition
	for _, id := range gsi.GetIDs() {
		partKey := GetPartitionKey(id)
		_, deleteErr := state.DeleteTrieNode(partKey)
		if deleteErr != nil {
			// Continue deleting even if some entries fail
			continue
		}
	}

	// 3. Clear the index
	_, err = state.InsertTrieNode(GSoSIndexKey, NewGroupSharesIndex())
	return err
}

// GetIDs returns the list of all ShareOrSigns IDs in the MPT
func (m *PartitionedGroupSharesManager) GetIDs(state state.StateContextI) ([]string, error) {
	// Get the current index
	gsi := NewGroupSharesIndex()
	err := state.GetTrieNode(GSoSIndexKey, gsi)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}

	return gsi.GetIDs(), nil
}

// MigrateFromLegacy migrates data from the legacy GroupSharesOrSigns structure to the new partitioned structure
func (m *PartitionedGroupSharesManager) MigrateFromLegacy(state state.StateContextI, legacyKey string) error {
	// 1. Get the legacy data
	legacyGSOS := block.NewGroupSharesOrSigns()
	err := state.GetTrieNode(legacyKey, legacyGSOS)
	if err != nil {
		return err
	}

	// 2. Initialize the index
	gsi := NewGroupSharesIndex()

	// 3. Migrate each entry
	for id, sos := range legacyGSOS.GetShares() {
		// Add to index
		gsi.AddID(id)

		// Store in individual partition
		partKey := GetPartitionKey(id)
		_, err = state.InsertTrieNode(partKey, sos)
		if err != nil {
			return err
		}
	}

	// 4. Store the index
	_, err = state.InsertTrieNode(GSoSIndexKey, gsi)
	if err != nil {
		return err
	}

	// 5. Delete the legacy data (optional)
	// _, _ = state.DeleteTrieNode(legacyKey)

	return nil
}
