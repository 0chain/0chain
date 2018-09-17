package node

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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

/*Self represents the node of this intance */
var Self *SelfNode

func init() {
	Self = &SelfNode{}
	Self.Node = &Node{}
}
