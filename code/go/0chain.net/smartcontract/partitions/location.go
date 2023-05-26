package partitions

import (
	"fmt"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
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
	if err != nil {
		return fmt.Errorf("save item location failed: %v", err)
	}
	if p.locations == nil {
		p.locations = make(map[string]int)
	}
	p.locations[p.getLocKey(id)] = partIndex
	return nil
}

func (p *Partitions) removeItemLoc(state state.StateContextI, id string) error {
	kid := p.getLocKey(id)
	root := util.ToHex(state.GetState().GetRoot())
	_, err := state.DeleteTrieNode(kid)
	if err != nil {
		logging.Logger.Error("remove item location failed",
			zap.String("kid", kid),
			zap.String("id", id),
			zap.String("state root", root),
			zap.Int64("round", state.GetBlock().Round),
			zap.String("block", state.GetBlock().Hash),
			zap.Error(err),
		)
		return fmt.Errorf("remove item location failed: %v", err)
	}
	if len(p.locations) > 0 {
		delete(p.locations, kid)
	}
	return nil
}

func (p *Partitions) loadLocations(idx int) {
	if p.locations == nil {
		p.locations = make(map[string]int)
	}
	if idx <= 0 {
		return
	}

	part, ok := p.Partitions[idx]
	if !ok {
		return
	}

	for _, it := range part.Items {
		kid := p.getLocKey(it.ID)
		p.locations[kid] = idx
	}

	logging.Logger.Debug("load cache locations", zap.Any("locations", p.locations))
}
