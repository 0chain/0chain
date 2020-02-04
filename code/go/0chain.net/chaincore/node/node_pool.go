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

	// single mutex for array and map
	mmx sync.RWMutex

	Nodes    []*Node          `json:"-"`
	NodesMap map[string]*Node `json:"nodes"`

	// stat (using sync/atomic)
	medianNetworkTime uint64 // float64
}

func (np *Pool) setMedianNetworkTime(val float64) {
	atomicStoreFloat64(&np.medianNetworkTime, val)
}

func (np *Pool) getMedianNetworkTime() float64 {
	return atomicLoadFloat64(&np.medianNetworkTime)
}

/*NewPool - create a new node pool of given type */
func NewPool(Type int8) *Pool {
	np := Pool{Type: Type}
	np.NodesMap = make(map[string]*Node)
	return &np
}

// without lock
func (np *Pool) size() int {
	return len(np.Nodes)
}

/*Size - size of the pool without regards to the node status */
func (np *Pool) Size() int {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	return np.size()
}

// without lock
func (np *Pool) addNode(node *Node) {
	if np.Type != node.Type {
		return
	}
	np.NodesMap[node.GetKey()] = node
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

	return np.NodesMap[id]
}

var none = make([]*Node, 0)

// returns shuffled copy of nodes array
// (with lock)
func (np *Pool) shuffleNodes() (shuffled []*Node) {
	shuffled = np.copyNodes() // <- lock is here
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return
}

/*GetActiveCount - get the active count */
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

/*GetRandomNodes - get a random set of nodes from the pool
* Doesn't consider active/inactive status
 */
func (np *Pool) GetRandomNodes(num int) []*Node {
	nodes := np.shuffleNodes() /// <- lock is here
	if num > len(nodes) {
		return nodes
	}
	return nodes[:num]
}

// copy of nodes array
// (uses lock)
func (np *Pool) copyNodes() (cp []*Node) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	cp = make([]*Node, np.size())
	copy(cp, np.Nodes)
	return
}

// Print - print this pool. This will be used for http
// response and Read method should be able to consume it
func (np *Pool) Print(w io.Writer) {
	nodes := np.shuffleNodes() // <- lock is here
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

// without locks
func (np *Pool) computeNodePositions() {
	sort.SliceStable(np.Nodes, func(i, j int) bool {
		return np.Nodes[i].GetKey() < np.Nodes[j].GetKey()
	})
	for idx, node := range np.Nodes {
		node.SetIndex = idx // TODO (kostyarin): async unsafe ?
	}
}

// without locks
func (np *Pool) computeNodesArray() {
	var array = make([]*Node, 0, len(np.NodesMap))
	for _, v := range np.NodesMap {
		array = append(array, v)
	}
	np.Nodes = array
	np.computeNodePositions()
}

/*ComputeProperties - compute properties after all the initialization of the node pool */
func (np *Pool) ComputeProperties() {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	np.computeNodesArray()
	for _, node := range np.Nodes {
		RegisterNode(node)
	}
}

/*GetNodesByLargeMessageTime - get the nodes in the node pool sorted by the time to send a large message */
func (np *Pool) GetNodesByLargeMessageTime() (sorted []*Node) {
	sorted = np.copyNodes()
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].getOptimalLargeMessageSendTime() < sorted[j].getOptimalLargeMessageSendTime()
	})
	return sorted
}

/*ComputeNetworkStats - compute the median time it takes for sending a large message to everyone in the network pool */
func (np *Pool) ComputeNetworkStats() {
	nodes := np.GetNodesByLargeMessageTime()
	var medianTime float64
	var count int
	for _, nd := range nodes {
		if nd == Self.Node {
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
	//
	np.setMedianNetworkTime(medianTime)
	mt := time.Duration(medianTime/1000000.) * time.Millisecond
	switch np.Type {
	case NodeTypeMiner:
		Self.Node.Info.MinersMedianNetworkTime = mt
	}
}

/*GetMedianNetworkTime - get the median network time for this pool */
func (np *Pool) GetMedianNetworkTime() float64 {
	return np.getMedianNetworkTime()
}

// Keys returns list of node ids of the map of the Pool.
func (np *Pool) Keys() (keys []string) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	keys = make([]string, 0, len(np.NodesMap))

	// TODO (kostyarin): can we use ` for k := range` instead of the GetKey?
	for _, node := range np.NodesMap {
		keys = append(keys, node.GetKey())
	}
	return
}

type NodeFunc func(*Node)

// ForEach performs given function for each node in map.
func (np *Pool) ForEach(nodeFunc NodeFunc) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	for _, node := range np.NodesMap {
		// release the mutex for iteration
		func(node *Node) {
			np.mmx.RUnlock()
			defer np.mmx.RLock()

			nodeFunc(node)
		}(node)
	}
}

type KeyNodeFunc func(key string, node *Node)

// ForEach performs given function for each node in map.
func (np *Pool) ForEachWithKey(keyNodeFunc KeyNodeFunc) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	// TODO (kostayrin): can we use node.GetKey() ?
	for key, node := range np.NodesMap {
		// release the mutex for the iteration
		func(key string, node *Node) {
			np.mmx.RUnlock()
			defer np.mmx.RLock()

			keyNodeFunc(key, node)
		}(key, node)
	}
}

// CopyMap returns copy of the nodes map.
func (np *Pool) CopyMap() (cp map[string]*Node) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	cp = make(map[string]*Node, len(np.NodesMap))
	for k, node := range np.NodesMap {
		cp[k] = node
	}
	return
}

// N2NURLs returns N2N URL bases of node of the map
func (np *Pool) N2NURLs() (urls []string) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	urls = make([]string, 0, len(np.NodesMap))
	for _, n := range np.NodesMap {
		urls = append(urls, n.GetN2NURLBase())
	}
	return
}

// CopyList returns copy of the nodes list.
func (np *Pool) CopyList() []*Node {
	return np.copyNodes()
}

// ForEachItem of nodes list (not map)
func (np *Pool) ForEachItem(nodeFunc NodeFunc) {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	for _, node := range np.Nodes {
		// release the mutex for the iteration
		func(node *Node) {
			np.mmx.RUnlock()
			defer np.mmx.RLock()

			nodeFunc(node)
		}(node)
	}
}

// ListSize returns list of nodes.
func (np *Pool) ListSize() int {
	np.mmx.RLock()
	defer np.mmx.RUnlock()

	return len(np.Nodes)
}
