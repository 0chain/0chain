package state

import "github.com/0chain/common/core/util"

//DeserializerI - transforms one serializable value (an abstract) to another (a concrete value)
type DeserializerI interface {
	Deserialize(sv util.Serializable) util.Serializable
}
