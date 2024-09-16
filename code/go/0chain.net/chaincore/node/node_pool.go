package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/core/common"
	"github.com/0chain/common/core/logging"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

//msgp:ignore Pool
//go:generate msgp -v -io=false -tests=false -unexported

// ErrNodeNotFound - to indicate that a node is not present in the pool
var ErrNodeNotFound = common.NewError("node_not_found", "Requested node is not found")

func atomicLoadFloat64(addr *uint64) float64 {
	return math.Float64frombits(atomic.LoadUint64(addr))
}

func atomicStoreFloat64(addr *uint64, val float64) {
	atomic.StoreUint64(addr, math.Float64bits(val))
}

/*Pool - a pool of nodes used for the same purpose */
type Pool struct {
	Type NodeType `json:"type"`

	// ---------------------------------------------
	mmx      sync.RWMutex     `json:"-" msgpack:"-" msg:"-"`
	Nodes    []*Node          `json:"-" msgpack:"-" msg:"-"`
	NodesMap map[string]*Node `json:"nodes"`
	// ---------------------------------------------

	medianNetworkTime uint64 `msg:"-"` // float64
}

/*NewPool - create a new node pool of given type */
func NewPool(Type NodeType) *Pool {
	p := &Pool{
		Type:     Type,
		NodesMap: make(map[string]*Node),
		Nodes:    []*Node{},
	}

	return p
}

/*Size - size of the pool regardless node status */
func (np *Pool) Size() int {
	np.mmx.RLock()
	defer np.mmx.RUnlock()
	return len(np.NodesMap)
}

// AddNode - add a node to the pool
func (np *Pool) AddNode(node *Node) error {
	if np.Type != node.Type {
		logging.Logger.Error("incorrect node type",
			zap.String("node_type", node.Type.String()),
			zap.String("pool_type", np.Type.String()))
		return errors.New("incorrect node type")
	}

	if err := node.SetPublicKey(node.PublicKey); err != nil {
		return fmt.Errorf("invalid public key, %v", err)
	}

	RegisterNode(node)

	np.mmx.Lock()
	_, ok := np.NodesMap[node.GetKey()]
	if !ok {
		np.Nodes = append(np.Nodes, node)
	} else {
		// node exist, replace with new one in the pool
		for i, nd := range np.Nodes {
			if nd.GetKey() == node.GetKey() {
				np.Nodes[i] = node
				break
			}
		}
	}

	np.NodesMap[node.GetKey()] = node
	np.computeNodePositions()
	np.mmx.Unlock()

	return nil
}

func (np *Pool) Delete(key string) {
	np.mmx.Lock()
	defer np.mmx.Unlock()
	delete(np.NodesMap, key)
	var idx int
	for i, n := range np.Nodes {
		if n.GetKey() == key {
			idx = i
			break
		}
	}

	np.Nodes[idx] = np.Nodes[len(np.Nodes)-1]
	np.Nodes = np.Nodes[:len(np.Nodes)-1]
	np.computeNodePositions()
	// TODO: remove from global nodes map
}

/*GetNode - given node id, get the node object or nil */
func (np *Pool) GetNode(id string) *Node {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	node, ok := np.NodesMap[id]
	if !ok {
		return nil
	}
	return node
}

// GetActiveCount returns the active count
func (np *Pool) GetActiveCount() (count int) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	for _, node := range np.NodesMap {
		if node.IsActive() {
			count++
		}
	}
	return
}

// GetNodesByLargeMessageTime - get the nodes in the node pool sorted by the
// time to send a large message
func (np *Pool) GetNodesByLargeMessageTime() (sorted []*Node) {
	np.mmx.RLock()
	for _, v := range np.NodesMap {
		sorted = append(sorted, v)
	}
	np.mmx.RUnlock()

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].getOptimalLargeMessageSendTime() <
			sorted[j].getOptimalLargeMessageSendTime()
	})

	return
}

func (np *Pool) ShuffleNodes(preferPrevMBNodes bool) (shuffled []*Node) {
	np.mmx.RLock()
	for _, v := range np.NodesMap {
		shuffled = append(shuffled, v)
	}
	defer np.mmx.RUnlock()

	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	if preferPrevMBNodes {
		sort.SliceStable(shuffled, func(i, j int) bool {
			return shuffled[i].InPrevMB
		})
	}

	return shuffled
}

// Print - print this pool. This will be used for http response and read method
// should be able to consume it
func (np *Pool) Print(w io.Writer) {
	nodes := np.ShuffleNodes(false)
	for _, node := range nodes {
		if node.IsActive() {
			node.Print(w)
		}
	}
}

func (np *Pool) computeNodePositions() {

	sort.SliceStable(np.Nodes, func(i, j int) bool {
		return np.Nodes[i].GetKey() < np.Nodes[j].GetKey()
	})
	for idx, node := range np.Nodes {
		node.SetIndex = idx
	}
}

