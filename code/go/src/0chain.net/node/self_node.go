package node

import (
	"fmt"

	"0chain.net/common"
	"0chain.net/encryption"
)

/*SelfNode -- self node type*/
type SelfNode struct {
	*Node
	privateKey string
}

/*SetKeys - setter */
func (sn *SelfNode) SetKeys(publicKey string, privateKey string) {
	sn.PublicKey = publicKey
	sn.privateKey = privateKey
}

/*Sign - sign the given hash */
func (sn *SelfNode) Sign(hash string) (string, error) {
	return encryption.Sign(sn.privateKey, hash)
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

func init() {
	Self = &SelfNode{}
	Self.Node = &Node{}
}
