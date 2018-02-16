package net_0chain;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.security.PrivateKey;
import java.security.PublicKey;
import java.util.ArrayList;

public class Client {
	private PublicKey pubKey;
	private PrivateKey priKey;
	private Ledger ledger;

	/**
	 * Creates a new client with a public and private key as well as a ledger
	 * with an account for the client.
	 */
	public Client()
	{
		try
		{
			KeyPairGenerator keyGen = KeyPairGenerator.getInstance("RSA");
			KeyPair pair = keyGen.generateKeyPair();
			pubKey = pair.getPublic();
			priKey = pair.getPrivate();
			ledger = new Ledger();
			ledger.addAccount(new Account(pubKey));

		}catch(NoSuchAlgorithmException e)
		{
			System.out.println("Failed to create client");	
		}
	}
	
	/**
	 * Creates a new client with a public and private key as well as a ledger
	 * with an account for the client with a balance of creds.
	 * @param creds the balance the client's account starts with
	 */
	public Client(double creds)
	{
		try
		{
			KeyPairGenerator keyGen = KeyPairGenerator.getInstance("RSA");
			KeyPair pair = keyGen.generateKeyPair();
			pubKey = pair.getPublic();
			priKey = pair.getPrivate();
			ledger = new Ledger();
			if(creds> 0.0)
			{
				ledger.addAccount(new Account(pubKey, creds));
			}
			else
			{
				ledger.addAccount(new Account(pubKey));
			}

		}catch(NoSuchAlgorithmException e)
		{
			System.out.println("Failed to create client");	
		}
	}
	
	/**
	 * This method returns the private key of the client
	 * @return private key
	 */
	public PrivateKey getPrivatekey()
	{
		return priKey;
	}
	
	/**
	 * This method returns the public key of the client
	 * @return public key
	 */
	public PublicKey getPublickey()
	{
		return pubKey;
	}
	
	/**
	 * This method creates a transactions from this client to the
	 * client identified by the public key to with an amount to transfer
	 * from this client's account to the other's account
	 * @param to public key of the client receiving the credits
	 * @param credsTransfered the amount of credits
	 * @return transaction hashed and signed by this client
	 */
	public Transaction createTransaction(PublicKey to, double credsTransfered)
	{
		Transaction newTransaction = new Transaction(to, pubKey, credsTransfered);
		newTransaction.hashTransaction();
		newTransaction.setSign(getPrivatekey());
		return  newTransaction;
	}
	
	/**
	 * This method adds an arraylist of accounts to the ledger of the client
	 * @param newAccounts the list of accounts to add to the ledger
	 */
	public void addAccountsToLedger(ArrayList<Account> newAccounts)
	{
		ledger.addAccounts(newAccounts);
	}
	
	/**
	 * This method returns the account of this client from the ledger
	 * @return account of the client
	 */
	public Account getAccount()
	{
		return ledger.getAccount(pubKey);
	}
	
	/**
	 * This method returns the ledger of the clients
	 * @return ledger
	 */
	public Ledger getLedger()
	{
		return ledger;
	}
	
	/**
	 * This method sets the ledger of the client to the ledger l
	 * @param l the new ledger of the client
	 */
	public void setLedger(Ledger l)
	{
		ledger = l;
	}
	
}
