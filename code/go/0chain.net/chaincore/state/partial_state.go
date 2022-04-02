package state

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

var ErrHashMismatch = errors.New("Root hash mistatch")

//PartialState - an entity to exchange partial state
type PartialState struct {
	Hash      util.Key    `json:"root"`
	Version   string      `json:"version"`
	StartRoot util.Key    `json:"start"`
	Nodes     []util.Node `json:"-" msgpack:"-"`
	mndb      *util.MemoryNodeDB
	root      util.Node
}

//NewPartialState - create a new partial state object with initialization
func NewPartialState(key util.Key) *PartialState {
	ps := datastore.GetEntityMetadata("partial_state").Instance().(*PartialState)
	ps.Hash = key
	ps.ComputeProperties()
	return ps
}

var partialStateEntityMetadata *datastore.EntityMetadataImpl

/*PartialStateProvider - a block summary instance provider */
func PartialStateProvider() datastore.Entity {
	ps := &PartialState{}
	ps.Version = "1.0"
	return ps
}

/*GetEntityMetadata - implement interface */
func (ps *PartialState) GetEntityMetadata() datastore.EntityMetadata {
	return partialStateEntityMetadata
}

/*GetKey - implement interface */
func (ps *PartialState) GetKey() datastore.Key {
	return datastore.ToKey(ps.Hash)
}

/*SetKey - implement interface */
func (ps *PartialState) SetKey(key datastore.Key) {
	skey := datastore.ToString(key)
	bkey, err := hex.DecodeString(skey)
	if err == nil {
		ps.Hash = bkey
	} else {
		ps.Hash = []byte(skey)
	}
}

/*Read - store read */
func (ps *PartialState) Read(ctx context.Context, key datastore.Key) error {
	return ps.GetEntityMetadata().GetStore().Read(ctx, key, ps)
}

/*GetScore - score for write*/
func (ps *PartialState) GetScore() int64 {
	return 0
}

/*Write - store read */
func (ps *PartialState) Write(ctx context.Context) error {
	return ps.GetEntityMetadata().GetStore().Write(ctx, ps)
}

/*Delete - store read */
func (ps *PartialState) Delete(ctx context.Context) error {
	return ps.GetEntityMetadata().GetStore().Delete(ctx, ps)
}

/*SetupPartialState - setup the block summary entity */
func SetupPartialState(store datastore.Store) {
	partialStateEntityMetadata = datastore.MetadataProvider()
	partialStateEntityMetadata.Name = "partial_state"
	partialStateEntityMetadata.Provider = PartialStateProvider
	partialStateEntityMetadata.Store = store
	partialStateEntityMetadata.IDColumnName = "hash"
	datastore.RegisterEntityMetadata("partial_state", partialStateEntityMetadata)
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
		if bytes.Equal(root.GetHashBytes(), ps.Hash) {
			ps.mndb = mndb
			ps.root = root
		} else {
			logging.Logger.Error("partial state root hash mismatch", zap.Any("hash", ps.Hash), zap.Any("root", root.GetHashBytes()))
		}
	} else {
		logging.Logger.Error("partial state root is null", zap.Int("nodes", len(ps.Nodes)))
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

//MarshalJSON - implement Marshaler interface
func (ps *PartialState) MarshalJSON() ([]byte, error) {
	var data = make(map[string]interface{})
	return ps.MarshalPartialStateJSON(data)
}

func (ps *PartialState) MarshalMsgpack() ([]byte, error) {
	data := make(map[string]interface{})
	return ps.MarshalPartialStateMsgpack(data)
}

//UnmarshalJSON - implement Unmarshaler interface
func (ps *PartialState) UnmarshalJSON(data []byte) error {
	var obj map[string]interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		logging.Logger.Error("unmarshal json - state change", zap.Error(err))
		return err
	}
	return ps.UnmarshalPartialStateJSON(obj)
}

// UnmarshalMsgpack implements Unmarshaler interface
func (ps *PartialState) UnmarshalMsgpack(data []byte) error {
	obj := make(map[string]interface{})
	err := msgpack.Unmarshal(data, &obj)
	if err != nil {
		logging.Logger.Error("unmarshal json - state change", zap.Error(err))
		return err
	}
	return ps.UnmarshalPartialStateMsgpack(obj)
}

