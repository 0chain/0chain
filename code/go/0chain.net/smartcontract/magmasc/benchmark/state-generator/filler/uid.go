package filler

const (
	// acknowledgment contents a value of acknowledgment string type.
	acknowledgment = "acknowledgment"
)

// nodeUID returns an uniq id for Node interacting with magma smart contract.
// Should be used while inserting, removing or getting nodes into state.StateContextI.
func nodeUID(scID, prefix, key string) string {
	colon := ":"
	return "sc:" + scID + colon + prefix + colon + key
}
