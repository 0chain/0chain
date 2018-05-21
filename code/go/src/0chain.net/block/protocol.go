package block

import (
	"context"

	"0chain.net/datastore"
	"0chain.net/transaction"
)

const BLOCK_SIZE = 500

/*GenerateBlock - This works on generating a block
* The context should be a background context which can be used to stop this logic if there is a new
* block published while working on this
 */
func (b *Block) GenerateBlock(ctx context.Context) error {
	txns := make([]*transaction.Transaction, BLOCK_SIZE)
	idx := 0
	var txnIterHandler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		txn, ok := qe.(*transaction.Transaction)
		if !ok {
			return true
		}
		if txn.Status != transaction.TXN_STATUS_FREE {
			return true
		}
		txns[idx] = txn
		idx++
		if len(txns) == BLOCK_SIZE {
			// TODO: createBlock(ctx)
			return false
		}
		return true
	}
	err := datastore.IterateCollection(ctx, txnIterHandler, transaction.TransactionProvider)
	return err
}

/*ValidateBlock - given a set of transaction ids within a block, validate the block */
func (b *Block) ValidateBlock(ctx context.Context, txns []interface{}) (bool, error) {
	return true, nil
}

/*UpdateTxnStatusToMined - given a set of transaction ids within a block, update them to mined */
func (b *Block) UpdateTxnStatusToMined(ctx context.Context, txns []interface{}) error {
	transactions, err := datastore.AllocateEntities(BLOCK_SIZE, transaction.TransactionProvider)
	modifiedTxns := make([]datastore.Entity, BLOCK_SIZE)

	if err != nil {
		return err
	}
	for start := 0; start < len(txns); start += BLOCK_SIZE {
		end := start + BLOCK_SIZE
		if end > len(txns) {
			end = len(txns)
		}
		keys := txns[start:end]
		datastore.MultiRead(ctx, keys, transactions)
		ind := 0
		for i := 0; i < end-start; i++ {
			if transactions[i].GetKey() == nil {
				continue
			}
			txn := transactions[i].(*transaction.Transaction)
			txn.Status = transaction.TXN_STATUS_MINED
			modifiedTxns[ind] = txn
			ind++
		}
		if ind > 0 {
			datastore.MultiWrite(ctx, modifiedTxns[:ind]...)
		}
	}
	return nil
}
