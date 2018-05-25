package node

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
)

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
	var nodeID = node.GetID()
	np.NodesMap[nodeID] = node
	np.computeNodesArray()
}

/*GetNode - given node id, get the node object or nil */
func (np *Pool) GetNode(id string) *Node {
	node, ok := np.NodesMap[id]
	if !ok {
		return nil
	}
	return node
}

/*RemoveNode - Remove a node from the pool */
func (np *Pool) RemoveNode(nodeID string) {
	if _, ok := np.NodesMap[nodeID]; !ok {
		return
	}
	delete(np.NodesMap, nodeID)
	np.computeNodesArray()
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
		if Self != nil && node.Equals(Self.Node) {
			continue
		}
		switch node.Type {
		case NodeTypeMiner:
			minerPool.AddNode(node)
		case NodeTypeSharder:
			sharderPool.AddNode(node)
		case NodeTypeBlobber:
			blobberPool.AddNode(node)
		default:
			panic(fmt.Sprintf("unkown node type %v:%v\n", node.GetID(), node.Type))
		}
	}
}
