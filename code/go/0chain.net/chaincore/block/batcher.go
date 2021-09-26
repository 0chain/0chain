package block

import (
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"sort"
)

type Batcher interface {
	Batch(b *Block) (ret [][]*transaction.Transaction)
}

type ContentionFreeBatcher struct {
	batchSize int
}

type RankedTx struct {
	rank int
	rset map[datastore.Key]bool
	wset map[datastore.Key]bool
	*transaction.Transaction
}

type ByRank []*RankedTx

func (a ByRank) Len() int           { return len(a) }
func (a ByRank) Less(i, j int) bool { return a[i].rank < a[j].rank }
func (a ByRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (t *RankedTx) HasContention(sets ...map[datastore.Key]bool) bool {
	//there is no contention with empty set, since transaction accesses nothing
	if len(t.wset) == 0 {
		return false
	}

	//check every key for every set given and if anything is found then contention is detected
	for _, set := range sets {
		for k := range set {
			_, ok := t.wset[k]
			if ok {
				return true
			}
		}
	}

	return false
}

func (c *ContentionFreeBatcher) Batch(b *Block) (ret [][]*transaction.Transaction) {
	txns := b.Txns
	accessMap := b.AccessMap
	//we have no parallelization here
	if len(accessMap) == 0 {
		ret = [][]*transaction.Transaction{} //return empty list for block without txs
		for _, txn := range txns {
			//pack transactions in batches 1 tx each
			ret = append(ret, []*transaction.Transaction{txn})
		}
		return
	}

	rankedTxs := rankTransactions(txns, accessMap)
	return c.batchTxs(rankedTxs)

}

//Groups transactions to contention free batches not larger than c.batchSize
func (c *ContentionFreeBatcher) batchTxs(txs []*RankedTx) (ret [][]*transaction.Transaction) {
	currentRank := 0
	currentBatch := make([]*transaction.Transaction, 0, c.batchSize)
	for _, tx := range txs {
		if len(currentBatch) == c.batchSize || currentRank != tx.rank {
			ret = append(ret, currentBatch)
			currentBatch = make([]*transaction.Transaction, 0, c.batchSize)
			currentRank = tx.rank
		}
		currentBatch = append(currentBatch, tx.Transaction)
	}

	//do not forget the last batch if it is not full
	if len(currentBatch) != c.batchSize {
		ret = append(ret, currentBatch)
	}

	return ret
}

//Convert transaction to inner struct holding ranks.
//Transactions with equal rank has no contentions, transaction with smaller rank should be executed before transaction with bigger rank.
//Ranking can be improved in the future, this approach is pretty simple to implement, alas can potentially create a lot of half-empty batches.
//Transactions with different ranks actually can be executed together to pack batches more accurately, more sophisticated rank algo can be used.
//We can extract independent subgraphs of dependencies to rank transactions inside them and process this subgraphs in parallel.
func rankTransactions(txns []*transaction.Transaction, accessMap map[datastore.Key]*AccessList) []*RankedTx {
	rankedTxs := make([]*RankedTx, len(txns))
	for i, txn := range txns {
		rankedTxs[i] = &RankedTx{
			rank:        0,
			rset:        accessMap[txn.GetKey()].Rset(),
			wset:        accessMap[txn.GetKey()].Wset(),
			Transaction: txn,
		}
	}

	for i, tx := range rankedTxs {
		for j := i + 1; j < len(txns); j++ {
			if rankedTxs[j].HasContention(tx.rset, tx.wset) { //If this transaction has contention on R or W set
				if rankedTxs[j].rank <= tx.rank { // If this transaction is not before given, move it further in rank list
					rankedTxs[j].rank = tx.rank + 1
				}
			}
		}
	}
	sort.Sort(ByRank(rankedTxs))
	return rankedTxs
}
