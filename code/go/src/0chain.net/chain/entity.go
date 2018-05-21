package chain

import (
	"context"
	"fmt"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*MAIN_CHAIN - the main 0chain.net blockchain id */
const MAIN_CHAIN = "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe" // TODO:

/*ServerChainID - the chain this server is responsible for */
var ServerChainID = ""

/*ErrSupportedChain error for indicating which chain is supported by the server */
var ErrSupportedChain error

/*SetServerChainID  - set the chain this server is responsible for processing */
func SetServerChainID(chain string) {
	if chain == "" {
		ServerChainID = MAIN_CHAIN
	} else {
		ServerChainID = chain
	}
	ErrSupportedChain = common.NewError("supported_chain", fmt.Sprintf("chain %v is not supported by this server", ServerChainID))
}

/*GetServerChainID - get the chain this server is responsible for processing */
func GetServerChainID() string {
	if ServerChainID == "" {
		return MAIN_CHAIN
	}
	return ServerChainID
}

/*Chain - data structure that holds the chain data*/
type Chain struct {
	datastore.IDField
	datastore.CreationDateField
	ClientID      string `json:"client_id"`    // Client who created this chain
	LatestBlockID string `json:"latest_block"` // Latest block on the chain the program is aware of
}

/*GetEntityName - implementing the interface */
func (c *Chain) GetEntityName() string {
	return "chain"
}

/*Validate - implementing the interface */
func (c *Chain) Validate(ctx context.Context) error {
	if c.ID == "" {
		return common.InvalidRequest("chain id is required")
	}
	if c.ClientID == "" {
		return common.InvalidRequest("client id is required")
	}
	return nil
}

/*Read - datastore read */
func (c *Chain) Read(ctx context.Context, key string) error {
	return datastore.Read(ctx, key, c)
}

/*Write - datastore read */
func (c *Chain) Write(ctx context.Context) error {
	return datastore.Write(ctx, c)
}

/*Delete - datastore read */
func (c *Chain) Delete(ctx context.Context) error {
	return datastore.Delete(ctx, c)
}

/*ChainProvider - entity provider for chain object */
func ChainProvider() interface{} {
	c := &Chain{}
	c.InitializeCreationDate()
	return c
}

/*ValidChain - Is this the chain this server is supposed to process? */
func ValidChain(chain string) error {
	result := chain == ServerChainID || (chain == "" && ServerChainID == MAIN_CHAIN)
	if result {
		return nil
	}
	return ErrSupportedChain
}
