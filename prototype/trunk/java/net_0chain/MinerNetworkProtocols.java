package net_0chain;
import java.security.PublicKey;
import java.util.ArrayList;
import java.util.Arrays;

public class MinerNetworkProtocols extends MinerNetwork{
	
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
	public MinerNetworkProtocols(int p, int s, int b, double q, int dos[], int bg[])
	{
		super(p,s,b,q,dos,bg);
		int i, j;
		Block g = new Block("First Block".getBytes());
		g.createCurrentHash();
		ArrayList<Account> accounts = new ArrayList<Account>();
		
		
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length; j++)
			{
				network[i][j] = new MinerProtocol0(g, 10.0, createMinerID(i, j));
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
	public MinerNetworkProtocols(int p, int s, int b, double q)
	{
		super(p,s,b,q);
		int i, j;
		network = new MinerProtocol0[p][s + b + 1];
		Block g = new Block("First Block".getBytes());
		g.createCurrentHash();
		ArrayList<Account> accounts = new ArrayList<Account>();
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length; j++)
			{
				network[i][j] = new MinerProtocol0(g, 10.0, createMinerID(i, j));
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
			for(int j = 0; j < network[0].length - getBench(); j++)
			{
				if(!isDDOS(i,j))
				{
					network[i][j].addTransaction(t);
				}
			}
		}
	}
	
	
	/**
	 * This method returns the minerID. The minerIDs are 10000, 10001, 10002, 10003 till 10014
	 * @param rowIndex row index of the miner in the network array
	 * @param columnIndex column index of the miner in the network array
	 * @return the ID of the miner
	 */
	private Integer createMinerID(int row_index, int column_index)
	{
		return ((1+getSecondary()+getBench())*row_index) + (column_index) + 10000;
	}
	
	/**
	 * This method has the network run through one round of block creation and all the rounds of verification
	 * to add a new block to the chain.
	 * @param start the column of miners that start the round
	 */
	public void singleRoundProtocol0(int start)
	{
		int i;
		ArrayList<Block> temp = generate(start%network.length);
		for(i = 1; i < network.length;i++)
		{
			//System.out.println("Verification round "+i);
			temp = verifyRound((start + i)%network.length, temp);
		}
		//System.out.println();
		decideBlocks(temp);
	}
	
	public void singleRoundProtocol1(int start, int preGenCount)
	{
		int i;
		ArrayList<ArrayList<Block>> allGeneratedBlocks = new ArrayList<ArrayList<Block>>();
		Block currentWorking = new Block();
		for(i = 0; i < preGenCount; i++)
		{
			if(i == 0)
			{
				allGeneratedBlocks.add(generate((start+i)%network.length));
				currentWorking = ((MinerProtocol0)network[0][0]).consolidateBlocks(allGeneratedBlocks.get(i));
			}
			else
			{
				allGeneratedBlocks.add(generate((start+i)%network.length, allGeneratedBlocks.get(i)));
				currentWorking = ((MinerProtocol0)network[0][0]).consolidateBlocks(allGeneratedBlocks.get(i));
			}
		}
		ArrayList<Block> temp = new ArrayList<Block>();
		for(i = 0; i < preGenCount;i++)
		{
			temp = allGeneratedBlocks.get(i);
			
			for(int j = 0; j < network.length; j++)
			{
				if(i != j)
				{
					temp = verifyRound(j, temp);
				}
			}
			decideBlocks(temp);
		}
		//System.out.println();
	}
	
	public void singleRoundProtocol2(int timeSlots)
	{
		ArrayList<Block> gen = new ArrayList<Block>();
		ArrayList<Block> working = new ArrayList<Block>();
		for(int i = 0; i < timeSlots; i++)
		{
			if(i == 0)
			{
				gen = generate(i%network.length);
			}
			else
			{
				gen = generate(i%network.length, working);
			}
			working = gen;
			for(int j = 0; j < network.length; j++)
			{
				if(j != (i%network.length))
				{
					ArrayList<Block> verifyBlocks = new ArrayList<Block>();
					if(((MinerProtocol0)network[0][0]).getQueueCount() > 0)
					{
						verifyBlocks = verifyBlockInQueue(j);
						//Block verify = consolidateBlocks(verifyBlocks);
						if(minerInCol(verifyBlocks.get(0).getSignatures().get(0).getPublicKey(), (j+1)%network.length))
						{
							decideBlocks(verifyBlocks);
						}
						else
						{
							addBlocksToQueue((j+1)%network.length, verifyBlocks);
						}
					}
				}
				
			}
			addBlocksToQueue((i+1)%network.length, working);
			
		}
	}
	
	public boolean minerInCol(PublicKey pk, int col)
	{
		boolean inCol = false;
		int minPerCol = getSecondary() + 1;
		for(int i = 0; i < minPerCol && !inCol; i++)
		{
			if(network[col][i].getClient().getPublickey().equals(pk))
			{
				inCol = true;
			}
		}
		return inCol;
	}
	
