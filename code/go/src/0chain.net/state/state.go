package state

import (
	"fmt"
	"strconv"

	"0chain.net/encryption"
	"0chain.net/util"
)

//Balance - any quantity that is represented as an integer in the lowest denomination
type Balance int64

//State - state that needs consensus within the blockchain.
type State struct {
	Balance Balance `json:"balance"`
}

/*GetHash - implement SecureSerializableValueI interface */
func (s *State) GetHash() string {
	return util.ToHex(s.GetHashBytes())
}

/*GetHashBytes - implement SecureSerializableValueI interface */
func (s *State) GetHashBytes() []byte {
	return encryption.RawHash(s.Encode())
}

/*Encode - implement SecureSerializableValueI interface */
func (s *State) Encode() []byte {
	return []byte(fmt.Sprintf("%v", s.Balance))
}

/*Decode - implement SecureSerializableValueI interface */
func (s *State) Decode(data []byte) error {
	balance, err := strconv.ParseInt(string(data), 10, 63)
	if err != nil {
		return err
	}
	s.Balance = Balance(balance)
	return nil
}

//Deserializer - a deserializer to convert raw serialized data to a state object
type Deserializer struct {
}

//Deserialize - implement interface
func (bd *Deserializer) Deserialize(sv util.Serializable) util.Serializable {
	s := &State{}
	s.Decode(sv.Encode())
	return s
}
