package net.zerochain.Transaction;

import net.zerochain.Response.Response;

public interface ITransactionService {
	void saveTransaction(TransactionEntity transactionEntity);
	boolean lookupTransaction(TransactionEntity transactionEntity);
	Response verifyNewTransaction(TransactionEntity transactionEntity);
}
