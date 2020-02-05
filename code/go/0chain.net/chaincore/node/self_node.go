package node

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"0chain.net/core/build"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

/*SelfNode -- self node type*/
type SelfNode struct {
	mutex sync.RWMutex // protect the *Node field
	*Node

	signatureScheme encryption.SignatureScheme
}

/*SetSignatureScheme - getter */
func (sn *SelfNode) GetSignatureScheme() encryption.SignatureScheme {
	return sn.signatureScheme
}

/*SetSignatureScheme - setter */
func (sn *SelfNode) SetSignatureScheme(signatureScheme encryption.SignatureScheme) {
	sn.signatureScheme = signatureScheme
	sn.SetPublicKey(signatureScheme.GetPublicKey())
}

/*Sign - sign the given hash */
func (sn *SelfNode) Sign(hash string) (string, error) {
	return sn.signatureScheme.Sign(hash)
}

// GetKey returns ID (public key) of underlying Node instance.
func (sn *SelfNode) GetKey() string {
	sn.mutex.RLock()
	defer sn.mutex.RUnlock()

	return sn.ID
}

/*TimeStampSignature - get timestamp based signature */
func (sn *SelfNode) TimeStampSignature() (string, string, string, error) {
	data := fmt.Sprintf("%v:%v", sn.GetKey(), common.Now())
	hash := encryption.Hash(data)
	signature, err := sn.Sign(hash)
	if err != nil {
		return "", "", "", err
	}
	return data, hash, signature, err
}

/*ValidateSignatureTime - validate if the time stamp used in the signature is valid */
func (sn *SelfNode) ValidateSignatureTime(data string) (bool, error) {
	segs := strings.Split(data, ":")
	if len(segs) < 2 {
		return false, errors.New("invalid data")
	}
	ts, err := strconv.ParseInt(segs[1], 10, 64)
	if err != nil {
		return false, err
	}
	if !common.Within(ts, 3) {
		return false, errors.New("timestamp not within tolerance")
	}
	return true, nil
}

// SetNode field of the SeflNode. I.e. make given Node
// instance global for an application uses this package.
func (sn *SelfNode) SetNode(node *Node) {
	sn.mutex.Lock()
	defer sn.mutex.Unlock()

	sn.Node = node
}

// SetNodeIfEqKey makes given node underlying for the SelfNode
// if its public key is equal to current underlying Node.
func (sn *SelfNode) SetNodeIfEqKey(node *Node) {
	sn.mutex.Lock()
	defer sn.mutex.Unlock()

	if sn.ID == node.GetKey() {
		sn.Node = node

		node.mutex.Lock()
		defer node.mutex.Unlock()

		sn.Node.Info.StateMissingNodes = -1
		sn.Node.Info.BuildTag = build.BuildTag
		sn.Node.Status = NodeStatusActive
	}
}

// IsEq returns true if given Node reference
// is the same as Node of the SelfNode.
func (sn *SelfNode) IsEq(node *Node) bool {
	sn.mutex.Lock()
	defer sn.mutex.Unlock()

	return sn.Node == node
}

// IsEqKey returns true if public key of given node is
// equal to public key of the SelfNode.
func (sn *SelfNode) IsEqKey(node *Node) bool {
	return sn.GetKey() == node.GetKey()
}

// OnNode performs an asynchronous work on underlying Node instance.
func (sn *SelfNode) OnNode(onNodeFunc func(node *Node)) {
	sn.mutex.Lock()
	defer sn.mutex.Unlock()

	onNodeFunc(sn.Node)
}

// new blank SelfNode
func newSelfNode() (sn *SelfNode) {
	sn = new(SelfNode)
	sn.Node = new(Node)
	return
}

/*Self represents the node of this instance */
var Self = newSelfNode()
