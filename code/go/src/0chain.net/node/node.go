package node

import (
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
)

var nodes = make(map[string]*Node)

/*RegisterNode - register a node to a global registery
* We need to keep track of a global register of nodes. This is required to ensure we can verify a signed request
* coming from a node
 */
func RegisterNode(node *Node) {
	nodes[node.GetKey()] = node
}

/*DeregisterNode - deregisters a node */
func DeregisterNode(nodeID string) {
	delete(nodes, nodeID)
}

/*GetNode - get the node from the registery */
func GetNode(nodeID string) *Node {
	return nodes[nodeID]
}

var (
	NodeStatusInactive = 0
	NodeStatusActive   = 1
)

var (
	NodeTypeMiner   int8 = 0
	NodeTypeSharder int8 = 1
	NodeTypeBlobber int8 = 2
)

var NodeTypeNames = common.CreateLookups("m", "Miner", "s", "Sharder", "b", "Blobber")

/*Node - a struct holding the node information */
type Node struct {
	client.Client
	N2NHost        string
	Host           string
	Port           int
	Type           int8
	Description    string
	SetIndex       int
	Status         int
	LastActiveTime time.Time
	ErrorCount     int
	CommChannel    chan bool

	//These are approximiate as we are not going to lock to update
	Sent       int64 // messages sent to this node
	SendErrors int64 // failed message sent to this node
	Received   int64 // messages received from this node

	TimersByURI map[string]metrics.Timer
	SizeByURI   map[string]metrics.Histogram

	LargeMessageSendTime float64
	SmallMessageSendTime float64

	LargeMessagePullServeTime float64
	SmallMessagePullServeTime float64

	mutex *sync.Mutex

	ProtocolStats interface{}

	idBytes []byte
}

/*Provider - create a node object */
func Provider() *Node {
	node := &Node{}
	// queue up at most these many messages to a node
	// because of this, we don't want the status monitoring to use this communication layer
	node.CommChannel = make(chan bool, 5)
	for i := 0; i < cap(node.CommChannel); i++ {
		node.CommChannel <- true
	}
	node.mutex = &sync.Mutex{}
	node.TimersByURI = make(map[string]metrics.Timer, 10)
	node.SizeByURI = make(map[string]metrics.Histogram, 10)
	return node
}

/*Equals - if two nodes are equal. Only check by id, we don't accept configuration from anyone */
func (n *Node) Equals(n2 *Node) bool {
	if datastore.IsEqual(n.GetKey(), n2.GetKey()) {
		return true
	}
	if n.Port == n2.Port && n.Host == n2.Host {
		return true
	}
	return false
}

/*Print - print node's info that is consumable by Read */
func (n *Node) Print(w io.Writer) {
	fmt.Fprintf(w, "%v,%v,%v,%v,%v\n", n.GetNodeType(), n.Host, n.Port, n.GetKey(), n.PublicKey)
}

/*Read - read a node config line and create the node */
func Read(line string) (*Node, error) {
	node := Provider()
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
	if node.Host == "" {
		if node.Port != config.Configuration.Port {
			node.Host = config.Configuration.Host
		} else {
			panic(fmt.Sprintf("invalid node setup for %v\n", node.GetKey()))
		}
	}

	port, err := strconv.ParseInt(fields[2], 10, 32)
	if err != nil {
		return nil, err
	}
	node.Port = int(port)
	node.SetID(fields[3])
	node.PublicKey = fields[4]
	node.Client.SetPublicKey(node.PublicKey)
	hash := encryption.Hash(node.PublicKeyBytes)
	if node.ID != hash {
		return nil, common.NewError("invalid_client_id", fmt.Sprintf("public key: %v, client_id: %v, hash: %v\n", node.PublicKey, node.ID, hash))
	}
	node.ComputeProperties()
	if Self.PublicKey == node.PublicKey {
		setSelfNode(node)
	}
	return node, nil
}

/*NewNode - read a node config line and create the node */
func NewNode(nc map[interface{}]interface{}) (*Node, error) {
	node := Provider()
	node.Type = nc["type"].(int8)
	node.Host = nc["public_ip"].(string)
	node.N2NHost = nc["n2n_ip"].(string)
	node.Port = nc["port"].(int)
	node.SetID(nc["id"].(string))
	node.PublicKey = nc["public_key"].(string)
	if description, ok := nc["description"]; ok {
		node.Description = description.(string)
	}

	node.Client.SetPublicKey(node.PublicKey)
	hash := encryption.Hash(node.PublicKeyBytes)
	if node.ID != hash {
		return nil, common.NewError("invalid_client_id", fmt.Sprintf("public key: %v, client_id: %v, hash: %v\n", node.PublicKey, node.ID, hash))
	}
	node.ComputeProperties()
	if Self.PublicKey == node.PublicKey {
		setSelfNode(node)
	}
	return node, nil
}

func setSelfNode(n *Node) {
	Self.Node = n
	Self.Node.Status = NodeStatusActive
}

/*ComputeProperties - implement entity interface */
func (n *Node) ComputeProperties() {
	n.Client.ComputeProperties()
	if n.Host == "" {
		n.Host = "localhost"
	}
	if n.N2NHost == "" {
		n.N2NHost = n.Host
	}
}

/*GetURLBase - get the end point base */
func (n *Node) GetURLBase() string {
	return fmt.Sprintf("http://%v:%v", n.Host, n.Port)
}

/*GetN2NURLBase - get the end point base for n2n communication */
func (n *Node) GetN2NURLBase() string {
	return fmt.Sprintf("http://%v:%v", n.N2NHost, n.Port)
}

