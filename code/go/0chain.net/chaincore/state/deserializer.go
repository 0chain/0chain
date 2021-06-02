package state

import "github.com/0chain/0chain/code/go/0chain.net/core/util"

//DeserializerI - transforms one serializable value (an abstract) to another (a concrete value)
type DeserializerI interface {
	Deserialize(sv util.Serializable) util.Serializable
}
