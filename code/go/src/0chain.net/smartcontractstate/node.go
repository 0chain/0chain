package smartcontractstate

import (
	"0chain.net/common"
)

//Separator - used to separate fields when creating data array to hash
const Separator = ":"

//ErrInvalidEncoding - error to indicate invalid encoding
var ErrInvalidEncoding = common.NewError("invalid_node_encoding", "invalid node encoding")

/*Node - a node interface */
type Node []byte

type Key []byte
