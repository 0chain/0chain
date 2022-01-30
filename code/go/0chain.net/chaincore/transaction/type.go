package transaction

import "0chain.net/core/common"

const (
	TxnTypeSend = 0 // A transaction to send tokens to another account, state is maintained by account

	TxnTypeLockIn = 2 // A transaction to lock tokens, state is maintained on the account and the parent lock in transaction

	// Any txn type that refers to a parent txn should have an odd value
	TxnTypeStorageWrite = 101 // A transaction to write data to the blobber
	TxnTypeStorageRead  = 103 // A transaction to read data from the blobber

	TxnTypeData = 10 // A transaction to just store a piece of data on the block chain

	TxnTypeSmartContract = 1000 // A smart contract transaction type
)

var ErrSmartContractContext = common.NewError("smart_contract_execution_ctx_err", "context deadline")
var ErrInsufficientBalance = common.NewError("insufficient_balance", "Balance not sufficient for transfer")
