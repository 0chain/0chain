import java.io.IOException;
import java.security.InvalidKeyException;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.PrivateKey;
import java.security.PublicKey;
import java.security.Signature;
import java.security.SignatureException;
import java.security.SignedObject;
import java.util.ArrayList;
import java.util.Arrays;
import java.io.Serializable;

public class Block implements Serializable {
	private ArrayList<Transaction> transactions;
	private byte[] currentHash;
	private byte[] previousBlock;
	private ArrayList<BlockSignature> signs;
	
	/**
	 * Creates an uninitialized block
	 */
	public Block()
	{
		transactions = new ArrayList<Transaction>();
		currentHash = null;
		previousBlock = null;
		signs = new ArrayList<BlockSignature>();
	}
	
	/**
	 * Creates a block with the previous block's hash
	 * @param prevBlock the hash from the previous block
	 */
	public Block(byte[] prevBlock)
	{
		transactions = new ArrayList<Transaction>();
		currentHash = null;
		previousBlock = prevBlock;
		signs = new ArrayList<BlockSignature>();
	}
	
	/**
	 * This method sets the previous block hash
	 * @param prevBlock The hash of the previous block
	 */
	public void setPreviousBlock(byte[] prevBlock)
	{
		previousBlock = prevBlock;
	}
	
	/**
	 * This method is used to return the previous hash
	 * of the current block.
	 * @return previous block's hash
	 */
	public byte[] getPreviousBlock()
	{
		return previousBlock;
	}
	
	/**
	 * Adds a transaction to the transaction list
	 * @param t the transaction added to the list
	 */
	public void addTransaction(Transaction t)
	{
		transactions.add(t);
	}
	
	/**
	 * returns the list of transactions in the block
	 * @return list of transactions 
	 */
	public ArrayList<Transaction> getTransactions()
	{
		return transactions;
	}
	
	/**
	 * This method is used to create the current hash of the block.
	 * The current hash is created by using SHA256 to hash the previous 
	 * block's hash along with the hashes of all the transactions in the
	 * block.
	 */
	public void createCurrentHash()
	{
		if(currentHash == null)
		{
			try {
				MessageDigest digest = MessageDigest.getInstance("SHA-256");
				digest.update(previousBlock);
				for(int i = 0; i < transactions.size(); i++)
				{
					digest.update(transactions.get(i).getHash());
				}
				currentHash = digest.digest();
				
			} catch (NoSuchAlgorithmException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			}
		}
		
	}
	
	/**
	 * This method returns the block's hash
	 * @return block's hash
	 */
	public byte[] getCurrentHash() {
		return currentHash;
	}
	
	/**
	 * This method will be used to try to make fake blocks
	 * by bad miners. Instead of creating the hash the way it
	 * should the miner can set it to anything they'd like.
	 * @param currentHash the bad hash to set the block to.
	 */
	public void setCurrentHash(byte[] currentHash) {
		this.currentHash = currentHash;
	}
	
	/**
	 * This method adds a BlockSignature of a miner who either
	 * verifies the block or the miner who generated the block.
	 * @param privateK the private key of the miner signing the block
	 * @param publicK the public key added to the BlockSignature as identification
	 */
	public void addSign(PrivateKey privateK, PublicKey publicK)
	{
		BlockSignature temp = new BlockSignature();
		temp.setPublicKey(publicK);
		try {
			SignedObject so = new SignedObject(this.currentHash, privateK, Signature.getInstance("SHA256withRSA"));
			temp.setSign(so);
			this.signs.add(temp);
		} catch (InvalidKeyException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} catch (SignatureException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} catch (NoSuchAlgorithmException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} catch (IOException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}
	}
	