//UnmarshalPartialStateJSON - unmarshal the partial state
func (ps *PartialState) UnmarshalPartialStateJSON(obj map[string]interface{}) error {
	if root, ok := obj["root"]; ok {
		switch rootImpl := root.(type) {
		case string:
			ps.SetKey(rootImpl)
		case []byte:
			ps.Hash = rootImpl
		default:
			logging.Logger.Error("unmarshal json - unknown type", zap.Any("obj", obj))
		}
	} else {
		logging.Logger.Error("unmarshal json - no hash", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	if str, ok := obj["version"].(string); ok {
		ps.Version = str
	} else {
		logging.Logger.Error("unmarshal json - no version", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	if nodes, ok := obj["nodes"].([]interface{}); ok {
		ps.Nodes = make([]util.Node, len(nodes))
		for idx, nd := range nodes {
			if nd, ok := nd.(string); ok {
				buf, err := base64.StdEncoding.DecodeString(nd)
				if err != nil {
					logging.Logger.Error("unmarshal json - state change", zap.Error(err))
					return err
				}
				ps.Nodes[idx], err = util.CreateNode(bytes.NewBuffer(buf))
				if err != nil {
					logging.Logger.Error("unmarshal json - state change", zap.Error(err))
					return err
				}
			} else {
				logging.Logger.Error("unmarshal json - invalid node", zap.Int("idx", idx), zap.Any("node", nd), zap.Any("obj", obj))
				return common.ErrInvalidData
			}
		}
	} else {
		logging.Logger.Error("unmarshal json - no nodes", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	return nil
}

//UnmarshalPartialStateMsgpack - unmarshal the partial state
func (ps *PartialState) UnmarshalPartialStateMsgpack(obj map[string]interface{}) error {
	if root, ok := obj["root"]; ok {
		switch rootImpl := root.(type) {
		case string:
			ps.SetKey(rootImpl)
		default:
			logging.Logger.Error("unmarshal json - unknown type", zap.Any("obj", obj))
		}
	} else {
		logging.Logger.Error("unmarshal json - no hash", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	if str, ok := obj["version"].(string); ok {
		ps.Version = str
	} else {
		logging.Logger.Error("unmarshal json - no version", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	if nodes, ok := obj["nodes"].([]interface{}); ok {
		ps.Nodes = make([]util.Node, len(nodes))
		for idx, nd := range nodes {
			if nd, ok := nd.([]byte); ok {
				var err error
				ps.Nodes[idx], err = util.CreateNode(bytes.NewBuffer(nd))
				if err != nil {
					logging.Logger.Error("unmarshal json - state change", zap.Error(err))
					return err
				}
			} else {
				logging.Logger.Error("unmarshal json - invalid node", zap.Int("idx", idx), zap.Any("node", nd), zap.Any("obj", obj))
				return common.ErrInvalidData
			}
		}
	} else {
		logging.Logger.Error("unmarshal json - no nodes", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	return nil
}

//MarshalPartialStateJSON - martal the partial state
func (ps *PartialState) MarshalPartialStateJSON(data map[string]interface{}) ([]byte, error) {
	data = ps.setMarshalFields(data)
	b, err := json.Marshal(data)
	if err != nil {
		logging.Logger.Error("marshal JSON - state change", zap.Any("block", ps.Hash), zap.Error(err))
	} else {
		logging.Logger.Info("marshal JSON - state change", zap.Any("block", ps.Hash))
	}
	return b, err
}

func (ps *PartialState) MarshalPartialStateMsgpack(data map[string]interface{}) ([]byte, error) {
	data = ps.setMarshalFields(data)
	b, err := msgpack.Marshal(data)
	if err != nil {
		logging.Logger.Error("marshal Msgpack - partial state", zap.Any("block", ps.Hash), zap.Error(err))
		return nil, err
	}

	return b, nil
}

func (ps *PartialState) setMarshalFields(data map[string]interface{}) map[string]interface{} {
	data["root"] = util.ToHex(ps.Hash)
	data["version"] = ps.Version
	nodes := make([][]byte, len(ps.Nodes))
	for idx, nd := range ps.Nodes {
		nodes[idx] = nd.Encode()
	}
	data["nodes"] = nodes
	return data
}

//AddNode - add node to the partial state
func (ps *PartialState) AddNode(node util.Node) {
	ps.Nodes = append(ps.Nodes, node)
}

//SaveState - save the partial state into another state db
func (ps *PartialState) SaveState(ctx context.Context, stateDB util.NodeDB) error {
	return util.MergeState(ctx, ps.mndb, stateDB)
}

// GetNodeDB returns the node db containing all the changes
func (ps *PartialState) GetNodeDB() util.NodeDB {
	return ps.mndb
}
