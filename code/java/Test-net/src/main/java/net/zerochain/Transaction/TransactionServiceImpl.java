package net.zerochain.Transaction;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import net.zerochain.TransactionDAO.ITransactionDAO;
import net.zerochain.ClientDAO.IClientDAO;
import net.zerochain.resources.Utils;
import net.zerochain.resources.crypto.asymmetric.*;
import net.zerochain.Response.Response;
import net.zerochain.Client.ClientEntity;
import net.zerochain.Transaction.TransactionEntity;

import java.sql.Timestamp;

@Service("transactionService")
public class TransactionServiceImpl implements ITransactionService{
	@Autowired 
	private ITransactionDAO iTransactionDAO;

	@Autowired
	private IClientDAO iClientDAO;
	
	@Override
	public void saveTransaction(TransactionEntity transactionEntity) {
		iTransactionDAO.saveTransaction(transactionEntity);
		
	}

	@Override
	public boolean lookupTransaction(TransactionEntity transactionEntity)
	{
		return iTransactionDAO.lookupTransaction(transactionEntity);
	}

	@Override
	public Response verifyNewTransaction(TransactionEntity transactionEntity)
	{
        AsymmetricSigning algo = new EDDSA();
		Response response = new Response();
		Timestamp minerTime = Utils.getTimestamp();
		ClientEntity clientEntity = new ClientEntity("",transactionEntity.getClientID());
		String public_key = "";
		boolean isRegistered = iClientDAO.lookupClient(clientEntity);
		if(isRegistered)
		{
			public_key = iClientDAO.getClientPublic_key(clientEntity);
		}
		boolean correctTransactionHash = isRegistered && Utils.verifyHash(transactionEntity.getClientID()+transactionEntity.getData()+Utils.timestampToString(transactionEntity.getTimestamp()), transactionEntity.getHash_msg());
        boolean signedCorrectly = false;
        if(algo.verifyKey(public_key))
        {
            signedCorrectly = algo.verifySignature(public_key, transactionEntity.getSign(), transactionEntity.getHash_msg());
        }
        boolean freshTransaction = correctTransactionHash && signedCorrectly && !iTransactionDAO.lookupTransaction(transactionEntity);
        boolean validTransaction = freshTransaction && Utils.inTime(minerTime,transactionEntity.getTimestamp());

        if(transactionEntity.getClientID().equals("") || transactionEntity.getData().equals("") || transactionEntity.getTimestamp().equals(new Timestamp(1L)) || transactionEntity.getHash_msg().equals("") || transactionEntity.getSign().equals(""))
        {
            response.setName("Error");
            response.setMessage("JSON not filled in correctly");
        }
        else if(validTransaction)
        {
        	response.setName("Success");
        	response.setMessage("Transaction accepted");
        }
        else if(!isRegistered)
        {
        	response.setName("Error");
        	response.setMessage("Not registered");
        }
        else if(!correctTransactionHash)
        {
        	response.setName("Error");
        	response.setMessage("Bad transaction hash");
        }
        else if(!signedCorrectly)
        {
            response.setName("Error");
            response.setMessage("Signature is just an X");
        }
        else if(!freshTransaction)
        {
        	response.setName("Error");
        	response.setMessage("I've heard this one before");
        }
        else
        {
        	response.setName("Error");
        	response.setMessage("Too old");
        }

        return response;
	}

}
