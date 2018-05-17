package net.zerochain.Transaction;

import java.util.List;

public interface ITransactionService {
	List<TransactionEntity> getTwoHundredTransactions();
	boolean verifyTransactionWithTime(TransactionEntity transactionEntity);
	boolean verifyTransactionWithoutTime(TransactionEntity transactionEntity);
	List<String> verifyTransactionsWithTime(List<TransactionEntity> transactions);
	List<String> verifyTransactionsWithoutTime(List<TransactionEntity> transactions);
	void updateTransactionsToPending(List<TransactionEntity> transactions);
}
