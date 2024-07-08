//	Miner API:
//	 version: 0.1.0
//	 title: Miner API
//	Schemes: http, https
//	BasePath: /
//	Produces:
//	  - application/json
//
// swagger:meta
package miner

import (
	"0chain.net/chaincore/transaction"
)

// swagger:parameters PutTransaction GetTxnFees
type TransactionParams struct {
	// Transaction Data
	//
	// in: body
	// required: true
	Transaction transaction.Transaction `json:"transaction"`
}
