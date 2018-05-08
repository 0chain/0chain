package net.zerochain.TransactionDAO;

import net.zerochain.Transaction.TransactionEntity;

public interface ITransactionDAO {
	void saveTransaction(TransactionEntity transactionEntity);
	boolean lookupTransaction(TransactionEntity transactionEntity);
}
