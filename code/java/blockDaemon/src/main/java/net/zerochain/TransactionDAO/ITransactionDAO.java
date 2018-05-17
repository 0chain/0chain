package net.zerochain.TransactionDAO;

import net.zerochain.Transaction.TransactionEntity;
import java.util.List;

public interface ITransactionDAO {
	List<TransactionEntity> getTwoHundredTransactions();
	void updateTransactionsToPending(List<TransactionEntity> transactions);
}