	/**
	 * When two blocks have the same hash but different signatures
	 * the signatures of one block can be added to the other as long
	 * as they aren't already on the block.
	 * @param b the block generated or verified by another miner
	 */
	public void addSameBlockSign(Block b)
	{
		boolean same = this.sameBlock(b);
		ArrayList<BlockSignature> check = b.getSignatures();
		ArrayList<BlockSignature> toAdd = new ArrayList<BlockSignature>();
		for(int i = 0; i < check.size() && same; i++)
		{
			SignedObject so = check.get(i).getSign();
			try {
				if(so.verify(check.get(i).getPublicKey(), Signature.getInstance("SHA256withRSA")))
				try {
					byte[] c = (byte []) so.getObject();
					boolean sameBlock = Arrays.equals(this.getCurrentHash(), c);
					if(sameBlock && Arrays.equals(this.getPreviousBlock(), b.getPreviousBlock()) && !signs.contains(check.get(i)))
					{
						toAdd.add(check.get(i));
					}
				} catch (ClassNotFoundException e) {
					same = false;
					e.printStackTrace();
				} catch (IOException e) {
					same = false;
					e.printStackTrace();
				}
			} catch (InvalidKeyException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			} catch (SignatureException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			} catch (NoSuchAlgorithmException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			}
		}
		if(same)
		{
			signs.addAll(toAdd);
		}
	}
	
	/**
	 * This method returns the list of signatures of
	 * the miners who verified or generated the block
	 * @return list of signatures
	 */
	public ArrayList<BlockSignature> getSignatures()
	{
		return signs;
	}
	
	/**
	 * This method compares the hashes of two blocks
	 * to determine if they are the same block
	 * @param b the block to compare
	 * @return true if the two blocks are the same; false otherwise
	 */
	public boolean sameBlock(Block b)
	{
		return Arrays.equals(this.getCurrentHash(), b.getCurrentHash());
	}
	
	/**
	 * This method returns a clone of a block to simulate broadcast the
	 * block to different miners.
	 * @return a clone of the block
	 */
	public Block cloneBlock()
	{
		Block temp = new Block(previousBlock);
		temp.currentHash = new byte[currentHash.length];
		temp.currentHash = currentHash;
		temp.signs = new ArrayList<BlockSignature>();
		temp.signs.addAll(this.signs);
		temp.transactions = new ArrayList<Transaction>();
		temp.transactions.addAll(this.transactions);
		return temp;
	}

	
	/**
	 * This method adds the signatures of a list to the block
	 * @param s the list of signatures to add to the block
	 */
	public void addSigns(ArrayList<BlockSignature> s)
	{
		signs.addAll(s);
	}
	
	/**
	 * Prints a block's hash, number of transactions,
	 * and every transaction's public key from, to, and 
	 * the amount transfered.
	 */
	public void printBlock()
	{
		System.out.println("Block hash: "+new String(currentHash)+"\nNumber of transactions: "+transactions.size());
		for(int i = 0; i < transactions.size(); i++)
		{
			System.out.println("Transaction "+i+": ");
			System.out.println("\tFrom: "+new String(transactions.get(i).getFrom().getEncoded()));
			System.out.println("\tTo: "+new String(transactions.get(i).getTo().getEncoded()));
			System.out.println("\tCreds: "+transactions.get(i).getCreds());
		}
		
	}
	
	/**
	 * Verifies the list of signatures on the block are valid
	 * @return true if all the signatures are valid; false otherwise
	 */
	public boolean verifySignatures()
	{
		boolean verified = true;
		for(int i = 0; i < signs.size() && verified; i++)
		{
			SignedObject so = signs.get(i).getSign();
			try {
				if(so.verify(signs.get(i).getPublicKey(), Signature.getInstance("SHA256withRSA")))
				{
					try {
						byte[] testHash = (byte[]) so.getObject();
						if(!Arrays.equals(testHash, this.getCurrentHash()))
						{
							verified = false;
						}
					} catch (ClassNotFoundException e) {
						// TODO Auto-generated catch block
						verified = false;
						e.printStackTrace();
					} catch (IOException e) {
						// TODO Auto-generated catch block
						verified = false;
						e.printStackTrace();
					}
				}
				else
				{
					verified = false;
				}
			} catch (InvalidKeyException e1) {
				// TODO Auto-generated catch block
				verified = false;
				e1.printStackTrace();
			} catch (SignatureException e1) {
				// TODO Auto-generated catch block
				verified = false;
				e1.printStackTrace();
			} catch (NoSuchAlgorithmException e1) {
				// TODO Auto-generated catch block
				verified = false;
				e1.printStackTrace();
			}
			
		}
		return verified;
	}
}
