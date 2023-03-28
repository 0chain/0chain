package transaction

const (
	TxnSuccess = 1 // Indicates the transaction is successful in updating the state or smart contract
	TxnError   = 2 // Indicates a transaction has errors during execution
	TxnFail    = 3 // Indicates a transaction has failed to update the state or smart contract
)
