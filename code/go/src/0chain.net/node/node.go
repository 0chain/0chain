package node

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/encryption"
)

/*Node - a struct holding the node information */
type Node struct {
	Host           string
	Port           int
	Type           int
	Status         int
	LastActiveTime common.Time
	ID             string
	PublicKey      string
	ErrorCount     int
}

/*SelfNode -- self node type*/
type SelfNode struct {
	*Node
	privateKey string
}

func (sn *SelfNode) SetPrivateKey(privateKey string) {
	sn.privateKey = privateKey
}

var Self *SelfNode

/*GetID - get the id of the node */
func (n *Node) GetID() string {
	return n.ID
}

/*Print - print node's info that is consumable by Read */
func (n *Node) Print(w io.Writer) {
	fmt.Printf("%v,%v,%v,%v,%v\n", n.GetNodeType(), n.Host, n.Port, n.GetID(), n.PublicKey)
}

func Read(line string) (*Node, error) {
	node := &Node{}
	fields := strings.Split(line, ",")
	if len(fields) != 5 {
		return nil, common.NewError("invalid_num_fields", fmt.Sprintf("invalid number of fields [%v]", line))
	}
	switch fields[0] {
	case "m":
		node.Type = NodeTypeMiner
	case "s":
		node.Type = NodeTypeSharder
	case "b":
		node.Type = NodeTypeBlobber
	default:
		return nil, common.NewError("unknown_node_type", fmt.Sprintf("Unkown node type %v", fields[0]))
	}
	node.Host = fields[1]
	port, err := strconv.ParseInt(fields[2], 10, 32)
	if err != nil {
		return nil, err
	}
	node.Port = int(port)
	node.ID = fields[3]
	node.PublicKey = fields[4]
	if node.Host == config.Configuration.Host && node.Port == config.Configuration.Port {
		Self = &SelfNode{Node: node}
	}
	return node, nil
}

/*GetStatusURL - get the end point where to ping for the status */
func (n *Node) GetStatusURL() string {
	host := n.Host
	if host == "" {
		if n.Port != Self.Port {
			host = Self.Host
			if host == "" {
				host = "localhost"
			}
		} else {
			panic(fmt.Sprintf("invalid node setup for %v\n", n.GetID()))
		}
	}
	return fmt.Sprintf("http://%v:%v/_nh/status?id=%v&publicKey=%v", n.Host, n.Port, n.ID, n.PublicKey)
}

func (n *Node) TimeStampSignature(privateKey string) (string, string, string, error) {
	data := fmt.Sprintf("%v:%v", n.ID, common.Now())
	hash := encryption.Hash(data)
	signature, err := encryption.Sign(privateKey, hash)
	if err != nil {
		return "", "", "", err
	} else {
		return data, hash, signature, err
	}
}

func (n *Node) Verify(ts common.Time, data string, hash string, signature string) (bool, error) {
	// TODO: Ensure time is within 3 seconds  and hash and signature match using n.PublicKey using encryption.Verify()
	return true, nil
}

/*GetNodeType - as a string */
func (n *Node) GetNodeType() string {
	switch n.Type {
	case NodeTypeMiner:
		return "m"
	case NodeTypeSharder:
		return "s"
	case NodeTypeBlobber:
		return "b"
	default:
		return "u"
	}
}

var (
	NodeTypeMiner   = 1
	NodeTypeSharder = 2
	NodeTypeBlobber = 3
)

var (
	NodeStatusActive   = 1
	NodeStatusInactive = 2
)

/*NodePool - a pool of nodes used for the same purpose */
type NodePool struct {
	//Mutex &sync.Mutex{}
	Type     int
	Nodes    []*Node
	NodesMap map[string]*Node
}

/*NewNodePool - create a new node pool of given type */
func NewNodePool(Type int) NodePool {
	np := NodePool{Type: Type}
	np.NodesMap = make(map[string]*Node)
	return np
}

/*Size - size of the pool without regards to the node status */
func (np *NodePool) Size() int {
	return len(np.Nodes)
}