	/**
	 * This method has the miners in a certain column create a block as long as they aren't experiencing a ddos or bad guys
	 * @param col the column of miners that create the blocks
	 * @return an arraylist of blocks created by the miners
	 */
	public ArrayList<Block> generate(int col)
	{
		int minPerCol = getSecondary() + 1;
		ArrayList<Block> blocks = new ArrayList<Block>();
		
		for(int i = 0; i < minPerCol; i++)
		{
			if(!isDDOS(col,i) && !isBadGuy(col,i))
			{
				//blocks[i] = network[col][i].createBlock();
				//long generationTime = System.currentTimeMillis();
				blocks.add(((MinerProtocol0)network[col][i]).createBlock(getMaxTransactions()));
				//System.out.println("Time to create one block: "+ (System.currentTimeMillis()-generationTime));
			}
			if(isBadGuy(col,i))
			{
				ArrayList<Transaction> temp = new ArrayList<Transaction>();
				Transaction t = network[col][i].createTransaction(network[col][i].getClient().getPublickey(), 2.0);
				temp.add(t);
				blocks.add(((MinerProtocol0)network[col][i]).createBadBlock(getMaxTransactions(), "test".getBytes(), temp));
			}
		}
		return blocks;
	}
	
	public ArrayList<Block> generate(int col, ArrayList<Block> bset)
	{
		int minPerCol = getSecondary() + 1;
		ArrayList<Block> blocks = new ArrayList<Block>();
		
		for(int i = 0; i < minPerCol; i++)
		{
			ArrayList<Block> temp0 = new ArrayList<Block>();
			
			for(int x = 0; x < bset.size(); x++)
			{
				temp0.add(bset.get(i).cloneBlock());
			}
			
			if(!isDDOS(col,i) && !isBadGuy(col,i))
			{
				//blocks[i] = network[col][i].createBlock();
				//long generationTime = System.currentTimeMillis();
				blocks.add(((MinerProtocol0)network[col][i]).createBlock(getMaxTransactions(), temp0));
				//System.out.println("Time to create one block: "+ (System.currentTimeMillis()-generationTime));
			}
			if(isBadGuy(col,i))
			{
				ArrayList<Transaction> temp = new ArrayList<Transaction>();
				Transaction t = network[col][i].createTransaction(network[col][i].getClient().getPublickey(), 2.0);
				temp.add(t);
				blocks.add(((MinerProtocol0)network[col][i]).createBadBlock(getMaxTransactions(), "test".getBytes(), temp));
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
		int minPerCol = getSecondary() + 1;
		ArrayList<Block> blocks = new ArrayList<Block>();
		
		//long startTime = System.currentTimeMillis();
		Block working = ((MinerProtocol0)network[0][0]).consolidateBlocks(b);
		//System.out.println("Consolidation took "+(System.currentTimeMillis()-startTime));
		
		//startTime = System.currentTimeMillis();
		for(int i = 0; i < minPerCol; i++)
		{
			if(!isDDOS(col, i))
			{
				Block temp = working.cloneBlock();
				//long blockTime = System.currentTimeMillis();
				boolean blockVerified = ((MinerProtocol0)network[col][i]).verifyBlock(temp, getMaxTransactions());
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
	 * This method determines which block to try and add to the blockchain
	 * @param b a list of blocks to choose from
	 */
	public void decideBlocks(ArrayList<Block> b)
	{
		for(int i = 0; i < network.length; i++)
		{
			for(int j = 0; j < network[0].length - getBench(); j++)
			{
				if(!isDDOS(i,j))
				{
					((MinerProtocol0)network[i][j]).addNewBlock(b, (network.length * (network[0].length - getBench())), getQuorum());
				}
			}
		}
		addNewBlock(b);
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
	public void addNewBlock(ArrayList<Block> bset)
	{
		Block b = ((MinerProtocol0)network[0][0]).consolidateBlocks(bset);
		if(Arrays.equals(getChain().getCurrentHash(), b.getPreviousBlock()) && b.verifySignatures() && b.getSignatures().size()/(double)(network.length*(network[0].length - getBench())) > getQuorum())
		{
			getChain().addBlock(b);
		}
	}
	
	/**
	 * This method prints an array that matches the miner network.
	 * Each integer in the array shows the number of blocks in that 
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
	
	public void addBlocksToQueue(int col, ArrayList<Block> bset)
	{
		int minPerCol = getSecondary() + 1;
		for(int i = 0; i < minPerCol; i++)
		{
			((MinerProtocol0)network[col][i]).addBlockToQueue(bset);
		}
	}
	
	public ArrayList<Block> verifyBlockInQueue(int col)
	{
		ArrayList<Block> verified = new ArrayList<Block>();
		int minPerCol = getSecondary() + 1;
		for(int i = 0; i < minPerCol; i++)
		{
			verified.add(((MinerProtocol0)network[col][i]).getBlockInQueue());
		}
		
		return verifyRound(col,verified);
	}
	
	/**
	 * WORKING ON
	 * @param b
	 * @return
	 *
	public Block consolidateBlocks(ArrayList<Block> b)
	{
		Block working = b.get(0);
		/**
		boolean found = false;
		ArrayList<byte[]> hashes = new ArrayList<byte[]>();
		for(int i = 0; i < b.size() && !found; i++)
		{
			if(!hashes.contains(b.get(i).getCurrentHash()))
			{
				hashes.add(b.get(i).getCurrentHash());
			}
			else
			{
				found = true;
				working = b.get(i);
			}
		}
		*
		for(int j = 1; j < b.size(); j++)
		{
			if(working.sameBlock(b.get(j)))
			{
				working.addSameBlockSign(b.get(j));
			}
		}
		
		return working;
	}
	*/
	
	/**
	 * This method is called from the runShuffleProtocol() function to keep track of the 
	 * miner information.
	 */
	public void broadcastProtoInfo(Miner minerObj)
	{
		for(int i = 0; i < network.length; i++)
		{
			for(int j = 0; j < network[0].length - getBench(); j++)
			{
				network[i][j].updateShuffleProtoInfo(minerObj.getMinerID(),minerObj.getShuffleProto());
 			}
		}
	}
	
	/**
	 * This method prints the initial active miner set along with the bench miners. It displays the
	 * initial minerIDs in the network.
	 */
	public void printMinerSet()
	{
		for(int i = 0; i < network.length; i++)
		 {
			for(int j = 0; j < network[0].length; j++)
			{
				System.out.print(network[i][j].getMinerID()+"\t");
			}
			System.out.println();
		 }
	}
	
	/**
	 * This method calls the functions in the Miner class to simulate the Random Number 
	 * and Shuffling Miners protocol.
	 * @return networkShuffleIDs which has the new shuffled positions for the minerIDs.
	 */
	
	public int[][] runShuffleProtocol()
	{
		int i, j;
		
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length - getBench(); j++)
			{
				network[i][j].minerShuffleProtocolRun(network[i][j].getMinerID());
 			}
			
		}
			
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length - getBench(); j++)
			{
				network[i][j].updateShuffleProtoInfo(network[i][j].getMinerID(),network[i][j].getShuffleProto());
				broadcastProtoInfo(network[i][j]);
 			}
			
		}
		
    	for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length - getBench(); j++)
			{
	            System.out.println("The Miner ID :"+network[i][j].getMinerID());
				network[i][j].printRandProto();
		    }
			System.out.println();
		} 
		
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length - getBench(); j++)
			{
	            System.out.println("The Miner ID :"+network[i][j].getMinerID() + " verifying the miner's signatures and they are :" + network[i][j].minerVerifySignHash());
		    }
			System.out.println();
		}
		
		
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length - getBench(); j++)
			{
	            System.out.println("The Miner ID :"+network[i][j].getMinerID() + " verifying the hash and random numbers matches and they are: " + network[i][j].minerVerifyRandHash());
				network[i][j].minerVerifyRandHash();
		    }
			System.out.println();
		} 
	
		byte[] finalRand = null;
		
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length - getBench(); j++)
			{
	            System.out.println("The Miner ID :"+network[i][j].getMinerID() + " is calculating the final rand for shuffling miners");
				finalRand = network[i][j].minerConcatRandNum();
		    }
			System.out.println();
		}
		
		for(i = 0; i < network.length; i++)
		{
			for(j = getSecondary() + 1; j < getSecondary() + getBench() + 1; j++)
			{
			    
				network[i][j].benchMinerSetFinalRand(finalRand);
					
		    }
			
		}
		
		int[][] networkShuffleArray = null;
		
		System.out.print("All the miners generates the shuffled array position");
		for(i = 0; i < network.length; i++)
		{
			for(j = 0; j < network[0].length; j++)
			{
			   
			 networkShuffleArray = network[i][j].minerShufflePositions(getPrimary(),getSecondary(), getBench());
					
		    }
			System.out.println();
		}
		
		System.out.println("The contents of network shuffle array : ");
		
		for(i = 0; i < networkShuffleArray.length; i++)
		 {
			for(j = 0; j < networkShuffleArray[0].length; j++)
			{
				System.out.print(networkShuffleArray[i][j]+"\t");
			}
			System.out.println();
		 }
		
		int[][] networkShuffleIDs = networkShuffleArray;
		
		for(i = 0; i < networkShuffleArray.length; i++)
		 {
				for(j = 0; j < networkShuffleArray[0].length; j++)
				{
					int row_index = networkShuffleArray[i][j]/(1+getSecondary()+getBench());
					int column_index = networkShuffleArray[i][j]%(1+getSecondary()+getBench());
					networkShuffleIDs[i][j] = network[row_index][column_index].getMinerID();
				}
			
		 }
		
		
		return networkShuffleIDs;
	
  }
	
}
