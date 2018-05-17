package net.zerochain.Client;

import net.zerochain.Transaction.TransactionEntity;

public interface IClientService {
	void setClient();
	void sendClient();
	void sendTransaction(TransactionEntity transactionEntity);
	TransactionEntity createTransaction(String data);
	void sendTransactions(long timeToSend);
}