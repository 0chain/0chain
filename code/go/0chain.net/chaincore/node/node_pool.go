package node

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/logging"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

//ErrNodeNotFound - to indicate that a node is not present in the pool
var ErrNodeNotFound = common.NewError("node_not_found", "Requested node is not found")

func atomicLoadFloat64(addr *uint64) float64 {
	return math.Float64frombits(atomic.LoadUint64(addr))
}

func atomicStoreFloat64(addr *uint64, val float64) {
	atomic.StoreUint64(addr, math.Float64bits(val))
}

/*Pool - a pool of nodes used for the same purpose */
type Pool struct {
	Type int8 `json:"type"`

	// ---------------------------------------------
	mmx      sync.RWMutex     `json:"-" msgpack:"-"`
	Nodes    []*Node          `json:"-" msgpack:"-"`
	NodesMap map[string]*Node `json:"nodes"`
	// ---------------------------------------------

	medianNetworkTime uint64                     // float64
	getNodesC         chan []*Node               `json:"-"`
	updateNodesC      chan *updateNodesWithReply `json:"-"`
	startOnce         *sync.Once                 `json:"-"`
	id                int32                      `json:"-"`
}

/*NewPool - create a new node pool of given type */
func NewPool(Type int8) *Pool {
	p := &Pool{
		Type:      Type,
		NodesMap:  make(map[string]*Node),
		startOnce: &sync.Once{},
	}

	p.initGetNodesC()

	p.start()

	return p
}

func (np *Pool) initGetNodesC() {
	if np.updateNodesC == nil {
		np.updateNodesC = make(chan *updateNodesWithReply, 100)
	}

	if np.getNodesC == nil {
		np.getNodesC = make(chan []*Node)
	}
}

var poolID int32

func (np *Pool) start() {
	np.startOnce.Do(func() {
		go func() {
			np.id = atomic.AddInt32(&poolID, 1)

			var nds []*Node

			// return if there's no request to get nodes from this pool in one minutes
			const expireTime = time.Minute
			cleanTimer := time.NewTimer(expireTime)

			for {
				select {
				case np.getNodesC <- nds:
					cleanTimer = time.NewTimer(expireTime)
				case v := <-np.updateNodesC:
					nds = v.nodes
					v.reply <- struct{}{}
					cleanTimer = time.NewTimer(expireTime)
				case <-cleanTimer.C:
					np.startOnce = &sync.Once{}
					return
				}
			}
		}()
	})
}

func (np *Pool) getNodesFromC() (nds []*Node) {
	i := 0
	for {
		np.start()
		select {
		case nds = <-np.getNodesC:
			return
		case <-time.After(500 * time.Millisecond):
			logging.Logger.Warn("get nodes timeout",
				zap.Int32("ID", np.id),
				zap.Int("retry", i))
			i++
			continue
		}
	}
}

type updateNodesWithReply struct {
	nodes []*Node
	reply chan struct{}
}

func (np *Pool) updateNodesToC(nds []*Node) {
	np.start()

	ndsWithReply := &updateNodesWithReply{
		nodes: nds,
		reply: make(chan struct{}, 1),
	}

	select {
	case np.updateNodesC <- ndsWithReply:
		<-ndsWithReply.reply
	case <-time.After(500 * time.Millisecond):
		logging.Logger.Warn("update nodes to channel timeout")
	}
	return
}

/*Size - size of the pool regardless node status */
func (np *Pool) Size() int {
	nds := np.getNodesFromC()

	return len(nds)
}

// AddNode - add a node to the pool
func (np *Pool) AddNode(node *Node) {
	if np.Type != node.Type {
		return
	}

	node.SetPublicKey(node.PublicKey)
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
	np.updateNodesToC(np.Nodes)
	np.mmx.Unlock()
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
	nds := np.getNodesFromC()

	for _, node := range nds {
		if node.IsActive() {
			count++
		}
	}
	return
}

// GetNodesByLargeMessageTime - get the nodes in the node pool sorted by the
// time to send a large message
func (np *Pool) GetNodesByLargeMessageTime() (sorted []*Node) {
	sorted = np.getNodesFromC()
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].getOptimalLargeMessageSendTime() <
			sorted[j].getOptimalLargeMessageSendTime()
	})

	return
}

func (np *Pool) shuffleNodes(preferPrevMBNodes bool) []*Node {
	ts := time.Now()
	defer func() {
		du := time.Since(ts)
		if du > time.Second {
			logging.Logger.Debug("shuffleNodes takes too long", zap.Any("duration", du))
		}
	}()

	shuffled := np.getNodesFromC()
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

// Print - print this pool. This will be used for http response and Read method
// should be able to consume it
func (np *Pool) Print(w io.Writer) {
	nodes := np.shuffleNodes(false)
	for _, node := range nodes {
		if node.IsActive() {
			node.Print(w)
		}
	}
}

/*ReadNodes - read the pool information */
func ReadNodes(r io.Reader, minerPool *Pool, sharderPool *Pool) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		node, err := Read(line)
		if err != nil {
			panic(err)
		}
		switch node.Type {
		case NodeTypeMiner:
			minerPool.AddNode(node)
		case NodeTypeSharder:
			sharderPool.AddNode(node)
		default:
			panic(fmt.Sprintf("unkown node type %v:%v\n", node.GetKey(), node.Type))
		}
	}
}

/*AddNodes - add nodes to the node pool */
func (np *Pool) AddNodes(nodes []interface{}) {
	for _, nci := range nodes {
		nc, ok := nci.(map[interface{}]interface{})
		if !ok {
			continue
		}
		nc["type"] = np.Type
		nd, err := NewNode(nc)
		if err != nil {
			panic(err)
		}
		np.AddNode(nd)
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
	nds := np.getNodesFromC()
	n2n = make([]string, 0, len(nds))
	for _, node := range nds {
		n2n = append(n2n, node.GetN2NURLBase())
	}
	return
}

// CopyNodes list.
func (np *Pool) CopyNodes() (list []*Node) {
	nds := np.getNodesFromC()
	if len(nds) == 0 {
		return
	}

	list = make([]*Node, len(nds))
	copy(list, np.Nodes)
	return
}

// CopyNodesMap returns copy of underlying map.
func (np *Pool) CopyNodesMap() (nodesMap map[string]*Node) {
	nds := np.getNodesFromC()
	nodesMap = make(map[string]*Node, len(nds))
	for i, n := range nds {
		nodesMap[n.GetKey()] = nds[i]
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
	nds := np.getNodesFromC()
	keys = make([]string, 0, len(nds))
	for _, n := range nds {
		keys = append(keys, n.GetKey())
	}
	return
}

// NewNodes returns list of nodes exist in
// given Pool, but don't exist in this pool.
func (np *Pool) NewNodes(newPool *Pool) (newNodes []*Node) {

	var (
		nps      = np.CopyNodesMap()
		newPools = newPool.CopyNodesMap()
	)

	for id, node := range newPools {
		if _, ok := nps[id]; !ok {
			newNodes = append(newNodes, node)
		}
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
		clone.AddNode(v.Clone())
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
			n.SetPublicKey(n.PublicKey)
		}
		np.Nodes = append(np.Nodes, n)
	}

	np.initGetNodesC()
	np.computeNodePositions()
	np.startOnce = &sync.Once{}
	np.updateNodesToC(np.Nodes)

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
			n.SetPublicKey(n.PublicKey)
		}
		np.Nodes = append(np.Nodes, n)
	}

	np.initGetNodesC()
	np.computeNodePositions()
	np.startOnce = &sync.Once{}
	np.updateNodesToC(np.Nodes)
	return nil
}