/*GetStatusURL - get the end point where to ping for the status */
func (n *Node) GetStatusURL() string {
	return fmt.Sprintf("%v/_nh/status", n.GetN2NURLBase())
}

/*GetNodeType - as a string */
func (n *Node) GetNodeType() string {
	return NodeTypeNames[n.Type].Code
}

/*GetNodeTypeName - get the name of this node type */
func (n *Node) GetNodeTypeName() string {
	return NodeTypeNames[n.Type].Value
}

//Grab - grab a slot to send message
func (n *Node) Grab() {
	<-n.CommChannel
	n.Sent++
}

//Release - release a slot after sending the message
func (n *Node) Release() {
	n.CommChannel <- true
}

//GetTimer - get the timer
func (n *Node) GetTimer(uri string) metrics.Timer {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	timer, ok := n.TimersByURI[uri]
	if !ok {
		timerID := fmt.Sprintf("%v.%v.time", n.ID, uri)
		timer = metrics.GetOrRegisterTimer(timerID, nil)
		n.TimersByURI[uri] = timer
	}
	return timer
}

//GetSizeMetric - get the size metric
func (n *Node) GetSizeMetric(uri string) metrics.Histogram {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	metric, ok := n.SizeByURI[uri]
	if !ok {
		metricID := fmt.Sprintf("%v.%v.size", n.ID, uri)
		metric = metrics.NewHistogram(metrics.NewUniformSample(256))
		n.SizeByURI[uri] = metric
		metrics.Register(metricID, metric)
	}
	return metric
}

//GetLargeMessageSendTime - get the time it takes to send a large message to this node
func (n *Node) GetLargeMessageSendTime() float64 {
	return n.LargeMessageSendTime / 1000000
}

//GetSmallMessageSendTime - get the time it takes to send a small message to this node
func (n *Node) GetSmallMessageSendTime() float64 {
	return n.SmallMessageSendTime / 1000000
}

func (n *Node) updateMessageTimings() {
	n.updateSendMessageTimings()
	n.updateRequestMessageTimings()
}

func (n *Node) updateSendMessageTimings() {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	var minval = math.MaxFloat64
	var maxval float64
	var minSize = math.MaxFloat64
	var maxSize float64
	for uri, timer := range n.TimersByURI {
		if timer.Count() == 0 {
			continue
		}
		if isPullRequestURI(uri) {
			continue
		}
		if sizer, ok := n.SizeByURI[uri]; ok {
			if sizer.Mean() == 0 {
				continue
			}
			v := timer.Mean()
			if sizer.Mean() > maxSize {
				maxSize = sizer.Mean()
				if v > maxval {
					maxval = v
				}
			}
			if sizer.Mean() < minSize {
				minSize = sizer.Mean()
				if v < minval {
					minval = v
				}
			}
		}
	}
	if minval > maxval {
		minval = maxval
	}
	n.LargeMessageSendTime = maxval
	n.SmallMessageSendTime = minval
}

func (n *Node) updateRequestMessageTimings() {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	var minval = math.MaxFloat64
	var maxval float64
	var minSize = math.MaxFloat64
	var maxSize float64
	for uri, timer := range n.TimersByURI {
		if timer.Count() == 0 {
			continue
		}
		if !isPullRequestURI(uri) {
			continue
		}
		v := timer.Mean()
		if sizer, ok := n.SizeByURI[uri]; ok {
			if sizer.Mean() == 0 {
				continue
			}
			if sizer.Mean() > maxSize {
				maxSize = sizer.Mean()
				if v > maxval {
					maxval = v
				}
			}
			if sizer.Mean() < minSize {
				minSize = sizer.Mean()
				if v < minval {
					minval = v
				}
			}
		}
	}
	if minval > maxval {
		minval = maxval
	}
	n.LargeMessagePullServeTime = maxval
	n.SmallMessagePullServeTime = minval
}

//ReadConfig - read configuration from the default config
func ReadConfig() {
	SetTimeoutSmallMessage(viper.GetDuration("network.timeout.small_message") * time.Millisecond)
	SetTimeoutLargeMessage(viper.GetDuration("network.timeout.large_message") * time.Millisecond)
	SetMaxConcurrentRequests(viper.GetInt("network.max_concurrent_requests"))
	SetLargeMessageThresholdSize(viper.GetInt("network.large_message_th_size"))
}

//SetID - set the id of the node
func (n *Node) SetID(id string) error {
	n.ID = id
	bytes, err := hex.DecodeString(id)
	if err != nil {
		return err
	}
	n.idBytes = bytes
	return nil
}

//IsActive - returns if this node is active or not
func (n *Node) IsActive() bool {
	return n.Status == NodeStatusActive
}

func serveMetricKey(uri string) string {
	return "p?" + uri
}

func isPullRequestURI(uri string) bool {
	return strings.HasPrefix(uri, "p?")
}

//GetPseudoName - create a pseudo name that is unique in the current active set
func (n *Node) GetPseudoName() string {
	return fmt.Sprintf("%v%.3d", n.GetNodeTypeName(), n.SetIndex)
}

//GetOptimalLargeMessageSendTime - get the push or pull based optimal large message send time
func (n *Node) GetOptimalLargeMessageSendTime() float64 {
	return n.getOptimalLargeMessageSendTime() / 1000000
}

func (n *Node) getOptimalLargeMessageSendTime() float64 {
	p2ptime := getPushToPullTime(n)
	if p2ptime < n.LargeMessageSendTime {
		return p2ptime
	}
	return n.LargeMessageSendTime
}

func (n *Node) getTime(uri string) float64 {
	pullTimer := n.GetTimer(uri)
	return pullTimer.Mean()
}
