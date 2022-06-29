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

var (
	// ErrPartialStateRootMismatch is returned when computed root does not match the partialState.Hash
	ErrPartialStateRootMismatch = errors.New("partial state root hash mismatch")
	// ErrMalformedPartialState is returned when detected the partialState.Nodes may have duplicate nodes
	ErrMalformedPartialState = errors.New("malformed partial state")
	// ErrPartialStateNilNodes is returned when partial state.Nodes slice is nil
	ErrPartialStateNilNodes = errors.New("partial state has no nodes")
)

//PartialState - an entity to exchange partial state
type PartialState struct {
	Hash      util.Key    `json:"root"`
	Version   string      `json:"version"`
	StartRoot util.Key    `json:"start"`
	Nodes     []util.Node `json:"-" msgpack:"-"`
	mndb      *util.MemoryNodeDB
	root      util.Node
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
func (ps *PartialState) GetScore() (int64, error) {
	return 0, nil
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
func (ps *PartialState) newNodeDB() (*util.MemoryNodeDB, error) {
	mndb := util.NewMemoryNodeDB()
	for _, n := range ps.Nodes {
		if err := mndb.PutNode(n.GetHashBytes(), n); err != nil {
			return nil, err
		}
	}
	return mndb, nil
}

// ComputeProperties - implement interface
func (ps *PartialState) ComputeProperties() error {
	if len(ps.Nodes) == 0 {
		return ErrPartialStateNilNodes
	}

	mnDB, err := ps.newNodeDB()
	if err != nil {
		return err
	}

	var (
		dbSize   = int(mnDB.Size(context.Background()))
		nodesNum = len(ps.Nodes)
	)

	if dbSize != nodesNum {
		logging.Logger.Error("malformed partial state, the db size must be the same as the nodes number",
			zap.Int("db size", dbSize),
			zap.Int("nodes num", nodesNum))

		return ErrMalformedPartialState
	}

	root, err := mnDB.ComputeRoot()
	if err != nil {
		logging.Logger.Error("partial state compute root failed", zap.Error(err))
		return err
	}

	if !bytes.Equal(root.GetHashBytes(), ps.Hash) {
		logging.Logger.Error("partial state root hash mismatch",
			zap.String("hash", util.ToHex(ps.Hash)),
			zap.String("root", util.ToHex(root.GetHashBytes())))
		return ErrPartialStateRootMismatch
	}

	ps.mndb = mnDB
	ps.root = root
	return nil
}

// Validate does nothing but to meet the Entity interface, the ComputeProperties
// have done the validation, and usually this function is called after the
// ComputeProperties
func (ps *PartialState) Validate(_ context.Context) error {
	return nil
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
			if node, ok := nd.(string); ok {
				buf, err := base64.StdEncoding.DecodeString(node)
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
				logging.Logger.Error("unmarshal json - invalid node", zap.Int("idx", idx), zap.String("node", node), zap.Any("obj", obj))
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
			if node, ok := nd.([]byte); ok {
				var err error
				ps.Nodes[idx], err = util.CreateNode(bytes.NewBuffer(node))
				if err != nil {
					logging.Logger.Error("unmarshal json - state change", zap.Error(err))
					return err
				}
			} else {
				logging.Logger.Error("unmarshal json - invalid node", zap.Int("idx", idx), zap.Any("node", node), zap.Any("obj", obj))
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
