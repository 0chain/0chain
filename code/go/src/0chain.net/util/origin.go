package util

import (
	"encoding/binary"
	"io"
)

//Sequence - a data type that can be used to track the origin, latest version and sync version of a node
type Sequence int64

/*OriginTrackerI - tracks the origin
* This is required to do pruning of nodes. When a new node is introduced in a given block generation round due to associated state changes, the node's origin and
* version is the corresponding block's round. When we want to do pruning, we start with the state root of a given round's block, sweep through all nodes accessible
* with that round and mark the version to that round number. Then, we iterate through the entire db and for any node that has the origin smaller than this value,
* it can be safely deleted. */
type OriginTrackerI interface {
	SetOrigin(origin Sequence)
	GetOrigin() Sequence

	SetVersion(version Sequence)
	GetVersion() Sequence

	Write(w io.Writer) error
	Read(r io.Reader) error
}

/*OriginTracker - implements the OriginTrackerI interface */
type OriginTracker struct {
	Origin  Sequence `json:"origin" msgpack:"o"`
	Version Sequence `json:"version" msgpack:"v"`
}

/*SetOrigin - set the origin */
func (o *OriginTracker) SetOrigin(origin Sequence) {
	o.Origin = origin
	o.SetVersion(origin)
}

/*GetOrigin - get the origin */
func (o *OriginTracker) GetOrigin() Sequence {
	return o.Origin
}

//SetVersion - implement interface
func (o *OriginTracker) SetVersion(version Sequence) {
	o.Version = version
}

//GetVersion - implement interface
func (o *OriginTracker) GetVersion() Sequence {
	return o.Version
}

func (o *OriginTracker) Write(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, o.Version)
	if err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, o.Origin)
}

func (o *OriginTracker) Read(r io.Reader) error {
	err := binary.Read(r, binary.LittleEndian, &o.Version)
	if err != nil {
		return err
	}
	return binary.Read(r, binary.LittleEndian, &o.Origin)
}
