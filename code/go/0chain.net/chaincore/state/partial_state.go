package state

import (
	"context"

	"0chain.net/core/util"
)

//PartialState - an entity to exchange partial state
type PartialState struct {
	Version string      `json:"version"`
	Nodes   []util.Node `json:"_"`
	mndb    *util.MemoryNodeDB
	root    util.Node
}

//NewNodeDB - create a node db from the changes
func (ps *PartialState) newNodeDB() *util.MemoryNodeDB {
	mndb := util.NewMemoryNodeDB()
	for _, n := range ps.Nodes {
		mndb.PutNode(n.GetHashBytes(), n)
	}
	return mndb
}

//ComputeProperties - implement interface
func (ps *PartialState) ComputeProperties() {
	mndb := ps.newNodeDB()
	root := mndb.ComputeRoot()
	if root != nil {
		ps.mndb = mndb
		ps.root = root
	}
}

//Validate - implement interface
func (ps *PartialState) Validate(ctx context.Context) error {
	return ps.mndb.Validate(ps.root)
}

/*GetRoot - get the root of this set of changes */
func (ps *PartialState) GetRoot() util.Node {
	return ps.root
}

/*GetNodeDB - get the node db containing all the changes */
func (ps *PartialState) GetNodeDB() util.NodeDB {
	return ps.mndb
}
