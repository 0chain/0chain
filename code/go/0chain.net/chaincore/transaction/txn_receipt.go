package transaction

import "github.com/0chain/0chain/code/go/0chain.net/core/util"

//TxnReceipt - a transaction receipt is a processed transaction that contains the output
type TxnReceipt struct {
	Transaction *Transaction
}

//GetHash - implement interface
func (rh *TxnReceipt) GetHash() string {
	return rh.Transaction.OutputHash
}

/*GetHashBytes - implement Hashable interface */
func (rh *TxnReceipt) GetHashBytes() []byte {
	return util.HashStringToBytes(rh.Transaction.OutputHash)
}

//NewTransactionReceipt - create a new transaction receipt
func NewTransactionReceipt(t *Transaction) *TxnReceipt {
	return &TxnReceipt{Transaction: t}
}
