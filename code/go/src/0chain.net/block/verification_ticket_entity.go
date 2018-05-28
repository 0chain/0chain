package block

import (
	"context"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*VerificationTicket - verification ticket for the block */
type VerificationTicket struct {
	VerifierID datastore.Key `json:"verifier_id"`
	Signature  string        `json:"signature"`
}

/*BlockVerificationTicket - verification ticket with the block id.
* As VerificationTickets are contained in a block, it doesn't need to have a reference to a block
* However, when the verifiers verify and send the tickets, they need to indicate what block the
* verification ticket is for. So, this wrapper data strcuture is used for that.
 */
type BlockVerificationTicket struct {
	VerificationTicket
	BlockID datastore.Key `json:"block_id"`
}

/*TODO: Making BlockVerificationTicket an entity for now as N2N handler framework uses entity.
* May be we get rid of the entity requirement later as there is no true ID for this */

/*GetEntityName - implementing the interface */
func (bvt *BlockVerificationTicket) GetEntityName() string {
	return "block_verification_ticket"
}

/*GetKey - implementing the interface */
func (bvt *BlockVerificationTicket) GetKey() datastore.Key {
	return datastore.EmptyKey
}

/*SetKey - implementing the interface */
func (bvt *BlockVerificationTicket) SetKey(key datastore.Key) {
}

/*ComputeProperties - implementing the interface */
func (bvt *BlockVerificationTicket) ComputeProperties() {
}

/*Validate - implementing the interface */
func (bvt *BlockVerificationTicket) Validate(ctx context.Context) error {
	if datastore.IsEmpty(bvt.VerifierID) {
		return common.InvalidRequest("block_verification_ticket id is required")
	}
	return nil
}

/*Read - datastore read */
func (bvt *BlockVerificationTicket) Read(ctx context.Context, key datastore.Key) error {
	return datastore.Read(ctx, key, bvt)
}

/*Write - datastore read */
func (bvt *BlockVerificationTicket) Write(ctx context.Context) error {
	return datastore.Write(ctx, bvt)
}

/*Delete - datastore read */
func (bvt *BlockVerificationTicket) Delete(ctx context.Context) error {
	return datastore.Delete(ctx, bvt)
}

/*BVTProvider - entity provider for block_verification_ticket object */
func BVTProvider() interface{} {
	bvt := &BlockVerificationTicket{}
	return bvt
}

/*SetupBVTEntity - setup the entity */
func SetupBVTEntity() {
	datastore.RegisterEntityProvider("block_verification_ticket", BVTProvider)
}
