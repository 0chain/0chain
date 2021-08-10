package magmasc

import (
	"encoding/json"

	"0chain.net/core/util"
)

type (
	// flagBool represents serializable boolean flag implementation.
	flagBool bool
)

var (
	// Make sure flagBool implements util.Serializable interface.
	_ util.Serializable = (*flagBool)(nil)
)

// newFlag returns a new constructed flag with given value.
func newFlag(val bool) *flagBool {
	flag := flagBool(val)
	return &flag
}

// Decode implements util.Serializable interface.
func (m *flagBool) Decode(blob []byte) error {
	return json.Unmarshal(blob, m)
}

// Encode implements util.Serializable interface.
func (m *flagBool) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}
