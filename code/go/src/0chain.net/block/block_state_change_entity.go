package block

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"

	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/util"
	"go.uber.org/zap"
)

//StateChange - an entity that captures all changes to the state by a given block
type StateChange struct {
	Hash    string      `json:"block"`
	Version string      `json:"version"`
	Nodes   []util.Node `json:"_"`
	mndb    *util.MemoryNodeDB
	root    util.Node
}

//NewBlockStateChange - if the block state computation is successfully completed, provide the changes
func NewBlockStateChange(b *Block) *StateChange {
	bsc := datastore.GetEntityMetadata("block_state_change").Instance().(*StateChange)
	bsc.Hash = b.Hash
	changes := b.ClientState.GetChangeCollector().GetChanges()
	bsc.Nodes = make([]util.Node, len(changes))
	for idx, change := range changes {
		bsc.Nodes[idx] = change.New
	}
	bsc.ComputeProperties()
	return bsc
}

//NewNodeDB - create a node db from the changes
func (sc *StateChange) newNodeDB() *util.MemoryNodeDB {
	mndb := util.NewMemoryNodeDB()
	for _, n := range sc.Nodes {
		mndb.PutNode(n.GetHashBytes(), n)
	}
	return mndb
}

var statChangeEntityMetadata *datastore.EntityMetadataImpl

/*StateChangeProvider - a block summary instance provider */
func StateChangeProvider() datastore.Entity {
	sc := &StateChange{}
	sc.Version = "1.0"
	return sc
}

/*GetEntityMetadata - implement interface */
func (sc *StateChange) GetEntityMetadata() datastore.EntityMetadata {
	return blockSummaryEntityMetadata
}

/*GetKey - implement interface */
func (sc *StateChange) GetKey() datastore.Key {
	return datastore.ToKey(sc.Hash)
}

/*SetKey - implement interface */
func (sc *StateChange) SetKey(key datastore.Key) {
	sc.Hash = datastore.ToString(key)
}

//ComputeProperties - implement interface
func (sc *StateChange) ComputeProperties() {
	mndb := sc.newNodeDB()
	root := mndb.ComputeRoot()
	if root != nil {
		sc.mndb = mndb
		sc.root = root
	}
}

//Validate - implement interface
func (sc *StateChange) Validate(ctx context.Context) error {
	return sc.mndb.Validate(sc.root)
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
	statChangeEntityMetadata = datastore.MetadataProvider()
	statChangeEntityMetadata.Name = "block_state_change"
	statChangeEntityMetadata.Provider = StateChangeProvider
	statChangeEntityMetadata.Store = store
	statChangeEntityMetadata.IDColumnName = "hash"
	datastore.RegisterEntityMetadata("block_state_change", statChangeEntityMetadata)
}

/*GetRoot - get the root of this set of changes */
func (sc *StateChange) GetRoot() util.Node {
	return sc.root
}

/*GetNodeDB - get the node db containing all the changes */
func (sc *StateChange) GetNodeDB() util.NodeDB {
	return sc.mndb
}

//MarshalJSON - implement Marshaler interface
func (sc *StateChange) MarshalJSON() ([]byte, error) {
	var data = make(map[string]interface{})
	data["block"] = sc.Hash
	data["version"] = sc.Version
	nodes := make([][]byte, len(sc.Nodes))
	for idx, nd := range sc.Nodes {
		nodes[idx] = nd.Encode()
	}
	data["nodes"] = nodes
	bytes, err := json.Marshal(data)
	if err != nil {
		Logger.Error("marshal JSON - state change", zap.String("block", sc.Hash), zap.Error(err))
	} else {
		Logger.Info("marshal JSON - state change", zap.String("block", sc.Hash))
	}
	return bytes, err
}

//UnmarshalJSON - implement Unmarshaler interface
func (sc *StateChange) UnmarshalJSON(data []byte) error {
	var obj map[string]interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		Logger.Error("unmarshal json - state change", zap.Error(err))
		return err
	}
	if str, ok := obj["block"].(string); ok {
		sc.Hash = str
	} else {
		Logger.Error("unmarshal json - state change", zap.String("block", sc.Hash), zap.Error(err))
		return common.ErrInvalidData
	}
	if str, ok := obj["version"].(string); ok {
		sc.Version = str
	} else {
		Logger.Error("unmarshal json - state change", zap.String("block", sc.Hash), zap.Error(err))
		return common.ErrInvalidData
	}
	if nodes, ok := obj["nodes"].([]interface{}); ok {
		sc.Nodes = make([]util.Node, len(nodes))
		for idx, nd := range nodes {
			if nd, ok := nd.(string); ok {
				buf, err := base64.StdEncoding.DecodeString(nd)
				if err != nil {
					Logger.Error("unmarshal json - state change", zap.String("block", sc.Hash), zap.Error(err))
					return err
				}
				sc.Nodes[idx], err = util.CreateNode(bytes.NewBuffer(buf))
				if err != nil {
					Logger.Error("unmarshal json - state change", zap.String("block", sc.Hash), zap.Error(err))
					return err
				}
			} else {
				Logger.Error("unmarshal json - state change", zap.String("block", sc.Hash), zap.Error(err))
				return common.ErrInvalidData
			}
		}
	} else {
		Logger.Error("unmarshal json - state change", zap.String("block", sc.Hash), zap.Error(err))
		return common.ErrInvalidData
	}
	return nil
}
