package net.zerochain.Client;

import org.apache.log4j.Logger;

//import org.springframework.stereotype.Component;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.boot.CommandLineRunner;
import net.zerochain.resources.Utils;
import net.zerochain.resources.crypto.asymmetric.*;
import net.zerochain.Transaction.TransactionEntity;
import java.util.List;
import java.util.ArrayList;
import java.sql.Timestamp;
import java.security.KeyPair;
import net.zerochain.Response.Response;
import org.springframework.web.client.RestTemplate;
import org.springframework.http.ResponseEntity;
import org.springframework.http.RequestEntity;
import java.net.ConnectException;
import org.springframework.web.client.ResourceAccessException;

@Service("walletService")
public class ClientServiceImpl implements IClientService {
	private static Logger logger = Logger.getLogger(ClientServiceImpl.class);

    private RestTemplate restTemplate;
    private ClientEntity client;
    private AsymmetricSigning algo;
    private String private_key;

    @Override
    public void setClient()
    {
    	restTemplate = new RestTemplate();
    	algo = new EDDSA();
    	KeyPair keys = algo.createKeys();
    	private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		client = new ClientEntity();
		client.setPublic_key(public_key);
		client.setClientID(client_id);
		logger.info("public key:\n"+public_key);
		logger.info("client id:\n"+client_id);
		logger.info("private key:\n"+private_key);
    }

    @Override 
    public void sendClient()
    {
    	try
    	{
    		ResponseEntity<Response> responseEntity0 = restTemplate.postForEntity("http://localhost:8080/v1/client", client, Response.class);
    	}catch(ResourceAccessException ex)
    	{
    		logger.info("Miner isn't reachable");
	    	try{Thread.sleep(60000);}catch(Exception e){logger.info("Thead interupted");}
    	}
    }

    @Override 
    public void sendTransaction(TransactionEntity transactionEntity)
    {
    	try
    	{
    		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("http://localhost:8080/v1/transaction", transactionEntity, Response.class);
    	}catch(ResourceAccessException ex)
    	{
    		logger.info("Miner isn't reachable");
	    	try{Thread.sleep(60000);}catch(Exception e){logger.info("Thead interupted");}
    	}
    }

    @Override
    public TransactionEntity createTransaction(String data)
    {
		Timestamp timestamp = Utils.getTimestamp();
		String hash = Utils.createHash(client.getClientID()+data+Utils.timestampToString(timestamp));
		String sign = algo.createSignature(private_key, hash);
    	return new TransactionEntity(client.getClientID(),data,timestamp,hash,sign);
    }

	@Override
	public void sendTransactions(long timeToSend)
	{
        long start = System.nanoTime();
        int i = 1;
        //while(System.nanoTime() - start < 300000000000L)
        while(System.nanoTime() - start < timeToSend)
        {
        	TransactionEntity transactionEntity = createTransaction("Daemon! Aaahhhh! Fighter of the nightmon! Aaahhh! Champion of the Sun! Aaahhh! You're a master of karate and friendship for everyone!");
	        sendTransaction(transactionEntity);
	        System.out.print("\rTransactions sent: "+i);
            i++;
        }
        System.out.print("\r");
        logger.info("Transactions sent: "+i);
	}
}
