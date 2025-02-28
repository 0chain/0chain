package minersc

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"github.com/0chain/common/core/util"
)

// Constants for MPT keys
const (
	GSoSIndexKey     = "gsos_index" // Key for storing the index of all GSoS IDs
	GSoSPartitionKey = "gsos_part"  // Prefix for individual ShareOrSign partitions
)

//go:generate msgp -io=false -tests=false -v

// GroupSharesOrSignsV2 is a combined structure that both stores the IDs of ShareOrSigns
// and provides methods to manage them in state
type GroupSharesOrSignsV2 struct {
	IDs []string `json:"ids" msg:"ids"` // IDs of all ShareOrSigns
}

// NewGroupSharesOrSignsV2 creates a new GroupSharesManager instance
func NewGroupSharesOrSignsV2() *GroupSharesOrSignsV2 {
	return &GroupSharesOrSignsV2{
		IDs: make([]string, 0),
	}
}

// addID adds an ID to the index if it doesn't already exist
func (gsm *GroupSharesOrSignsV2) addID(id string) bool {
	// Check if ID already exists
	for _, existingID := range gsm.IDs {
		if existingID == id {
			return false
		}
	}

	gsm.IDs = append(gsm.IDs, id)
	return true
}

// removeID removes an ID from the index
func (gsm *GroupSharesOrSignsV2) removeID(id string) bool {
	for i, existingID := range gsm.IDs {
		if existingID == id {
			// Remove the ID by replacing it with the last element and truncating
			gsm.IDs[i] = gsm.IDs[len(gsm.IDs)-1]
			gsm.IDs = gsm.IDs[:len(gsm.IDs)-1]
			return true
		}
	}
	return false
}

// GetIDs returns a copy of all IDs in the index
func (gsm *GroupSharesOrSignsV2) GetIDs() []string {
	result := make([]string, len(gsm.IDs))
	copy(result, gsm.IDs)
	return result
}

// ContainsID checks if the ID exists in the index
func (gsm *GroupSharesOrSignsV2) ContainsID(id string) bool {
	for _, existingID := range gsm.IDs {
		if existingID == id {
			return true
		}
	}
	return false
}

// GetSOSPartitionKey generates the MPT key for a specific ShareOrSign partition
func GetSOSPartitionKey(id string) string {
	return GSoSPartitionKey + "_" + id
}

// GetHash computes a hash of the IDs to detect changes
func (gsm *GroupSharesOrSignsV2) GetHash() string {
	// Simple implementation - concatenate all IDs and compute a hash
	// For a real implementation, consider using a cryptographic hash function
	combined := ""
	for _, id := range gsm.IDs {
		combined += id
	}
	return combined
}

// LoadFromState loads the GroupSharesManager from state
func (gsm *GroupSharesOrSignsV2) Load(state state.StateContextI) error {
	err := state.GetTrieNode(GSoSIndexKey, gsm)
	if err != nil && err != util.ErrValueNotPresent {
		return err
	}
	return nil
}

// SaveToState saves the GroupSharesManager to state
func (gsm *GroupSharesOrSignsV2) Save(state state.StateContextI) error {
	_, err := state.InsertTrieNode(GSoSIndexKey, gsm)
	return err
}

// AddShareOrSigns adds a new ShareOrSigns to the MPT
// It updates the index and creates a new partition node for the share
func (gsm *GroupSharesOrSignsV2) AddShareOrSigns(state state.StateContextI, sos *block.ShareOrSigns) error {
	//  Add the ID to the index if it doesn't exist
	if added := gsm.addID(sos.ID); !added {
		return nil // ID already exists, nothing to do
	}

	// Update the index in the MPT
	if err := gsm.Save(state); err != nil {
		return err
	}

	// Store the individual ShareOrSigns in its own partition
	partKey := GetSOSPartitionKey(sos.ID)
	_, err := state.InsertTrieNode(partKey, sos)

	return err
}

// GetShareOrSigns retrieves a ShareOrSigns by ID from the MPT
func (gsm *GroupSharesOrSignsV2) GetShareOrSigns(state state.StateContextI, id string) (*block.ShareOrSigns, error) {
	// Check if the ID exists in the index
	if !gsm.ContainsID(id) {
		return nil, util.ErrValueNotPresent
	}

	// Get the ShareOrSigns from its partition
	partKey := GetSOSPartitionKey(id)
	sos := block.NewShareOrSigns()
	err := state.GetTrieNode(partKey, sos)
	if err != nil {
		return nil, err
	}

	return sos, nil
}

// loadShareOrSigns retrieves a single ShareOrSigns by ID from the MPT
func loadShareOrSigns(id string, state cstate.StateContextI) (*block.ShareOrSigns, error) {
	sos := block.NewShareOrSigns()
	partKey := GetSOSPartitionKey(id)
	err := state.GetTrieNode(partKey, sos)
	if err != nil {
		return nil, err
	}
	return sos, nil
}

// GetAllShareOrSigns retrieves all ShareOrSigns from the MPT
func (gsm *GroupSharesOrSignsV2) GetAllShareOrSigns(state cstate.StateContextI) (*block.GroupSharesOrSigns, error) {
	// Create a new GroupSharesOrSigns to hold the result
	gsos := block.NewGroupSharesOrSigns()

	// Get all IDs
	ids := gsm.GetIDs()
	if len(ids) == 0 {
		return gsos, nil
	}

	// Retrieve all ShareOrSigns concurrently
	shares, err := cstate.GetItemsByIDs(ids, loadShareOrSigns, state)
	if err != nil {
		return nil, err
	}

	// Populate the result map
	for i, id := range ids {
		if shares[i] != nil {
			gsos.Shares[id] = shares[i]
		}
	}

	return gsos, nil
}

// DeleteShareOrSigns deletes a ShareOrSigns by ID from the MPT
func (gsm *GroupSharesOrSignsV2) DeleteShareOrSigns(state state.StateContextI, id string) error {
	// Remove the ID from the index
	if removed := gsm.removeID(id); !removed {
		return nil // ID doesn't exist, nothing to do
	}

	// Update the index in the MPT
	return gsm.Save(state)
}

// DeleteAllShareOrSigns deletes all ShareOrSigns from the MPT
func (gsm *GroupSharesOrSignsV2) DeleteAllShareOrSigns(state state.StateContextI) error {
	// Reset manager to empty state
	gsm.IDs = make([]string, 0)

	// Save empty state back to MPT
	return gsm.Save(state)
}