/*AddNode - add a nodes to the pool */
func (np *NodePool) AddNode(node *Node) {
	if np.Type != node.Type {
		return
	}
	var nodeID = node.GetID()
	np.NodesMap[nodeID] = node
	np.computeNodesArray()
}

func (np *NodePool) GetNode(id string) *Node {
	node, ok := np.NodesMap[id]
	if !ok {
		return nil
	}
	return node
}

/*RemoveNode - Remove a node from the pool */
func (np *NodePool) RemoveNode(nodeID string) {
	if _, ok := np.NodesMap[nodeID]; !ok {
		return
	}
	delete(np.NodesMap, nodeID)
	np.computeNodesArray()
}

var NONE = make([]*Node, 0)

func (np *NodePool) shuffleNodes() []*Node {
	size := np.Size()
	if size == 0 {
		return NONE
	}
	shuffled := make([]*Node, size)
	perm := rand.Perm(size)
	for i, v := range perm {
		shuffled[v] = np.Nodes[i]
	}
	return shuffled
}

func (np *NodePool) computeNodesArray() {
	// TODO: Do we need to use Mutex while doing this?
	var array = make([]*Node, 0, len(np.NodesMap))
	for _, v := range np.NodesMap {
		array = append(array, v)
	}
	np.Nodes = array
}

var r = rand.New(rand.NewSource(99))

/*GetRandomNodes - get a random set of nodes from the pool
* Doesn't consider active/inactive status
 */
func (np *NodePool) GetRandomNodes(num int) []*Node {
	var size = np.Size()
	if num > size {
		num = size
	}
	nodes := np.shuffleNodes()
	return nodes[:num]
}

/*Print - print this pool. This will be used for http response and Read method should be able to consume it*/
func (np *NodePool) Print(w io.Writer) {
	nodes := np.shuffleNodes()
	for _, node := range nodes {
		node.Print(w)
	}
}

/*ReadNodes - read the pool information */
func ReadNodes(r io.Reader, minerPool *NodePool, sharderPool *NodePool, blobberPool *NodePool) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		node, err := Read(line)
		if err != nil {
			panic(err)
		}
		if Self != nil && node == Self.Node {
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

/*StatusMonitor - a background job that keeps checking the status of the nodes */
func (np *NodePool) StatusMonitor() {
	tr := &http.Transport{
		MaxIdleConns:       1000,            // TODO: since total nodes is expected to be fixed, this may be OK
		IdleConnTimeout:    2 * time.Minute, // more than the frequency of checking will ensure always on
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr, Timeout: 500 * time.Millisecond}
	//ticker := time.NewTicker(2 * time.Minute)
	ticker := time.NewTicker(20 * time.Second)
	for _ = range ticker.C {
		nodes := np.shuffleNodes()
		for _, node := range nodes {
			statusURL := node.GetStatusURL()
			data, hash, signature, err := Self.TimeStampSignature(Self.privateKey)
			if err != nil {
				panic(err)
			}
			statusURL = fmt.Sprintf("%v&data=%v&hash=%v&signature=%v", statusURL, data, hash, signature)
			_, err = client.Get(statusURL)
			if err != nil {
				fmt.Printf("error connecting to %v: %v\n", node.GetID(), err)
				node.ErrorCount++
				if node.ErrorCount > 5 {
					node.Status = NodeStatusInactive
					fmt.Printf("node %v became inactive\n", node.GetID())
				}
			} else {
				node.ErrorCount = 0
				node.Status = NodeStatusActive
				node.LastActiveTime = common.Now()
				fmt.Printf("node %v became active\n", node.GetID())
			}
		}
	}
}

/*Miners - this is the pool of miners */
var Miners = NewNodePool(NodeTypeMiner)

/*Sharders - this is the pool of sharders */
var Sharders = NewNodePool(NodeTypeSharder)

/*Blobbers - this is the pool of blobbers */
var Blobbers = NewNodePool(NodeTypeBlobber)
