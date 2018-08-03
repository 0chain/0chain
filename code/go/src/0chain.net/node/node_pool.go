package node

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"sort"

	"0chain.net/common"
	"0chain.net/datastore"
)

//ErrNodeNotFound - to indicate that a node is not present in the pool
var ErrNodeNotFound = common.NewError("node_not_found", "Requested node is not found")

/*Pool - a pool of nodes used for the same purpose */
type Pool struct {
	//Mutex &sync.Mutex{}
	Type     int
	Nodes    []*Node
	NodesMap map[string]*Node
}

/*NewPool - create a new node pool of given type */
func NewPool(Type int) *Pool {
	np := Pool{Type: Type}
	np.NodesMap = make(map[string]*Node)
	return &np
}

/*Size - size of the pool without regards to the node status */
func (np *Pool) Size() int {
	return len(np.Nodes)
}

/*AddNode - add a nodes to the pool */
func (np *Pool) AddNode(node *Node) {
	if np.Type != node.Type {
		return
	}
	var nodeID = datastore.ToString(node.GetKey())
	np.NodesMap[nodeID] = node
}

/*GetNode - given node id, get the node object or nil */
func (np *Pool) GetNode(id string) *Node {
	node, ok := np.NodesMap[id]
	if !ok {
		return nil
	}
	return node
}

var none = make([]*Node, 0)

func (np *Pool) shuffleNodes() []*Node {
	size := np.Size()
	if size == 0 {
		return none
	}
	shuffled := make([]*Node, size)
	perm := rand.Perm(size)
	for i, v := range perm {
		shuffled[v] = np.Nodes[i]
	}
	return shuffled
}

func (np *Pool) computeNodesArray() {
	// TODO: Do we need to use Mutex while doing this?
	var array = make([]*Node, 0, len(np.NodesMap))
	for _, v := range np.NodesMap {
		array = append(array, v)
	}
	np.Nodes = array
	np.computeNodePositions()
}

/*GetActiveCount - get the active count */
func (np *Pool) GetActiveCount() int {
	count := 0
	for _, node := range nodes {
		if node == Self.Node {
			continue
		}
		if node.Status == NodeStatusActive {
			count++
		}
	}
	return count
}

/*GetRandomNodes - get a random set of nodes from the pool
* Doesn't consider active/inactive status
 */
func (np *Pool) GetRandomNodes(num int) []*Node {
	var size = np.Size()
	if num > size {
		num = size
	}
	nodes := np.shuffleNodes()
	return nodes[:num]
}

/*Print - print this pool. This will be used for http response and Read method should be able to consume it*/
func (np *Pool) Print(w io.Writer) {
	nodes := np.shuffleNodes()
	for _, node := range nodes {
		if node.Status == NodeStatusActive {
			node.Print(w)
		}
	}
}

/*ReadNodes - read the pool information */
func ReadNodes(r io.Reader, minerPool *Pool, sharderPool *Pool, blobberPool *Pool) {
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
		case NodeTypeBlobber:
			blobberPool.AddNode(node)
		default:
			panic(fmt.Sprintf("unkown node type %v:%v\n", node.GetKey(), node.Type))
		}
	}
}

func (np *Pool) computeNodePositions() {
	sort.SliceStable(np.Nodes, func(i, j int) bool { return np.Nodes[i].GetKey() < np.Nodes[j].GetKey() })
	for idx, node := range np.Nodes {
		node.SetIndex = idx
	}
}

/*ComputeProperties - compute properties after all the initialization of the node pool */
func (np *Pool) ComputeProperties() {
	np.computeNodesArray()
	for _, node := range np.Nodes {
		RegisterNode(node)
	}
}
