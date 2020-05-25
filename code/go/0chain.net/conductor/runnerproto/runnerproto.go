package runnerproto

import (
	"0chain.net/conductor/config"
)

// Msg of the protocol.
type Msg struct {
	Do   string       `json:"do"`
	Node *config.Node `json:"node"`
}
