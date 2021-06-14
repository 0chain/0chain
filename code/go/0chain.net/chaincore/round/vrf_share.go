package round

import (
	"context"
	"fmt"

	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
)

//VRFShare - a VRF share
type VRFShare struct {
	datastore.NOIDField
	Round             int64  `json:"round"`
	Share             string `json:"share"`
	RoundTimeoutCount int    `json:"timeoutcount"`
	party             *node.Node
}

var vrfsEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (vrfs *VRFShare) GetEntityMetadata() datastore.EntityMetadata {
	return vrfsEntityMetadata
}

/*GetKey - returns the round number as the key */
func (vrfs *VRFShare) GetKey() datastore.Key {
	return datastore.ToKey(fmt.Sprintf("%v", vrfs.Round))
}

/*Read - read round entity from store */
func (vrfs *VRFShare) Read(ctx context.Context, key datastore.Key) error {
	return vrfs.GetEntityMetadata().GetStore().Read(ctx, key, vrfs)
}

/*Write - write round entity to store */
func (vrfs *VRFShare) Write(ctx context.Context) error {
	return vrfs.GetEntityMetadata().GetStore().Write(ctx, vrfs)
}

/*Delete - delete round entity from store */
func (vrfs *VRFShare) Delete(ctx context.Context) error {
	return vrfs.GetEntityMetadata().GetStore().Delete(ctx, vrfs)
}

/*VRFShareProvider - entity provider for client object */
func VRFShareProvider() datastore.Entity {
	vrfs := &VRFShare{}
	return vrfs
}

/*SetupVRFShareEntity - setup the entity */
func SetupVRFShareEntity(store datastore.Store) {
	vrfsEntityMetadata = datastore.MetadataProvider()
	vrfsEntityMetadata.Name = "vrfs"
	vrfsEntityMetadata.Provider = VRFShareProvider
	vrfsEntityMetadata.Store = store
	vrfsEntityMetadata.IDColumnName = "round"
	datastore.RegisterEntityMetadata("vrfs", vrfsEntityMetadata)
}

//GetRoundNumber - return the round associated with this vrf share
func (vrfs *VRFShare) GetRoundNumber() int64 {
	return vrfs.Round
}

// GetRoundTimeoutCount return timeout count for this round
func (vrfs *VRFShare) GetRoundTimeoutCount() int {
	return vrfs.RoundTimeoutCount
}

//SetParty - set the party contributing this vrf share
func (vrfs *VRFShare) SetParty(party *node.Node) {
	vrfs.party = party
}

//GetParty - get the party contributing this vrf share
func (vrfs *VRFShare) GetParty() *node.Node {
	return vrfs.party
}

// Clone returns a clone of the VRFShare
func (vrfs *VRFShare) Clone() *VRFShare {
	clone := &VRFShare{
		NOIDField:         vrfs.NOIDField,
		Round:             vrfs.Round,
		Share:             vrfs.Share,
		RoundTimeoutCount: vrfs.RoundTimeoutCount,
	}

	if vrfs.party != nil {
		clone.party = vrfs.party.Clone()
	}

	return clone
}
