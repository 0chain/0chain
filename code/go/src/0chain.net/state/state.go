package state

import (
	"fmt"
	"strconv"

	"0chain.net/encryption"
)

//Balance - any quantity that is represented as an integer in the lowest denomination
type Balance int64

//State - state that needs consensus within the blockchain.
type State struct {
	Balance Balance
}

/*GetHash - implement interface */
func (s *State) GetHash() string {
	return encryption.Hash(string(s.Encode()))
}

/*Encode - implement interface */
func (s *State) Encode() []byte {
	return []byte(fmt.Sprintf("%v", s.Balance))
}

/*Decode - implement interface */
func (s *State) Decode(data []byte) error {
	balance, err := strconv.ParseInt(string(data), 10, 63)
	if err != nil {
		return err
	}
	s.Balance = Balance(balance)
	return nil
}
