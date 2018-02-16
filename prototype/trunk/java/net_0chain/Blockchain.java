package net_0chain;
import java.util.Arrays;
import java.util.LinkedList;

public class Blockchain {
	private LinkedList<Block> chain;
	
	/**
	 * Creates a new linked list of blocks to create 
	 * a new blockchain without a genesis block
	 */
	public Blockchain()
	{
		chain = new LinkedList<Block>();
	}
	
	/**
	 * Creates a new linked list of blocks to create
	 * a new blockchain and adds the genesis block to it
	 * @param b
	 */
	public Blockchain(Block b)
	{
		chain = new LinkedList<Block>();
		chain.add(b);
	}
	
	/**
	 * This method returns the length of the blockchain
	 * @return length of chain
	 */
	public int getLength()
	{
		return chain.size();
	}
	
	/**
	 * This method returns the genesis block from the blockchain
	 * @return genesis block
	 */
	public Block getGenesis()
	{
		return chain.getFirst();
	}
	
	/**
	 * This method adds a new block to the chain provided that the new block
	 * has the previous hash set to the current block in the chain
	 * @param b the new block to be added to the chain
	 */
	public void addBlock(Block b)
	{
		if(chain.size() == 0 || Arrays.equals(chain.getLast().getCurrentHash(),b.getPreviousBlock()))
		{
			chain.addLast(b);
		}
	}
	
	/**
	 * This method returns the current block of the blockchain
	 * @return current block
	 */
	public Block getCurrentBlock()
	{
		return chain.getLast();
	}
	
	/**
	 * This method returns the current hash of the current block for the blockchain
	 * @return current hash
	 */
	public byte[] getCurrentHash()
	{
		return getCurrentBlock().getCurrentHash();
	}
	
	/**
	 * This method prints out the blocks of the blockchain in order from first to last
	 * printing the block number, each hash of the block as well as the previous hash, 
	 * number of signed transactions, number of signatures from miners, as well as if 
	 * all the signatures are valid
	 */
	public void printHashes()
	{
		int i, j;
		System.out.println("The number of blocks: "+chain.size());
		for(i = 0; i < chain.size(); i++)
		{
			System.out.println("Block "+i);
			System.out.print("\tPrevious hash: ");
			for(j = 0; j < chain.get(i).getPreviousBlock().length; j++)
			{
				System.out.print(chain.get(i).getPreviousBlock()[j]);
			}
			System.out.println();
			System.out.println("\tNumber of signed transactions: "+chain.get(i).getTransactions().size());
			System.out.println("\tNumber of signatures: "+chain.get(i).getSignatures().size());
			System.out.println("\tAll signatures valid: "+chain.get(i).verifySignatures());
			
			System.out.print("\tCurrent hash: ");
			for(j = 0; j < chain.get(i).getCurrentHash().length; j++)
			{
				System.out.print(chain.get(i).getCurrentHash()[j]);
			}
			System.out.println("\n");

		}
	}
	
	/**
	 * This method prints the current block's previous hash,
	 * the number of signed transactions, number of signatures,
	 * if all the signature are valid, and the current block's hash
	 */
	public void printCurrentBlock()
	{
		int j;
		System.out.println("Block "+chain.size());
		System.out.print("\tPrevious hash: ");
		for(j = 0; j < getCurrentBlock().getPreviousBlock().length; j++)
		{
			System.out.print(getCurrentBlock().getPreviousBlock()[j]);
		}
		System.out.println();
		System.out.println("\tNumber of signed transactions: "+getCurrentBlock().getTransactions().size());
		System.out.println("\tNumber of signatures: "+getCurrentBlock().getSignatures().size());
		System.out.println("\tAll signatures valid: "+getCurrentBlock().verifySignatures());
		
		System.out.print("\tCurrent hash: ");
		for(j = 0; j < getCurrentBlock().getCurrentHash().length; j++)
		{
			System.out.print(getCurrentBlock().getCurrentHash()[j]);
		}
		System.out.println("\n");
	}
	
}
