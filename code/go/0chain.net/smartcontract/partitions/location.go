package partitions

import (
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

func (p *Partitions) getItemPartIndex(state state.StateContextI, id string) (int, bool, error) {
	var pl location

	kid := p.getLocKey(id)
	loc, ok := p.locations[kid]
	if ok {
		return loc, true, nil
	}

	if err := state.GetTrieNode(kid, &pl); err != nil {
		if err == util.ErrValueNotPresent {
			return -1, false, nil
		}

		return -1, false, err
	}

	return pl.Location, true, nil
}

func (p *Partitions) getLocKey(id string) datastore.Key {
	return encryption.Hash(fmt.Sprintf("%s:%s", p.Name, id))
}

func (p *Partitions) saveItemLoc(state state.StateContextI, id string, partIndex int) error {
	_, err := state.InsertTrieNode(p.getLocKey(id), &location{Location: partIndex})
	return err
}

func (p *Partitions) removeItemLoc(state state.StateContextI, id string) error {
	_, err := state.DeleteTrieNode(p.getLocKey(id))
	return err
}

func (p *Partitions) loadLocations(idx int) {
	if p.locations == nil {
		p.locations = make(map[string]int)
	}
	if idx < 0 {
		return
	}

	// could happen removing last item and it's the last one in a partition
	if idx >= len(p.Partitions) {
		return
	}

	part := p.Partitions[idx]
	for _, it := range part.Items {
		kid := p.getLocKey(it.ID)
		if _, ok := p.locations[kid]; ok {
			return
		}

		p.locations[kid] = idx
	}
}
