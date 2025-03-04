package minersc

import (
	"fmt"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

//go:generate msgp -io=false -tests=false

// MpksV2 - a collection of miner MPK IDs
type MpksV2 struct {
	IDs []string `json:"ids" msg:"ids"`
}

// NewMpksV2 creates a new MpksV2 instance
func NewMpksV2() *MpksV2 {
	return &MpksV2{
		IDs: make([]string, 0),
	}
}

func getMPKKey(id string) string {
	return fmt.Sprintf("mpk_%s", id)
}

// AddMpk adds an MPK to the state
// It adds the mpk.ID to the IDs array and saves the individual MPK data with key mpk_${id}
func (m *MpksV2) AddMpk(mpk *block.MPK, balances cstate.StateContextI) error {
	// Check if ID already exists
	for _, id := range m.IDs {
		if id == mpk.ID {
			return common.NewError("contribute_mpk_failed", "already have mpk for miner")
		}
	}

	// Save the individual MPK data
	mpkKey := getMPKKey(mpk.ID)
	_, err := balances.InsertTrieNode(mpkKey, mpk)
	if err != nil {
		return err
	}

	// Add the ID to the array
	m.IDs = append(m.IDs, mpk.ID)

	return nil
}

func getMPK(id string, balances cstate.StateContextI) (*block.MPK, error) {
	mpkKey := getMPKKey(id)
	mpk := &block.MPK{}
	err := balances.GetTrieNode(mpkKey, mpk)
	if err != nil {
		return nil, err
	}

	return mpk, nil
}

func getAllMinerMPKs(ids []string, balances cstate.StateContextI) ([]*block.MPK, error) {
	mpks, err := cstate.GetItemsByIDs(ids, getMPK, balances)
	if err != nil {
		return nil, err
	}
	return mpks, nil
}

// GetAllMpks retrieves all MPKs from the state
func (m *MpksV2) GetAllMpks(balances cstate.StateContextI) (*block.Mpks, error) {
	mpks := make(map[string]*block.MPK)
	allMPKs, err := getAllMinerMPKs(m.IDs, balances)
	if err != nil {
		return nil, err
	}

	for _, mpk := range allMPKs {
		mpks[mpk.ID] = mpk
	}

	return &block.Mpks{Mpks: mpks}, nil
}

func getMinersMPKs(balances cstate.StateContextI) (*MpksV2, error) {
	mpks := &MpksV2{}
	err := balances.GetTrieNode(MinersMPKKey, mpks)
	if err != nil {
		return nil, err
	}

	return mpks, nil
}

// UpdateMpksV2 updates the MpksV2 in state
func updateMinersMPKs(balances cstate.StateContextI, mpks *MpksV2) error {
	_, err := balances.InsertTrieNode(MinersMPKKey, mpks)
	return err
}
