package net_0chain;


import java.util.ArrayList;
import java.util.Arrays;
import java.util.LinkedList;
import java.util.Queue;

public class MinerProtocol0 extends Miner{
	private Queue<Block> waitingVerification;
	private ArrayList<Block> waitingConfirmation;
	
	/**
	 * Creates a miner with a Client, a pool of transactions, and a blockchain.
	 */
	public MinerProtocol0()
	{
		super();
		waitingVerification = new LinkedList<Block>();
		waitingConfirmation = new ArrayList<Block>();
	}
	
	/**
	 * Creates a miner with a Client, a pool of transactions, and a blockchain.
	 * The client's account is initialized to a balance of d
	 * @param d the balance of the Client's Account
	 */
	public MinerProtocol0(double d)
	{
		super(d);
		//waitingVerification = new LinkedList<Block>();
	}
	
	/**
	 * Creates a miner with a Client, a pool of transactions, and a blockchain.
	 * The client's account is initialized to a balance of d and the blockchain's
	 * genesis block is set to b
	 * @param b the genesis block for the blockchain
	 * @param d the balance of the Client's Account
	 */
	public MinerProtocol0(Block b, double d)
	{
		super(b,d);
		waitingVerification = new LinkedList<Block>();
		waitingConfirmation = new ArrayList<Block>();
	}
	
	
	/**
	 * This method creates a new block with a time limit. The miner has that time 
	 * to add transactions to the block before it has to create a hash for the block
	 * and signed it.
	 * @param maxTime the max amount of time in milliseconds the miner can add transactions to a block
	 * @return a new Block signed by the miner
	 */
	public Block createBlock(long maxTime)
	{
		long start = System.currentTimeMillis();
		Block b = new Block();
		int i = 0;
		while(i < getTransactionPool().size() && (System.currentTimeMillis() - start) < maxTime/2)
		{
			Transaction temp = getTransactionPool().get(i);
			if(getClient().getLedger().validTransaction(temp) && temp.isSignatureValid() && temp.hashValid())
			{
				b.addTransaction(temp);
			}
			i++;
		}
		moveTransactionsToPending(b);
		b.setPreviousBlock(getChain().getCurrentHash());
		b.createCurrentHash();
		signBlock(b);
		//waitingConfirmation.add(b);
		while ((System.currentTimeMillis() - start) < maxTime);
		return b;
	}
	
	public Block createBadBlock(long maxTime, byte[] previousHash, ArrayList<Transaction> t)
	{
		long start = System.currentTimeMillis();
		Block b = new Block();
		for(int i = 0; i < t.size(); i++)
		{
			b.addTransaction(t.get(i));
		}
		b.setPreviousBlock(previousHash);
		b.createCurrentHash();
		signBlock(b);
		while ((System.currentTimeMillis() - start) < maxTime);
		return b;
	}
	
	public Block createBlock(long maxTime, ArrayList<Block> bset)
	{
		long start = System.currentTimeMillis();
		Block b = new Block();
		int i = 0;
		Block d = consolidateBlocks(bset);
		moveTransactionsToPending(d);
		while(i < getTransactionPool().size() && (System.currentTimeMillis() - start) < maxTime/2)
		{
			Transaction temp = getTransactionPool().get(i);
			if(getClient().getLedger().validTransaction(temp) && temp.isSignatureValid() && temp.hashValid())
			{
				b.addTransaction(temp);
			}
			i++;
		}
		moveTransactionsToPending(b);
		if(!(waitingVerification.size() == 0))
		{
			b.setPreviousBlock(d.getCurrentHash());
		}
		else
		{
			b.setPreviousBlock(getChain().getCurrentHash());
		}
		b.createCurrentHash();
		signBlock(b);
		
		while ((System.currentTimeMillis() - start) < maxTime);
		return b;
	}
	
	/**
	 * This method verifies that a new block's previous hash is equal to the current block of the chain
	 * @param b the new block
	 * @return true if the new block's previous hash is equal to the current block's hash
	 */
	public boolean verifyNewBlockHash(Block b)
	{
		return Arrays.equals(b.getPreviousBlock(), getChain().getCurrentHash()) || (waitingConfirmation.size() > 0 && Arrays.equals(waitingConfirmation.get(waitingConfirmation.size() - 1).getCurrentHash(), b.getPreviousBlock())) || (getQueueCount() > 0 && Arrays.equals(b.getPreviousBlock(), waitingVerification.peek().getCurrentHash()));
	}
	
	/**
	 * This method verifies a block that has been generated. The new block's previous hash
	 * is compared to the current block in the chain. Then all the transactions are verified
	 * by the ledger of the client (to make sure each transaction has sufficient funds) and the
	 * signature of the transaction. If all the transactions are valid the block is verified.
	 * @param b the new block to verify 
	 * @return true if the block is verified by the miner; false otherwise
	 */
	public boolean verifyBlock(Block b, long maxTime)
	{
		long start = System.currentTimeMillis();
		boolean verify = true;
		if(verifyNewBlockHash(b) && b.verifySignatures() && b.getTransactions().size() > 0)
		//if(b.verifySignatures() && b.getTransactions().size() > 0)	
		{
			ArrayList<Transaction> temp = b.getTransactions();
			for(int i = 0; i < temp.size() && verify; i++)
			{
				if(getClient().getLedger().validTransaction(temp.get(i)) && temp.get(i).isSignatureValid() && temp.get(i).hashValid())
				{
					verify = true;
				}
				else
				{
					verify = false;
				}
			}
		}
		else
		{
			verify = false;
		}
		
		if(verify)
		{
			waitingConfirmation.add(b);
		}
		while ((System.currentTimeMillis() - start) < maxTime);
		
		return verify;
	}
	
	/**
	 * This method adds a new block to the miner's blockchain.
	 * @param b the bew block
	 * @param numMiners the number of miners in the miner pool
	 * @param q the quorum need to successfully add a new block to the chain
	 */
	public void addNewBlock(ArrayList<Block> bset, int numMiners, double q)
	{
		Block b = consolidateBlocks(bset);
		boolean found = false;
		for(int i = 0; i < waitingConfirmation.size() && !found; i++)
		{
			if(b.sameBlock(waitingConfirmation.get(i)))
			{
				found = true;
				waitingConfirmation.remove(i);
			}
		}
		
		if(verifyNewBlockHash(b) && b.verifySignatures() && b.getSignatures().size()/(double)numMiners > q)
		{
			getClient().getLedger().updateAll(b.getTransactions());
			deleteTransactions(b);
			getChain().addBlock(b);
		}
		
		else
		{
			moveTransactionsToPool(b);
			waitingVerification.clear();
		}
	}
	
	
	public void addBlockToQueue(ArrayList<Block> bset)
	{
		waitingVerification.add(consolidateBlocks(bset));
	}
	
	public Block getBlockInQueue()
	{
		return waitingVerification.poll();
	}
	
	public int getQueueCount()
	{
		return waitingVerification.size();
	}
	
	
	/**
	 * WORKING ON
	 * @param b
	 * @return
	 */
	public Block consolidateBlocks(ArrayList<Block> b)
	{
		Block working = new Block();
		if(b.size() > 0)
		{
			working = b.get(0);
		}
		
		for(int j = 1; j < b.size(); j++)
		{
			if(working.sameBlock(b.get(j)))
			{
				working.addSameBlockSign(b.get(j));
			}
		}
		
		return working;
	}
	
 
	
}
