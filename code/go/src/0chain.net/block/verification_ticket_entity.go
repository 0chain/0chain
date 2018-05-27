package block

import (
	"context"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*VerificationTicket - verification ticket for the block */
type VerificationTicket struct {
	datastore.IDField
	Signature string `json:"signature"`
}

/*BlockVerificationTicket - verification ticket with the block id.
* As VerificationTickets are contained in a block, it doesn't need to have a reference to a block
* However, when the verifiers verify and send the tickets, they need to indicate what block the
* verification ticket is for. So, this wrapper data strcuture is used for that.
 */
type BlockVerificationTicket struct {
	VerificationTicket
	BlockID string `json:"block_id"`
}

/*TODO: Making BlockVerificationTicket an entity for now as N2N handler framework uses entity.
* May be we get rid of the entity requirement later */

/*GetEntityName - implementing the interface */
func (bvt *BlockVerificationTicket) GetEntityName() string {
	return "block_verification_ticket"
}

/*Validate - implementing the interface */
func (bvt *BlockVerificationTicket) Validate(ctx context.Context) error {
	if bvt.ID == "" {
		return common.InvalidRequest("block_verification_ticket id is required")
	}
	return nil
}

/*Read - datastore read */
func (bvt *BlockVerificationTicket) Read(ctx context.Context, key string) error {
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
