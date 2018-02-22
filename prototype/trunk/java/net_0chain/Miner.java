package net_0chain;

import java.security.PublicKey;
import java.util.ArrayList;

public class Miner {
	private Client self;
	private ArrayList<Transaction> transactionPool;
	private Blockchain chain;
	private ArrayList<Transaction> pendingCon;
	private ShuffleProtocol shuffleProto;
	private Integer minerID;
	
	public Miner()
	{
		self = new Client();
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		pendingCon = new ArrayList<Transaction>();
		shuffleProto = new ShuffleProtocol();
		minerID = null;
	}
	
	public Miner(double d)
	{
		self = new Client(d);
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		pendingCon = new ArrayList<Transaction>();
		shuffleProto = new ShuffleProtocol();
		minerID = null;
	}
	public Miner(Block b, double d)
	{
		self = new Client(d);
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		chain.addBlock(b);
		pendingCon = new ArrayList<Transaction>();
		shuffleProto = new ShuffleProtocol();
		minerID = null;
	}
	
	public Miner(Block b, double d, Integer id)
	{
		self = new Client(d);
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		chain.addBlock(b);
		pendingCon = new ArrayList<Transaction>();
		shuffleProto = new ShuffleProtocol();
		minerID = id;
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
	 * This method returns the ID of the miner
	 * @return ID of the miner
	 */
	
	public Integer getMinerID() {
		return minerID;
	}
	
	/**
	 * This method returns the ShuffleProtocol object of the miner
	 * @return ShuffleProtocol object of the miner
	 */
	
	public ShuffleProtocol getShuffleProto() {
		return shuffleProto;
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
		}
	}
	
	/**
	 * This method calls the functions from the ShuffleProtocol class. The function involves
	 * generating the random number for each miner and signing the hash of the random number
	 * This method is invoked from the MinerNetworkProtocols class.
	 * @param minerID the miner ID 
	 */
	
	public void minerShuffleProtocolRun(int minerID)
	{
		shuffleProto.createRandHash(minerID);
		shuffleProto.addSignHashedRand(self.getPrivatekey(), self.getPublickey());
	}
	
	/**
	 * This method updates the miner info for the ShuffleProtocol class.
	 @param minerID the miner ID
	 @param protoObj the ShuffleProtocol object for each miner
	 */
	public void updateShuffleProtoInfo(int minerID, ShuffleProtocol protoObj)
	{
	    shuffleProto.updateTable(minerID,protoObj);  
	}
	
	/**
	 * This method prints the miner info which all the miners has in the network.
	 */
	public void printRandProto()
	{
		System.out.println("Has the contents : ");
		shuffleProto.printSignHashRandNum();
		
	}
	
	/**
	 * This method verifies the miner's signature of the signed hash which the miner sends
	 * in the ShuffleProtocol.
	 * @return verified whether the signatures are true or false
	 */
	public boolean minerVerifySignHash()
	{
		return shuffleProto.verifySignatures();
	}
	
	/**
	 * This method verifies whether the hash of the random number sent and the random 
	 * numbers matches in the random number protocol.
	 * @return verified whether the hash of the random number matches with the random 
	 * number generated by each miner.
	 */
	public boolean minerVerifyRandHash()
	{
		return shuffleProto.verifyRandHashes();
	}
	
	/**
	 * This method calculates the final random number which is used for shuffling the miners.
	 * @return the final random number for shuffling miners
	 */
	public byte[] minerConcatRandNum()
	{
		return shuffleProto.concatRandNum();
	}
	
	/**
	 * This method has the bench miners get access to the final random number which is used for 
	 * shuffling the miners. All the miners including the bench miners does the shuffling protocol.
	 */
	public void benchMinerSetFinalRand(byte[] finalRand)
	{
		shuffleProto.benchSetFinalRand(finalRand);
	}
	
	/**
	 * This method returns the final shuffled array with the new shuffled positions 
	 * for each miner for the next round.
	 * @param p the number of primary miners
	 * @param s the number of secondary miners
	 * @param b the number of bench miners
	 * @return the 2D array which has the new shuffled positions for each miner.
	 */
	public int[][] minerShufflePositions(int p, int s, int b)
	{
		return shuffleProto.shufflePositions(p,s,b);
	}
	
}
