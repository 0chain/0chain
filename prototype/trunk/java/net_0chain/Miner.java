package net_0chain;

import java.security.PublicKey;
import java.util.ArrayList;

enum MinerType
{
	PRIMARY, SECONDARY, BENCH;
}

public class Miner {
	private Client self;
	private ArrayList<Transaction> transactionPool;
	private Blockchain chain;
	private ArrayList<Transaction> pendingCon;
	private ShuffleProtocol randproto;
	private Integer minerID;
	private MinerType mtype;
	
	public Miner()
	{
		self = new Client();
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		pendingCon = new ArrayList<Transaction>();
		randproto = new ShuffleProtocol();
		minerID = null;
		mtype = null;
	}
	
	public Miner(double d)
	{
		self = new Client(d);
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		pendingCon = new ArrayList<Transaction>();
		randproto = new ShuffleProtocol();
		minerID = null;
		mtype = null;
	}
	public Miner(Block b, double d)
	{
		self = new Client(d);
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		chain.addBlock(b);
		pendingCon = new ArrayList<Transaction>();
	}
	
	public Miner(Block b, double d, Integer id)
	{
		self = new Client(d);
		transactionPool = new ArrayList<Transaction>();
		chain = new Blockchain();
		chain.addBlock(b);
		pendingCon = new ArrayList<Transaction>();
		randproto = new ShuffleProtocol();
		minerID = id;
		randproto = new ShuffleProtocol();
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
	 * This method returns the type of the miner
	 * @return type of the miner
	 */
	
	public MinerType getMinertype()
	{
		return mtype;
	}
	
	/**
	 * This method sets the type of the miner
	 */
	public void setMinertype(MinerType newval)
	{
		mtype = newval;
	}
	
	
	/**
	 * This method returns the ID of the miner
	 * @return ID of the miner
	 */
	
	public Integer getMinerID() {
		return minerID;
	}
	
	/**
	 * This method returns the RandNumProtocol object of the miner
	 * @return RandNumprotocol object of the miner
	 */
	
	public ShuffleProtocol getRandProto() {
		return randproto;
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
	 * This method calls the functions from the RandNumProtocol class. The function involves
	 * generating the random number for each miner and signing the hash of the random number
	 * This method is invoked from the MinerNetworkProtocols class.
	 * @param minerID the miner ID 
	 * @param credsTransfered the amount being transfered
	 * @return a new transaction created by the miner
	 */
	
	public void minerRandProtocolRun(int minerID)
	{
		randproto.createRandHash(minerID);
		randproto.addSignHashedRand(self.getPrivatekey(), self.getPublickey());
	}
	
	/**
	 * This method updates the miner info for the RandNumProtocol class.
	 @param minerID the miner ID
	 @param protoObj the RandNumProtocol object for each miner
	 */
	public void updateRandProtoInfo(int minerID, ShuffleProtocol protoObj)
	{
	    randproto.updateTable(minerID,protoObj);  
	}
	
	/**
	 * This method prints the miner info which all the miners has in the network.
	 */
	public void printRandProto()
	{
		System.out.println("Has the contents : ");
		randproto.printSignHashRandNum();
	}
	
	/**
	 * This method verifies the signature of the signed hash which the miner sends
	 * in the random number protocol.
	 */
	public void minerVerifySignHash()
	{
		randproto.verifySignatures();
	}
	
	/**
	 * This method verifies whether the hash of the random number sent and the random 
	 * numbers matches in the random number protocol.
	 */
	public void minerVerifyRandHash()
	{
		System.out.println(" verifying the hash and random random numbers and they are: ");
		randproto.verifyRandHashes();
	}
	
	/**
	 * This method calculates the final random number which is used for shuffling the miners.
	 * @return the final random number for shuffling miners
	 */
	public byte[] minerConcatRandNum()
	{
		return randproto.concatRandNum();
	}
	
	/**
	 * This method has the bench miners get access to the final random number which is used for 
	 * shuffling the miners. All the miners including the bench miners does the shuffling protocol.
	 */
	public void benchMinerSetFinalRand(byte[] finalRand)
	{
		randproto.benchSetFinalRand(finalRand);
	}
	
	/**
	 * This method returns the final shuffled array with the new shuffled positions 
	 * for each miner for the next round.
	 @param totalMiners the total number of miners 
	 */
	public void minerShufflePositions(int totalMiners)
	{
		System.out.println("The Miner ID : "+ getMinerID() +" generates the shuffled array position ");
		randproto.shufflePositions(totalMiners);
	}
	
}
