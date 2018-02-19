package net_0chain;

public class MinerNetwork {
	private Blockchain chain;
	private double quorum;
	private int maxTransactionTime;
	public Miner[][] network;
	private int primary;
	private int secondary;
	private int bench;
	private int ddos[];
	private int badGuy[];
	private int totalMiners;
	
	public MinerNetwork(int p, int s, int b)
	{
		Block g = new Block("First Block".getBytes());
		g.createCurrentHash();
		quorum = .50;
		primary = p;
		secondary = s;
		bench = b;
		network = new Miner[p][1+s+b];
		ddos = new int[0];
		badGuy = new int[0];
		chain = new Blockchain();
		chain.addBlock(g);
		maxTransactionTime = 300;
		totalMiners = p*(1+s+b);
	}
	
	public MinerNetwork(int p, int s, int b, double q)
	{
		Block g = new Block("First Block".getBytes());
		g.createCurrentHash();
		primary = p; 
		secondary = s;
		bench = b;
		quorum = q;
		network = new Miner[p][1+s+b];
		ddos = new int[0];
		badGuy = new int[0];
		chain = new Blockchain();
		chain.addBlock(g);
		maxTransactionTime = 300;
		totalMiners = p*(1+s+b);
	}
	
	public MinerNetwork(int p, int s, int b, double q, int[] dos, int[] bg)
	{
		Block g = new Block("First Block".getBytes());
		g.createCurrentHash();
		primary = p; 
		secondary = s;
		bench = b;
		quorum = q;
		network = new Miner[p][1+s+b];
		ddos = dos;
		badGuy = bg;
		chain = new Blockchain();
		chain.addBlock(g);
		maxTransactionTime = 300;
		totalMiners = p*(1+s+b);
	}
	
	/**
	 * This method returns the max amount of time a miner has to collect transactions to create a block
	 * @return max amount of time in milliseconds
	 */
	public int getMaxTransactions()
	{
		return maxTransactionTime;
	}
	
	/**
	 * This method sets the max amount of time a miner has to collect transactions to creata a block
	 * @param mt the max amount of time
	 */
	public void setMaxTransactions(int mt)
	{
		if(mt>0)
		{
			maxTransactionTime = mt;
		}
	}
	
	public void addBlock(Block b)
	{
		chain.addBlock(b);
	}
	
	/**
	 * This method returns the percentage of miners that need to sign a block out of the whole network to added it to the chain
	 * @return percentage to add new block
	 */
	public double getQuorum()
	{
		return quorum;
	}
	
	/**
	 * This method allows the percentage of miners need to add a block to the chain change.
	 * @param q the number quorum need for a block to be accepted
	 */
	public void changeQuorum(double q)
	{
		quorum = q;
	}
	
	public Blockchain getChain()
	{
		return chain;
	}
	
	public int getPrimary()
	{
		return primary;
	}
	
	public int getSecondary()
	{
		return secondary;
	}
	
	public int getBench()
	{
		return bench;
	}
	
	public int getTotalMiners()
	{
		return totalMiners;
	}
	
	public boolean isBadGuy(int col, int row)
	{
		boolean bad = false;
		for(int i = 0; i < badGuy.length; i++)
		{
			if(badGuy[i]%primary == col && badGuy[i]/primary == row)
			{
				bad = true;
			}
		}
		return bad;
	}
	
	/**
	 * This method determines if a miner in a certain row and column are being DDOSed.
	 * @param col the column
	 * @param row the row
	 * @return true if the miner is experiencing a ddos
	 */
	public boolean isDDOS(int col, int row)
	{
		boolean bad = false;
		for(int i = 0; i < ddos.length && !bad; i++)
		{
			if(ddos[i]%primary == col && ddos[i]/primary == row)
			{
				bad = true;
			}
		}
		return bad;
	}
	
	/**
	 * This method returns a miner in the network.
	 * @param i the integer to be modded to determine which miner in the network to return
	 * @return miner in that position
	 */
	public Miner getMiner(int i)
	{
		//		int x = i%(network.length * network[0].length);
		// return network[x%network.length][x/network.length];
		return network[i%primary][i/primary];
	}
	
	/**
	 * This method adds a transaction to the miner pools of the miners who are in the position of the array
	 * @param t the transaction to be added to the pools
	 * @param miner an array of integers that determines what miners receive the transaction
	 */
	public void acceptTransaction(Transaction t, int miner[])
	{
		for(int i = 0; i < miner.length; i++)
		{
			int j = miner[i]%network.length;
			int k = miner[i]/network.length;
			if(!isDDOS(j,k))
			{
				getMiner(miner[i]).addTransaction(t);
			}
		}
	}

}
