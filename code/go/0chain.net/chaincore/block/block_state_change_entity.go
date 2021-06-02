package block

import (
	"context"
	"encoding/json"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/state"
	"github.com/0chain/0chain/code/go/0chain.net/core/common"
	"github.com/0chain/0chain/code/go/0chain.net/core/datastore"
	"github.com/0chain/0chain/code/go/0chain.net/core/logging"
	"github.com/0chain/0chain/code/go/0chain.net/core/util"
	"go.uber.org/zap"
)

// StateChange - an entity that captures all
// changes to the state by a given block.
type StateChange struct {
	state.PartialState
	Block string `json:"block"`
}

// NewBlockStateChange - if the block state computation is successfully
// completed, provide the changes.
func NewBlockStateChange(b *Block) *StateChange {
	bsc := datastore.GetEntityMetadata("block_state_change").Instance().(*StateChange)
	bsc.Block = b.Hash
	bsc.Hash = b.ClientState.GetRoot()
	changes := b.ClientState.GetChangeCollector().GetChanges()
	bsc.Nodes = make([]util.Node, len(changes))
	for idx, change := range changes {
		bsc.Nodes[idx] = change.New
	}
	bsc.ComputeProperties()
	return bsc
}

var stateChangeEntityMetadata *datastore.EntityMetadataImpl

// StateChangeProvider - a block summary instance provider.
func StateChangeProvider() datastore.Entity {
	sc := &StateChange{}
	sc.Version = "1.0"
	return sc
}

/*GetEntityMetadata - implement interface */
func (sc *StateChange) GetEntityMetadata() datastore.EntityMetadata {
	return stateChangeEntityMetadata
}

/*Read - store read */
func (sc *StateChange) Read(ctx context.Context, key datastore.Key) error {
	return sc.GetEntityMetadata().GetStore().Read(ctx, key, sc)
}

/*Write - store read */
func (sc *StateChange) Write(ctx context.Context) error {
	return sc.GetEntityMetadata().GetStore().Write(ctx, sc)
}

/*Delete - store read */
func (sc *StateChange) Delete(ctx context.Context) error {
	return sc.GetEntityMetadata().GetStore().Delete(ctx, sc)
}

/*SetupStateChange - setup the block summary entity */
func SetupStateChange(store datastore.Store) {
	stateChangeEntityMetadata = datastore.MetadataProvider()
	stateChangeEntityMetadata.Name = "block_state_change"
	stateChangeEntityMetadata.Provider = StateChangeProvider
	stateChangeEntityMetadata.Store = store
	stateChangeEntityMetadata.IDColumnName = "hash"
	datastore.RegisterEntityMetadata("block_state_change", stateChangeEntityMetadata)
}

//MarshalJSON - implement Marshaler interface
func (sc *StateChange) MarshalJSON() ([]byte, error) {
	var data = make(map[string]interface{})
	data["block"] = sc.Block
	return sc.MartialPartialState(data)
}

//UnmarshalJSON - implement Unmarshaler interface
func (sc *StateChange) UnmarshalJSON(data []byte) error {
	var obj map[string]interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		logging.Logger.Error("unmarshal json - state change", zap.Error(err))
		return err
	}
	if block, ok := obj["block"]; ok {
		if sc.Block, ok = block.(string); !ok {
			logging.Logger.Error("unmarshal json - invalid block hash", zap.Any("obj", obj))
			return common.ErrInvalidData
		}
	} else {
		logging.Logger.Error("unmarshal json - invalid block hash", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	return sc.UnmarshalPartialState(obj)
}
