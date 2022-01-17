package minersc

import "encoding/json"

// VersionNode represents the smart contract version node stores in MPT
type VersionNode string

func (v VersionNode) Encode() []byte {
	return []byte(v)
}

func (v *VersionNode) Decode(b []byte) error {
	*v = VersionNode(b)
	return nil
}

func (v *VersionNode) String() string {
	return string(*v)
}

// UpdateVersionTxnInput represents the transaction data struct for
// updating the smart contract version
type UpdateVersionTxnInput struct {
	Version string `json:"version"`
}

// Decode implements the mpt node decode interface
func (v *UpdateVersionTxnInput) Decode(b []byte) error {
	return json.Unmarshal(b, v)
}

// Encode implements the mpt node encode interface
func (v *UpdateVersionTxnInput) Encode() ([]byte, error) {
	b, err := json.Marshal(v)
	return b, err
}