// ComputeNetworkStats - compute the median time it takes for sending a large message to everyone in the network pool */
func (np *Pool) ComputeNetworkStats() {
	nodes := np.GetNodesByLargeMessageTime()
	var medianTime float64
	var count int
	for _, nd := range nodes {
		if Self.IsEqual(nd) {
			continue
		}
		if !nd.IsActive() {
			continue
		}
		count++
		if count*2 >= len(nodes) {
			medianTime = nd.getOptimalLargeMessageSendTime()
			break
		}
	}
	atomicStoreFloat64(&np.medianNetworkTime, medianTime)
	mt := time.Duration(medianTime/1000000.) * time.Millisecond
	switch np.Type {
	case NodeTypeMiner:
		info := Self.Underlying().GetNodeInfo()
		info.MinersMedianNetworkTime = mt
		Self.Underlying().SetNodeInfo(&info)
	}
}

/*GetMedianNetworkTime - get the median network time for this pool */
func (np *Pool) GetMedianNetworkTime() float64 {
	return atomicLoadFloat64(&np.medianNetworkTime)
}

// N2NURLs returns the urls of all nodes in the pool
func (np *Pool) N2NURLs() (n2n []string) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()
	for _, node := range np.NodesMap {
		n2n = append(n2n, node.GetN2NURLBase())
	}
	return
}

// CopyNodes list.
func (np *Pool) CopyNodes() (list []*Node) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()
	if len(np.Nodes) == 0 {
		return
	}

	list = make([]*Node, len(np.Nodes))
	copy(list, np.Nodes)
	return
}

// CopyNodesMap returns copy of underlying map.
func (np *Pool) CopyNodesMap() (nodesMap map[string]*Node) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	nodesMap = make(map[string]*Node, len(np.NodesMap))
	for i, n := range np.NodesMap {
		nodesMap[n.GetKey()] = np.NodesMap[i]
	}

	return
}

// HasNode returns true if node with given key exists in the pool's map.
func (np *Pool) HasNode(key string) (ok bool) {
	np.mmx.RLock()
	_, ok = np.NodesMap[key]
	np.mmx.RUnlock()
	return
}

// Keys of all nods of the pool's map.
func (np *Pool) Keys() (keys []string) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	for _, n := range np.NodesMap {
		keys = append(keys, n.GetKey())
	}
	return
}

// Clone returns a clone of Pool instance
func (np *Pool) Clone() *Pool {
	np.mmx.RLock()
	defer np.mmx.RUnlock()
	clone := NewPool(np.Type)
	clone.Nodes = make([]*Node, 0, len(np.Nodes))
	clone.NodesMap = make(map[string]*Node, len(np.NodesMap))
	clone.medianNetworkTime = np.medianNetworkTime

	for _, v := range np.NodesMap {
		if err := clone.AddNode(v.Clone()); err != nil {
			logging.Logger.Warn("pool clone - add cloned node failed", zap.Error(err))
		}
	}

	return clone
}

// UnmarshalJSON implements the json decoding for the pool
func (np *Pool) UnmarshalJSON(data []byte) error {
	type Alias Pool
	var v = struct {
		*Alias
	}{
		Alias: (*Alias)(np),
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	np.Nodes = make([]*Node, 0, len(np.NodesMap))
	for k := range np.NodesMap {
		n := np.NodesMap[k]
		if n.SigScheme == nil {
			if err := n.SetPublicKey(n.PublicKey); err != nil {
				return err
			}
		}

		n.TimersByURI = make(map[string]metrics.Timer, 10)
		n.SizeByURI = make(map[string]metrics.Histogram, 10)
		n.setupCommChannel()
		np.Nodes = append(np.Nodes, n)
	}

	np.computeNodePositions()

	return nil
}

var _ msgpack.CustomDecoder = (*Pool)(nil)

// DecodeMsgpack implements custome decoder for msgpack
// to initialize variables in the Pool
func (np *Pool) DecodeMsgpack(dec *msgpack.Decoder) error {
	type Alias Pool
	var v = struct {
		*Alias
	}{
		Alias: (*Alias)(np),
	}

	if err := dec.Decode(&v); err != nil {
		return err
	}

	np.Nodes = make([]*Node, 0, len(np.NodesMap))
	for k := range np.NodesMap {
		n := np.NodesMap[k]
		if n.SigScheme == nil {
			if err := n.SetPublicKey(n.PublicKey); err != nil {
				return err
			}
		}
		np.Nodes = append(np.Nodes, n)
	}

	np.computeNodePositions()
	return nil
}

func (np *Pool) MarshalMsg(o []byte) ([]byte, error) {
	d := poolDecode(*np) //nolint: govet
	return d.MarshalMsg(o)
}

func (np *Pool) UnmarshalMsg(b []byte) ([]byte, error) {
	d := &poolDecode{}
	o, err := d.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	np.Nodes = make([]*Node, 0, len(d.NodesMap))
	np.NodesMap = make(map[string]*Node, len(d.NodesMap))
	np.Type = d.Type
	for k := range d.NodesMap {
		n := d.NodesMap[k]
		if n.SigScheme == nil {
			if err := n.SetPublicKey(n.PublicKey); err != nil {
				return nil, err
			}
		}
		np.Nodes = append(np.Nodes, n)
		np.NodesMap[k] = n
	}

	np.computeNodePositions()
	return o, nil
}

func (np *Pool) Msgsize() int {
	d := poolDecode(*np) //nolint: govet
	return d.Msgsize()
}

type poolDecode Pool
