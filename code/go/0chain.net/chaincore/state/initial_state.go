package state

import (
	"os"

	"0chain.net/core/datastore"
	"github.com/0chain/common/core/currency"

	"gopkg.in/yaml.v2"
)

// InitStates is a slice of InitState used for all the initial states in the genesis block.
type InitStates struct {
	States []InitState `yaml:"initialStates"`
	Stakes []InitStake `yaml:"initialStakes"`
}

// InitState is a clients initial state in the genesis block.
type InitState struct {
	ID     datastore.Key `yaml:"id"`     // smartcontract address
	Tokens currency.Coin `yaml:"tokens"` // smartcontract init tokens
	State  []IDTokens    `yaml:"state"`  // initial state of each client, tokens will be transfered from smartcontract address
}

// IDTokens wraps the ID and Tokens
type IDTokens struct {
	ID     datastore.Key `yaml:"id"`
	Tokens currency.Coin `yaml:"tokens"`
}

// InitStake represents the initial stakings for active set providers
type InitStake struct {
	ProviderID   datastore.Key `yaml:"provider_id"`
	ProviderType datastore.Key `yaml:"provider_type"`
	ClientID     datastore.Key `yaml:"client_id"`
	Tokens       currency.Coin `yaml:"tokens"`
}

// NewInitStates is used to return a new InitStates.
func NewInitStates() *InitStates {
	return &InitStates{}
}

// Read is use on the InitStates to read the initial states for the genesis block from a yaml file.
func (initStates *InitStates) Read(file string) (err error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(bytes, initStates)
	return
}
