//  Miner API:
//   version: 0.1.0
//   title: Miner API
//  Schemes: http, https
//  BasePath: /
//  Produces:
//    - application/json
//
// swagger:meta
package miner


// swagger:route POST /v1/transaction/put PutTransaction
// PutTransaction - Put a transaction to the transaction pool. Transaction size cannot exceed the max payload size which is a global configuration of the chain.
//
// Consumes:
//    - application/json
//
// parameters:
//    +name: Transaction
//      in: body
//      required: true
//      schema:
//        properties:
//          client_id:
//            type: string
//          to_client_id:
//            type: string
//          value:
//            type: integer
//          fee:
//            type: integer
//
// responses:
//
//	200:
//	400:
//	500:
