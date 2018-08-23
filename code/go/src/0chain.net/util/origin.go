package util

import (
	"encoding/binary"
	"io"
)

//Origin - a data type that tracks the origin of a node , for example a round number of the block
type Origin int64

/*OriginTrackerI - tracks the origin
* This is required to do pruning of nodes. When a new node is introduced in a given block generation round due to associated state changes, the node's origin is the
* corresponding block's round. When we want to do pruning, we start with the state root of a given round's block, sweep through all nodes accessible with that round
* and mark the origin to that round number. Then, we iterate through the entire db and for any node that has the origin smaller than this value, it can be safely
* deleted.
 */
type OriginTrackerI interface {
	SetOrigin(origin Origin)
	GetOrigin() Origin
	Write(w io.Writer) error
	Read(w io.Reader) error
}

/*OriginTracker - implements the OriginTrackerI interface */
type OriginTracker struct {
	Origin Origin `json:"origin" msgpack:"o"`
}

/*SetOrigin - set the origin */
func (o *OriginTracker) SetOrigin(origin Origin) {
	o.Origin = origin
}

/*GetOrigin - get the origin */
func (o *OriginTracker) GetOrigin() Origin {
	return o.Origin
}

func (o *OriginTracker) Write(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, o.GetOrigin())
}

func (o *OriginTracker) Read(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, &o.Origin)
}
