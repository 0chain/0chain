import java.security.PublicKey;
import java.util.ArrayList;
import java.util.Arrays;

public class Miner {
	private Client self;
	private ArrayList<Transaction> transactions;
	private Blockchain chain;
	
	/**
	 * Creates a miner with a Client, a pool of transactions, and a blockchain.
	 */
	public Miner()
	{
		self = new Client();
		transactions = new ArrayList<Transaction>();
		chain = new Blockchain();
	}
	
	/**
	 * Creates a miner with a Client, a pool of transactions, and a blockchain.
	 * The client's account is initialized to a balance of d
	 * @param d the balance of the Client's Account
	 */
	public Miner(double d)
	{
		self = new Client(d);
		transactions = new ArrayList<Transaction>();
		chain = new Blockchain();
	}
	
	/**
	 * Creates a miner with a Client, a pool of transactions, and a blockchain.
	 * The client's account is initialized to a balance of d and the blockchain's
	 * genesis block is set to b
	 * @param b the genesis block for the blockchain
	 * @param d the balance of the Client's Account
	 */
	public Miner(Block b, double d)
	{
		self = new Client(d);
		transactions = new ArrayList<Transaction>();
		chain = new Blockchain();
		chain.addBlock(b);
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
	 * This method adds a transaction to the transaction pool of the miner
	 * @param tran the transaction added to the pool
	 */
	public void addTransaction(Transaction tran)
	{
		transactions.add(tran);
	}
	
	/**
	 * This method determines if the miner has all the transactions of a block in its personal
	 * pool. 
	 * @param b the block with transactions to compare 
	 * @return true if the miner has all transactions in its pool; false otherwise
	 */
	public boolean hasAllTransactions(Block b)
	{
		return transactions.containsAll(b.getTransactions());
	}
	
	/**
	 * This method creates a new block with a time limit. The miner has that time 
	 * to add transactions to the block before it has to create a hash for the block
	 * and signed it.
	 * @param maxTime the max amount of time in milliseconds the miner can add transactions to a block
	 * @return a new Block signed by the miner
	 */
	public Block createBlock(int maxTime)
	{
		Block b = new Block();
		int i = 0;
		
		long start = System.currentTimeMillis();
		while(i < transactions.size() && (System.currentTimeMillis() - start) < maxTime)
		{
			Transaction temp = transactions.get(i);
			if(self.getLedger().validTransaction(temp) && temp.isSignatureValid() && temp.hashValid())
			{
				b.addTransaction(temp);
			}
			i++;
		}
		b.setPreviousBlock(chain.getCurrentHash());
		b.createCurrentHash();
		b = signBlock(b);
		return b;
	}
	
	/**
	 * This method verifies that a new block's previous hash is equal to the current block of the chain
	 * @param b the new block
	 * @return true if the new block's previous hash is equal to the current block's hash
	 */
	public boolean verifyNewBlockHash(Block b)
	{
		return Arrays.equals(b.getPreviousBlock(), chain.getCurrentHash());
	}
	
	/**
	 * This method verifies a block that has been generated. The new block's previous hash
	 * is compared to the current block in the chain. Then all the transactions are verified
	 * by the ledger of the client (to make sure each transaction has sufficient funds) and the
	 * signature of the transaction. If all the transactions are valid the block is verified.
	 * @param b the new block to verify 
	 * @return true if the block is verified by the miner; false otherwise
	 */
	public boolean verifyBlock(Block b)
	{
		boolean verify = true;
		if(verifyNewBlockHash(b) && b.getTransactions().size() > 0 && this.hasAllTransactions(b))
		{
			ArrayList<Transaction> temp = b.getTransactions();
			for(int i = 0; i < temp.size() && verify; i++)
			{
				if(self.getLedger().validTransaction(temp.get(i)) && temp.get(i).isSignatureValid() && temp.get(i).hashValid())
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
		return verify;
	}
	
	/**
	 * This method adds the signature of the miner to a new block
	 * @param b the new block the signature will be added to
	 * @return block with the new signature
	 */
	public Block signBlock(Block b)
	{
		b.addSign(self.getPrivatekey(), self.getPublickey());
		return b;
		
	}
	
	/**
	 * This method allows the miner create a transaction with its client
	 * @param to the public key the transaction is going to
	 * @param credsTransfered the amount being transfered
	 * @return a new transaction created by the miner
	 */
	public Transaction createTransaction(PublicKey to, double credsTransfered)
	{
		return self.createTransaction(to, credsTransfered);
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
			for(int j = 0; j < transactions.size() && !found; j++)
			{
				if(transactions.get(j).sameTransaction(temp))
				{
					found = true;
					transactions.remove(j);
				}
			}
		}
	}
	
	/**
	 * This method adds a new block to the miner's blockchain.
	 * @param b the bew block
	 * @param numMiners the number of miners in the miner pool
	 * @param q the quorum need to successfully add a new block to the chain
	 */
	public void addNewBlock(Block b, int numMiners, double q)
	{
		if(verifyNewBlockHash(b) && b.verifySignatures() && b.getSignatures().size()/(double)numMiners > q)
		{
			self.getLedger().updateAll(b.getTransactions());
			deleteTransactions(b);
			chain.addBlock(b);
		}
	}
	
}
