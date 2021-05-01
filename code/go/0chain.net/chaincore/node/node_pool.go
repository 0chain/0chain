package node

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/core/common"
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
	mmx      sync.RWMutex
	Nodes    []*Node          `json:"-"`
	NodesMap map[string]*Node `json:"nodes"`
	// ---------------------------------------------

	medianNetworkTime uint64 // float64
}

/*NewPool - create a new node pool of given type */
func NewPool(Type int8) *Pool {
	return &Pool{
		Type:     Type,
		NodesMap: make(map[string]*Node),
	}
}

/*Size - size of the pool regardless node status */
func (np *Pool) Size() int {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	return len(np.Nodes)
}

// MapSize returns number of nodes added to the pool.
func (np *Pool) MapSize() int {
	np.mmx.RLock()
	defer np.mmx.RUnlock()
	return len(np.NodesMap)
}

/*AddNode - add a nodes to the pool */
func (np *Pool) AddNode(node *Node) {
	if np.Type != node.Type {
		return
	}

	np.mmx.Lock()
	defer np.mmx.Unlock()

	np.NodesMap[node.GetKey()] = node
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

var none = make([]*Node, 0)

// TODO: refactor to return a copy of Nodes instead of the pointers
func (np *Pool) shuffleNodes() (shuffled []*Node) {
	shuffled = np.Nodes
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return
}

func (np *Pool) computeNodesArray() {
	var array = make([]*Node, 0, len(np.NodesMap))
	for _, v := range np.NodesMap {
		array = append(array, v)
	}
	np.Nodes = array
	np.computeNodePositions()
}

// GetActiveCount returns the active count
func (np *Pool) GetActiveCount() (count int) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	for _, node := range np.Nodes {
		if node.IsActive() {
			count++
		}
	}
	return
}

// GetRandomNodes returns a random set of nodes from the pool
// Doesn't consider active/inactive status
func (np *Pool) GetRandomNodes(num int) []*Node {
	np.mmx.Lock()
	defer np.mmx.Unlock()
	nodes := np.shuffleNodes()
	if num > len(nodes) {
		num = len(nodes)
	}
	return nodes[:num]
}

/*GetNodesByLargeMessageTime - get the nodes in the node pool sorted by the
time to send a large message */
func (np *Pool) GetNodesByLargeMessageTime() (sorted []*Node) {
	np.mmx.Lock()
	defer np.mmx.Unlock()
	sorted = np.Nodes
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].getOptimalLargeMessageSendTime() <
			sorted[j].getOptimalLargeMessageSendTime()
	})

	return
}

func (np *Pool) getNodesByLargeMessageTime() (sorted []*Node) {
	sorted = np.Nodes
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].getOptimalLargeMessageSendTime() <
			sorted[j].getOptimalLargeMessageSendTime()
	})
	return
}

func (np *Pool) shuffleNodesLock() []*Node {
	np.mmx.Lock()
	defer np.mmx.Unlock()
	return np.shuffleNodes()
}

/*Print - print this pool. This will be used for http response and Read method
should be able to consume it*/
func (np *Pool) Print(w io.Writer) {
	nodes := np.shuffleNodesLock()
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

/*ComputeProperties - compute properties after all the initialization of the node pool */
func (np *Pool) ComputeProperties() {
	np.mmx.Lock()
	defer np.mmx.Unlock()
	np.computeNodesArray()
	for _, node := range np.Nodes {
		RegisterNode(node)
	}
}

/*ComputeNetworkStats - compute the median time it takes for sending a large message to everyone in the network pool */
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

func (np *Pool) N2NURLs() (n2n []string) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()
	n2n = make([]string, 0, len(np.NodesMap))
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
	for k, v := range np.NodesMap {
		nodesMap[k] = v
	}

	return
}

// HasNode returns true if node with given key exists in the pool's map.
func (np *Pool) HasNode(key string) (ok bool) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()
	_, ok = np.NodesMap[key]
	return
}

// Keys of all nods of the pool's map.
func (np *Pool) Keys() (keys []string) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()
	keys = make([]string, 0, len(np.NodesMap))
	for k := range np.NodesMap {
		keys = append(keys, k)
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

	for k, v := range np.NodesMap {
		nv := v.Clone()
		clone.NodesMap[k] = nv
	}

	clone.computeNodesArray()

	return clone
}
