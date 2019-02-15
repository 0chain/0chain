package state

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

//PartialState - an entity to exchange partial state
type PartialState struct {
	Hash    string      `json:"root"`
	Version string      `json:"version"`
	Nodes   []util.Node `json:"_"`
	mndb    *util.MemoryNodeDB
	root    util.Node
}

//NewPartialState - create a new partial state object with initialization
func NewPartialState(key util.Key) *PartialState {
	ps := datastore.GetEntityMetadata("partial_state").Instance().(*PartialState)
	ps.Hash = string(key)
	ps.ComputeProperties()
	return ps
}

var partialStateMetadata *datastore.EntityMetadataImpl

/*PartialStateProvider - a block summary instance provider */
func PartialStateProvider() datastore.Entity {
	ps := &PartialState{}
	ps.Version = "1.0"
	return ps
}

/*GetEntityMetadata - implement interface */
func (ps *PartialState) GetEntityMetadata() datastore.EntityMetadata {
	return partialStateMetadata
}

/*GetKey - implement interface */
func (ps *PartialState) GetKey() datastore.Key {
	return datastore.ToKey(ps.Hash)
}

/*SetKey - implement interface */
func (ps *PartialState) SetKey(key datastore.Key) {
	ps.Hash = datastore.ToString(key)
}

/*Read - store read */
func (ps *PartialState) Read(ctx context.Context, key datastore.Key) error {
	return ps.GetEntityMetadata().GetStore().Read(ctx, key, ps)
}

/*Write - store read */
func (ps *PartialState) Write(ctx context.Context) error {
	return ps.GetEntityMetadata().GetStore().Write(ctx, ps)
}

/*Delete - store read */
func (ps *PartialState) Delete(ctx context.Context) error {
	return ps.GetEntityMetadata().GetStore().Delete(ctx, ps)
}

//NewNodeDB - create a node db from the changes
func (ps *PartialState) newNodeDB() *util.MemoryNodeDB {
	mndb := util.NewMemoryNodeDB()
	for _, n := range ps.Nodes {
		mndb.PutNode(n.GetHashBytes(), n)
	}
	return mndb
}

//ComputeProperties - implement interface
func (ps *PartialState) ComputeProperties() {
	mndb := ps.newNodeDB()
	root := mndb.ComputeRoot()
	if root != nil {
		ps.mndb = mndb
		ps.root = root
	}
}

//Validate - implement interface
func (ps *PartialState) Validate(ctx context.Context) error {
	return ps.mndb.Validate(ps.root)
}

/*GetRoot - get the root of this set of changes */
func (ps *PartialState) GetRoot() util.Node {
	return ps.root
}

/*GetNodeDB - get the node db containing all the changes */
func (ps *PartialState) GetNodeDB() util.NodeDB {
	return ps.mndb
}

//UnmarshalJSON - implement Unmarshaler interface
func (ps *PartialState) UnmarshalJSON(data []byte) error {
	var obj map[string]interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		Logger.Error("unmarshal json - state change", zap.Error(err))
		return err
	}
	return ps.UnmarshalPartialState(obj)
}

//UnmarshalPartialState - unmarshal the partial state
func (ps *PartialState) UnmarshalPartialState(obj map[string]interface{}) error {
	if str, ok := obj["root"].(string); ok {
		ps.Hash = str
	} else {
		Logger.Error("unmarshal json - no hash", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	if str, ok := obj["version"].(string); ok {
		ps.Version = str
	} else {
		Logger.Error("unmarshal json - no version", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	if nodes, ok := obj["nodes"].([]interface{}); ok {
		ps.Nodes = make([]util.Node, len(nodes))
		for idx, nd := range nodes {
			if nd, ok := nd.(string); ok {
				buf, err := base64.StdEncoding.DecodeString(nd)
				if err != nil {
					Logger.Error("unmarshal json - state change", zap.Error(err))
					return err
				}
				ps.Nodes[idx], err = util.CreateNode(bytes.NewBuffer(buf))
				if err != nil {
					Logger.Error("unmarshal json - state change", zap.Error(err))
					return err
				}
			} else {
				Logger.Error("unmarshal json - invalid node", zap.Int("idx", idx), zap.Any("node", nd), zap.Any("obj", obj))
				return common.ErrInvalidData
			}
		}
	} else {
		Logger.Error("unmarshal json - no nodes", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	return nil
}

//MarshalJSON - implement Marshaler interface
func (ps *PartialState) MarshalJSON() ([]byte, error) {
	var data = make(map[string]interface{})
	return ps.MartialPartialState(data)
}

//MartialPartialState - martal the partial state
func (ps *PartialState) MartialPartialState(data map[string]interface{}) ([]byte, error) {
	data["root"] = ps.Hash
	data["version"] = ps.Version
	nodes := make([][]byte, len(ps.Nodes))
	for idx, nd := range ps.Nodes {
		nodes[idx] = nd.Encode()
	}
	data["nodes"] = nodes
	bytes, err := json.Marshal(data)
	if err != nil {
		Logger.Error("marshal JSON - state change", zap.String("block", ps.Hash), zap.Error(err))
	} else {
		Logger.Info("marshal JSON - state change", zap.String("block", ps.Hash))
	}
	return bytes, err
}

//AddNode - add node to the partial state
func (ps *PartialState) AddNode(node util.Node) {
	ps.Nodes = append(ps.Nodes, node)
}

