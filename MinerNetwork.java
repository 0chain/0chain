import java.util.ArrayList;
import java.util.Arrays;

public class MinerNetwork {
	private Miner network[][];
	private int secondary;
	private Blockchain chain;
	private double quorum;
	private int ddos[];
	private int maxTransactionTime;
	
	/**
	 * Creates a new miner network with each miner given 10.0 credits initially, 
	 * and sets which miners will be experience a ddos. The miner network is created
	 * as seen below
	 * [primary(0)]		...		[primary(p)]
	 * [backup(0,0)]	...		[backup(p,0)]
	 * [backup(0,s)]	...		[backup(p,s)]
	 * @param p the number of primary miners
	 * @param s the number of backup miners for each primary
	 * @param q the number of signatures need to add a new block to the chain
	 * @param dos an array of miners that will experience a ddos
	 */
	public MinerNetwork(int p, int b, double q, int dos[])
	{
		int i, j;
		network = new Miner[p][b + 1];
		Block g = new Block("First Block".getBytes());
		g.createCurrentHash();
		ArrayList<Account> accounts = new ArrayList<Account>();
		this.secondary = b;
		quorum = q;
		chain = new Blockchain();
		chain.addBlock(g);
		ddos = dos;
		maxTransactionTime = 300;
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length; j++)
			{
				network[i][j] = new Miner(g, 10.0);
				accounts.add(network[i][j].getAccount());
			}
		}
		
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length; j++)
			{
				network[i][j].getClient().addAccountsToLedger(accounts);
			}
		}
	}
	
	/**
	 * Creates a new miner network with each miner given 10.0 credits initially, 
	 * and sets which miners will be experience a ddos. The miner network is created
	 * as seen below
	 * [primary(0)]		...		[primary(p)]
	 * [backup(0,0)]	...		[backup(p,0)]
	 * [backup(0,s)]	...		[backup(p,s)]
	 * @param p the number of primary miners
	 * @param s the number of backup miners for each primary
	 * @param q the number of signatures need to add a new block to the chain
	 */
	public MinerNetwork(int p, int s, double q)
	{
		int i, j;
		network = new Miner[p][s + 1];
		Block g = new Block("First Block".getBytes());
		g.createCurrentHash();
		ArrayList<Account> accounts = new ArrayList<Account>();
		this.secondary = s;
		quorum = q;
		maxTransactionTime = 300;
		chain = new Blockchain();
		chain.addBlock(g);
		ddos = new int[0];
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length; j++)
			{
				network[i][j] = new Miner(g, 10.0);
				accounts.add(network[i][j].getAccount());
			}
		}
		
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length; j++)
			{
				network[i][j].getClient().addAccountsToLedger(accounts);
			}
		}
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
	
	/**
	 * This method returns a miner in the network.
	 * @param i the integer to be modded to determine which miner in the network to return
	 * @return miner in that position
	 */
	public Miner getMiner(int i)
	{
		int x = i%(network.length * network[0].length);
		return network[x%network.length][x/network.length];
	}
	
	/**
	 * This method creates a transaction from the miner in position x to the miner in position y.
	 * @param x the miner who creates the transaction
	 * @param y the miner who receives the transaction
	 * @param creds the amount transfered
	 * @return the transaction from x to y for creds
	 */
	public Transaction createTransaction(int x, int y, double creds)
	{
		Transaction t = getMiner(x).createTransaction(getMiner(y).getClient().getPublickey(), creds);
		return t;
	}
	
	/**
	 * This method adds a transaction to all the miner's transaction pool 
	 * @param t transaction to be added to the pools
	 */
	public void acceptTransaction(Transaction t)
	{
		for(int i = 0; i < network.length; i++)
		{
			for(int j = 0; j < network[0].length; j++)
			{
				if(!isDDOS(i,j))
				{
					network[i][j].addTransaction(t);
				}
			}
		}
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
	
	/**
	 * This method has the network run through one round of block creation and all the rounds of verification
	 * to add a new block to the chain.
	 * @param start the column of miners that start the round
	 */
	public void singleRound(int start)
	{
		int i;
		ArrayList<Block> temp = roundStart(start%network.length);
		for(i = 1; i < network.length;i++)
		{
			//System.out.println("Verification round "+i);
			temp = verifyRound((start + i)%network.length, temp);
		}
		//System.out.println();
		decideBlocks(temp);
	}
	
	/**
	 * This method has the miners in a certain column create a block as long as they aren't experiencing a ddos
	 * @param col the column of miners that create the blocks
	 * @return an arraylist of blocks created by the miners
	 */
	public ArrayList<Block> roundStart(int col)
	{
		int minPerCol = secondary + 1;
		ArrayList<Block> blocks = new ArrayList<Block>();
		
		for(int i = 0; i < minPerCol; i++)
		{
			if(!isDDOS(col,i))
			{
				//blocks[i] = network[col][i].createBlock();
				//long generationTime = System.currentTimeMillis();
				blocks.add(network[col][i].createBlock(maxTransactionTime));
				//System.out.println("Time to create one block: "+ (System.currentTimeMillis()-generationTime));
			}
		}
		return blocks;
	}
	
	/**
	 * This method has the miners of a certain column verify a block that has been created
	 * @param col the column of miners that will verify the block
	 * @param b the block to be verified
	 * @return an arraylist of blocks either verified or just passed on to the next round
	 */
	public ArrayList<Block> verifyRound(int col, ArrayList<Block> b)
	{
		int minPerCol = secondary + 1;
		ArrayList<Block> blocks = new ArrayList<Block>();
		
		//long startTime = System.currentTimeMillis();
		Block working = consolidateBlocks(b);
		//System.out.println("Consolidation took "+(System.currentTimeMillis()-startTime));
		
		//startTime = System.currentTimeMillis();
		for(int i = 0; i < minPerCol; i++)
		{
			if(!isDDOS(col, i))
			{
				Block temp = working.cloneBlock();
				//long blockTime = System.currentTimeMillis();
				boolean blockVerified = network[col][i].verifyBlock(temp);
				//System.out.println("Time to verifiy block "+(System.currentTimeMillis()-blockTime));
				
				//long signTime = System.currentTimeMillis();
				//boolean signaturesVerified = network[col][i].verifySignatures(working);
				//System.out.println("Time to verifiy signatures "+(System.currentTimeMillis()-signTime));
				if(blockVerified)
				{
					//long signTime = System.currentTimeMillis();
					temp.addSign(network[col][i].getClient().getPrivatekey(), network[col][i].getClient().getPublickey());
					blocks.add(temp);
					//System.out.println("Time to sign block "+(System.currentTimeMillis()-signTime));
				}
				else
				{
					blocks.add(temp);
				}
			}
		}
		//System.out.println("Replay took "+(System.currentTimeMillis()-startTime));
		
		return blocks;
	}
	
	/**
	 * WORKING ON
	 * @param b
	 * @return
	 */
	public Block consolidateBlocks(ArrayList<Block> b)
	{
		Block working;
		if(b.size() >= 3)
		{
			boolean fNs = b.get(0).sameBlock(b.get(1));
			boolean fNt = b.get(0).sameBlock(b.get(2));
			boolean sNt = b.get(1).sameBlock(b.get(2));
			if(fNs || fNt || !sNt)
			{
				working = b.get(0).cloneBlock();
				working.addSameBlockSign(b.get(1));
				working.addSameBlockSign(b.get(2));
			}
			else
			{
				working = b.get(1).cloneBlock();
				working.addSameBlockSign(b.get(0));
				working.addSameBlockSign(b.get(2));
			}
		}
		else
		{
			working = b.get(0);
			for(int i = 1; i < b.size(); i++)
			{
				working.addSameBlockSign(b.get(i));
			}
		}
		return working;
	}
	
	/**
	 * This method determines which block to try and add to the blockchain
	 * @param b a list of blocks to choose from
	 */
	public void decideBlocks(ArrayList<Block> b)
	{
		Block working = consolidateBlocks(b);
		if(working.getTransactions().size() > 0)
		{
			for(int i = 0; i < network.length; i++)
			{
				for(int j = 0; j < network[0].length; j++)
				{
					if(!isDDOS(i,j))
					{
						network[i][j].addNewBlock(working, (network.length * (network[0].length)), getQuorum());
					}
				}
			}
			addNewBlock(working);
		}
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
	
	/**
	 * This method prints the ledger of every miner
	 */
	public void printLedgers()
	{
		for(int i = 0; i < network[0].length; i++)
		{
			for(int j = 0; j < network.length; j++)
			{
				network[j][i].getClient().getLedger().printLedger();
			}
		}
	}
	
	/**
	 * This method determines if the chain in the miner network will added a new block to the chain
	 * @param b new block to be added
	 */
	public void addNewBlock(Block b)
	{
		if(Arrays.equals(chain.getCurrentHash(), b.getPreviousBlock()) && b.verifySignatures() && b.getSignatures().size()/(double)(network.length*(network[0].length)) > getQuorum())
		{
			chain.addBlock(b);
		}
	}
	
	/**
	 * This method returns the chain of the miner network
	 * @return the blockchain
	 */
	public Blockchain getChain()
	{
		return chain;
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
			if(ddos[i]%network.length == col && ddos[i]/network.length == row)
			{
				bad = true;
			}
		}
		return bad;
	}
	
	/**
	 * This method prints an array that matches the miner network.
	 * Each interger in the array shows the number of blocks in that 
	 * miner's chain
	 */
	public void printMinerBlocks()
	{
		int i, j;
		for(i = 0; i < network[0].length; i++)
		{
			for(j = 0; j < network.length; j++)
			{
				System.out.print(network[j][i].getChain().getLength()+"\t");
			}
			System.out.println();
		}
	}
	
}
