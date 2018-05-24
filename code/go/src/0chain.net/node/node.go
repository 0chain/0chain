package node

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/encryption"
)

var (
	NodeStatusInactive = 0
	NodeStatusActive   = 1
)

var (
	NodeTypeMiner   = 1
	NodeTypeSharder = 2
	NodeTypeBlobber = 3
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

/*GetID - get the id of the node */
func (n *Node) GetID() string {
	return n.ID
}

/*Equals - if two nodes are equal. Only check by id, we don't accept configuration from anyone */
func (n *Node) Equals(n2 *Node) bool {
	if n.GetID() == n2.GetID() {
		return true
	}
	if n.Port == n2.Port && n.Host == n2.Host {
		return true
	}
	return false
}

/*Print - print node's info that is consumable by Read */
func (n *Node) Print(w io.Writer) {
	fmt.Fprintf(w, "%v,%v,%v,%v,%v\n", n.GetNodeType(), n.Host, n.Port, n.GetID(), n.PublicKey)
}

/*Read - read a node config line and create the node */
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
	if node.Host == "" {
		if node.Port != config.Configuration.Port {
			node.Host = config.Configuration.Host
		} else {
			panic(fmt.Sprintf("invalid node setup for %v\n", node.GetID()))
		}
	}

	port, err := strconv.ParseInt(fields[2], 10, 32)
	if err != nil {
		return nil, err
	}
	node.Port = int(port)
	node.ID = fields[3]
	node.PublicKey = fields[4]
	if Self == nil && node.Host == config.Configuration.Host && node.Port == config.Configuration.Port {
		Self = &SelfNode{Node: node}
	}
	return node, nil
}

/*GetURLBase - get the end point base */
func (n *Node) GetURLBase() string {
	host := n.Host
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%v:%v", host, n.Port)
}

/*GetStatusURL - get the end point where to ping for the status */
func (n *Node) GetStatusURL() string {
	return fmt.Sprintf("%v/_nh/status?id=%v&publicKey=%v", n.GetURLBase(), n.ID, n.PublicKey)
}

/*Verify signature */
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

/*SelfNode -- self node type*/
type SelfNode struct {
	*Node
	privateKey string
}

/*SetPrivateKey - setter */
func (sn *SelfNode) SetPrivateKey(privateKey string) {
	sn.privateKey = privateKey
}

/*TimeStampSignature - get timestamp based signature */
func (sn *SelfNode) TimeStampSignature() (string, string, string, error) {
	data := fmt.Sprintf("%v:%v", sn.ID, common.Now())
	hash := encryption.Hash(data)
	signature, err := encryption.Sign(sn.privateKey, hash)
	if err != nil {
		return "", "", "", err
	}
	return data, hash, signature, err
}

/*Self represents the node of this intance */
var Self *SelfNode
