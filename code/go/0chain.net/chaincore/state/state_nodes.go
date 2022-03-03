package state

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

//Nodes - a set of nodes for synching the state
type Nodes struct {
	datastore.IDField
	Version string      `json:"version"`
	Nodes   []util.Node `json:"-" msgpack:"-"`
}

//NewStateNodes - create a new partial state object with initialization
func NewStateNodes() *Nodes {
	ns := datastore.GetEntityMetadata("state_nodes").Instance().(*Nodes)
	ns.ComputeProperties()
	return ns
}

var nodesEntityMetadata *datastore.EntityMetadataImpl

/*NodesProvider - a block summary instance provider */
func NodesProvider() datastore.Entity {
	ns := &Nodes{}
	ns.Version = "1.0"
	//ps.SetKey(fmt.Sprintf("%v", time.Now().UnixNano()))
	return ns
}

/*GetEntityMetadata - implement interface */
func (ns *Nodes) GetEntityMetadata() datastore.EntityMetadata {
	return nodesEntityMetadata
}

/*Read - store read */
func (ns *Nodes) Read(ctx context.Context, key datastore.Key) error {
	return ns.GetEntityMetadata().GetStore().Read(ctx, key, ns)
}

/*Write - store read */
func (ns *Nodes) Write(ctx context.Context) error {
	return ns.GetEntityMetadata().GetStore().Write(ctx, ns)
}

/*Delete - store read */
func (ns *Nodes) Delete(ctx context.Context) error {
	return ns.GetEntityMetadata().GetStore().Delete(ctx, ns)
}

/*SetupStateNodes - setup the block summary entity */
func SetupStateNodes(store datastore.Store) {
	nodesEntityMetadata = datastore.MetadataProvider()
	nodesEntityMetadata.Name = "state_nodes"
	nodesEntityMetadata.Provider = NodesProvider
	nodesEntityMetadata.Store = store
	nodesEntityMetadata.IDColumnName = "id"
	datastore.RegisterEntityMetadata("state_nodes", nodesEntityMetadata)
}

//SaveState - save the partial state into another state db
func (ns *Nodes) SaveState(ctx context.Context, stateDB util.NodeDB) error {
	var keys []util.Key
	for _, nd := range ns.Nodes {
		keys = append(keys, nd.GetHashBytes())
	}
	return stateDB.MultiPutNode(keys, ns.Nodes)
}

//MarshalJSON - implement Marshaler interface
func (ns *Nodes) MarshalJSON() ([]byte, error) {
	data := ns.getMarshalFields()
	b, err := json.Marshal(data)
	if err != nil {
		logging.Logger.Error("marshal JSON - state nodes", zap.Error(err))
	} else {
		logging.Logger.Info("marshal JSON - state nodes", zap.Int("nodes", len(ns.Nodes)))
	}
	return b, err
}

//UnmarshalJSON - implement Unmarshaler interface
func (ns *Nodes) UnmarshalJSON(data []byte) error {
	var obj map[string]interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		logging.Logger.Error("unmarshal json - state nodes", zap.Error(err))
		return err
	}
	return ns.unmarshalStateNodesJSON(obj)
}

//unmarshalStateNodesJSON - unmarshal the partial state
func (ns *Nodes) unmarshalStateNodesJSON(obj map[string]interface{}) error {
	if str, ok := obj["version"].(string); ok {
		ns.Version = str
	} else {
		logging.Logger.Error("unmarshal json - no version", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	if nodes, ok := obj["nodes"].([]interface{}); ok {
		ns.Nodes = make([]util.Node, len(nodes))
		for idx, nd := range nodes {
			if nd, ok := nd.(string); ok {
				buf, err := base64.StdEncoding.DecodeString(nd)
				if err != nil {
					logging.Logger.Error("unmarshal json - state change", zap.Error(err))
					return err
				}
				ns.Nodes[idx], err = util.CreateNode(bytes.NewBuffer(buf))
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

func (ns *Nodes) MarshalMsgpack() ([]byte, error) {
	data := ns.getMarshalFields()
	b, err := msgpack.Marshal(data)
	if err != nil {
		logging.Logger.Error("marshal msgpack - state nodes", zap.Error(err))
		return nil, err
	}

	return b, nil
}

func (ns *Nodes) getMarshalFields() map[string]interface{} {
	data := make(map[string]interface{})
	data["version"] = ns.Version
	nodes := make([][]byte, len(ns.Nodes))
	for idx, nd := range ns.Nodes {
		nodes[idx] = nd.Encode()
	}
	data["nodes"] = nodes
	return data
}

//UnmarshalMsgpack - implement Unmarshaler interface
func (ns *Nodes) UnmarshalMsgpack(data []byte) error {
	var obj map[string]interface{}
	err := msgpack.Unmarshal(data, &obj)
	if err != nil {
		logging.Logger.Error("unmarshal msgpack - state nodes", zap.Error(err))
		return err
	}
	return ns.unmarshalStateNodesMsgpack(obj)
}

//unmarshalStateNodesMsgpack - unmarshal the partial state
func (ns *Nodes) unmarshalStateNodesMsgpack(obj map[string]interface{}) error {
	if str, ok := obj["version"].(string); ok {
		ns.Version = str
	} else {
		logging.Logger.Error("unmarshal msgpack - no version", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	if nodes, ok := obj["nodes"].([]interface{}); ok {
		ns.Nodes = make([]util.Node, len(nodes))
		for idx, nd := range nodes {
			if buf, ok := nd.([]byte); ok {
				var err error
				ns.Nodes[idx], err = util.CreateNode(bytes.NewBuffer(buf))
				if err != nil {
					logging.Logger.Error("unmarshal msgpack - state change", zap.Error(err))
					return err
				}
			} else {
				logging.Logger.Error("unmarshal msgpack - invalid node", zap.Int("idx", idx),
					zap.Any("node", nd), zap.Any("obj", obj))
				return common.ErrInvalidData
			}
		}
	} else {
		logging.Logger.Error("unmarshal msgpack - no nodes", zap.Any("obj", obj))
		return common.ErrInvalidData
	}
	return nil
}
