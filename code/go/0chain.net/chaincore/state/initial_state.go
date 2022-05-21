package state

import (
	"io/ioutil"

	"0chain.net/chaincore/currency"
	"0chain.net/core/datastore"

	"gopkg.in/yaml.v2"
)

// InitStates is a slice of InitState used for all the initial states in the genesis block.
type InitStates struct {
	States []InitState `yaml:"initialStates"`
}

// InitState is a clients initial state in the genesis block.
type InitState struct {
	ID     datastore.Key `yaml:"id"`
	Tokens currency.Coin `yaml:"tokens"`
}

// NewInitStates is used to return a new InitStates.
func NewInitStates() *InitStates {
	return &InitStates{}
}

// Read is use on the InitStates to read the initial states for the genesis block from a yaml file.
func (initStates *InitStates) Read(file string) (err error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(bytes, initStates)
	return
}
