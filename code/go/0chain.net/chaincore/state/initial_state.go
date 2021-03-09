package state

import (
	"0chain.net/core/datastore"

	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// InitStates ...
type InitStates struct {
	States []InitState `yaml:"initialStates"`
}

// InitState ...
type InitState struct {
	ID     datastore.Key `yaml:"id"`
	Tokens Balance       `yaml:"tokens"`
}

// NewInitStates ...
func NewInitStates() *InitStates {
	return &InitStates{}
}

// Read ...
func (is *InitStates) Read(file string) (err error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(bytes, is)
	return
}
