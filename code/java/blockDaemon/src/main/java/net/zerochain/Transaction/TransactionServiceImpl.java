package net.zerochain.Transaction;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import net.zerochain.TransactionDAO.ITransactionDAO;
import net.zerochain.ClientDAO.IClientDAO;
import net.zerochain.resources.Utils;
import net.zerochain.resources.crypto.asymmetric.*;
import net.zerochain.Client.ClientEntity;
import net.zerochain.Transaction.TransactionEntity;
import java.util.List;
import java.util.ArrayList;

import java.sql.Timestamp;

@Service("transactionService")
public class TransactionServiceImpl implements ITransactionService{
	@Autowired 
	private ITransactionDAO iTransactionDAO;

	@Autowired
	private IClientDAO iClientDAO;

	@Override
	public boolean verifyTransactionWithTime(TransactionEntity transactionEntity)
	{
        AsymmetricSigning algo = new EDDSA();
		Timestamp minerTime = Utils.getTimestamp();
		String public_key = iClientDAO.getClientPublic_key(transactionEntity.getClientID());
		boolean correctTransactionHash = Utils.verifyHash(transactionEntity.getClientID()+transactionEntity.getData()+Utils.timestampToString(transactionEntity.getTimestamp()), transactionEntity.getHash_msg());
        boolean signedCorrectly = false;
        if(algo.verifyKey(public_key))
        {
            signedCorrectly = algo.verifySignature(public_key, transactionEntity.getSign(), transactionEntity.getHash_msg());
        }
        boolean validTransaction = correctTransactionHash && signedCorrectly && Utils.inTime(minerTime,transactionEntity.getTimestamp());
        return validTransaction;
	}

    @Override
    public boolean verifyTransactionWithoutTime(TransactionEntity transactionEntity)
    {
        AsymmetricSigning algo = new EDDSA();
        String public_key = iClientDAO.getClientPublic_key(transactionEntity.getClientID());
        boolean correctTransactionHash = Utils.verifyHash(transactionEntity.getClientID()+transactionEntity.getData()+Utils.timestampToString(transactionEntity.getTimestamp()), transactionEntity.getHash_msg());
        boolean signedCorrectly = false;
        if(algo.verifyKey(public_key))
        {
            signedCorrectly = algo.verifySignature(public_key, transactionEntity.getSign(), transactionEntity.getHash_msg());
        }
        boolean validTransaction = correctTransactionHash && signedCorrectly;
        return validTransaction;
    }


    @Override
    public List<TransactionEntity> getTwoHundredTransactions()
    {
        return iTransactionDAO.getTwoHundredTransactions();
    }

    @Override
    public void updateTransactionsToPending(List<TransactionEntity> transactions)
    {
        iTransactionDAO.updateTransactionsToPending(transactions);
    }

    @Override
    public List<String> verifyTransactionsWithTime(List<TransactionEntity> transactions)
    {
        List<String> ths = new ArrayList<String>();
        int i = 0;
        while(i < transactions.size())
        {
            TransactionEntity t = transactions.get(i);
            if(verifyTransactionWithTime(t))
            {
                ths.add(t.getHash_msg());
                i++;
            }
            else
            {
                transactions.remove(i);
            }
        }
        return ths;
    }

    @Override
    public List<String> verifyTransactionsWithoutTime(List<TransactionEntity> transactions)
    {
        List<String> ths = new ArrayList<String>();
        int i = 0;
        while(i < transactions.size())
        {
            TransactionEntity t = transactions.get(i);
            if(verifyTransactionWithoutTime(t))
            {
                ths.add(t.getHash_msg());
                i++;
            }
            else
            {
                transactions.remove(i);
            }
        }
        return ths;
    }

}
