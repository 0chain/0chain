//	Sharder API:
//	 version: 0.1.0
//	 title: Sharder API
//	Schemes: http, https
//	BasePath: /
//	Produces:
//	  - application/json
//
// swagger:meta
package sharder

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
)

// swagger:model BlockResponse
type BlockResponse struct {
	// Will be returned if full block is requested.
	Block *block.Block `json:"block"`

	// Will be returned if only block summary is requested.
	Header *block.BlockSummary `json:"header"`

	// Will be returned if only merkle tree is requested.
	MerkleTree []string `json:"merkle_tree"`
}

// swagger:model HealthCheckResponse
type HealthCheckResponse struct {
	// Build tag of the image used to deploy the sharder.
	BuildTag string `json:"build_tag"`

	// Uptime of the sharder in Nanoseconds.
	Uptime  string `json:"uptime"`

	// Should always be "sharder"
	NodeType string `json:"node_type"`

	// Latest Finalized Block known to the sharder.
	Chain ChainInfo `json:"chain"`
}


// swagger:model ConfirmationResponse
type ConfirmationResponse struct {
	// Confirmation of the transaction.
	//
	// required: false
	Confirmation *transaction.Confirmation `json:"confirmation"`

	// Error message if any.
	//
	// required: false
	Error error `json:"error"`

	// Latest finalized block known to the sharder.
	//
	// required: false
	LatestFinalizedBlock *block.BlockSummary `json:"latest_finalized_block"`
}
