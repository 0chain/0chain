package net_0chain;

import java.security.PublicKey;
import java.util.ArrayList;

public class Miner {
	private Client self;
	private ArrayList<Transaction> transactionPool;
	private Blockchain chain;
	private ArrayList<Transaction> pendingCon;
	
	public Miner()
	{
		self = new Client();
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		pendingCon = new ArrayList<Transaction>();
	}
	
	public Miner(double d)
	{
		self = new Client(d);
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		pendingCon = new ArrayList<Transaction>();
	}
	
	public Miner(Block b, double d)
	{
		self = new Client(d);
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		chain.addBlock(b);
		pendingCon = new ArrayList<Transaction>();
	}
	
	/**
	 * This method returns the blockchain of the miner
	 * @return blockchain of the miner
	 */
	public Blockchain getChain()
	{
		return chain;
	}
	
	/**
	 * This method returns the client of the miner
	 * @return client of the miner
	 */
	public Client getClient()
	{
		return self;
	}
	
	/**
	 * This method returns the account of the client
	 * @return account of the client
	 */
	public Account getAccount()
	{
		return self.getAccount();
	}
	
	/**
	 * This method allows the miner create a transaction with its client
	 * @param to the public key the transaction is going to
	 * @param credsTransfered the amount being transfered
	 * @return a new transaction created by the miner
	 */
	public Transaction createTransaction(PublicKey to, double credsTransfered)
	{
		return getClient().createTransaction(to, credsTransfered);
	}
	
	/**
	 * This method adds a transaction to the transaction pool of the miner
	 * @param tran the transaction added to the pool
	 */
	public void addTransaction(Transaction tran)
	{
		transactionPool.add(tran);
	}
	
	public void moveTransactionsToPending(Block b)
	{
		ArrayList<Transaction> temp = b.getTransactions();
		for(int i = 0; i < temp.size(); i++)
		{
			if(transactionPool.contains(temp.get(i)))
			{
				pendingCon.add(temp.get(i));
				transactionPool.remove(temp.get(i));
			}
		}
	}
	
	public void moveTransactionsToPool(Block b)
	{
		ArrayList<Transaction> temp = b.getTransactions();
		for(int i = 0; i < temp.size(); i++)
		{
			if(pendingCon.contains(temp.get(i)))
			{
				transactionPool.add(temp.get(i));
				pendingCon.remove(temp.get(i));
			}
		}
	}
	
	public void signBlock(Block b)
	{
		b.addSign(self.getPrivatekey(), self.getPublickey());
	}
	
	public ArrayList<Transaction> getTransactionPool()
	{
		return transactionPool;
	}
	
	/**
	 * This method deletes all the transaction in the transaction pool that 
	 * a new block contains.
	 * @param b the new block
	 */
	public void deleteTransactions(Block b)
	{
		ArrayList<Transaction> temps = b.getTransactions();
		for(int i = 0; i < temps.size(); i++)
		{
			Transaction temp = temps.get(i);
			boolean found = false;
			for(int j = 0; j < pendingCon.size() && !found; j++)
			{
				if(pendingCon.get(j).sameTransaction(temp))
				{
					found = true;
					pendingCon.remove(j);
				}
			}
			
			for(int k = 0; k < transactionPool.size() && !found; k++)
			{
				if(transactionPool.get(k).sameTransaction(temp))
				{
					found = true;
					transactionPool.remove(k);
				}
			}
		}
	}
}
