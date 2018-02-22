package net_0chain;


import java.io.IOException;
import java.nio.ByteBuffer;
import java.security.InvalidKeyException;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.PrivateKey;
import java.security.PublicKey;
import java.security.SecureRandom;
import java.security.Signature;
import java.security.SignatureException;
import java.security.SignedObject;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map.Entry;

public class ShuffleProtocol{

	private BlockSignature hashSigns;
	private HashMap<Integer,ShuffleProtocol> randomHash;
	private Long randNum;
	private byte[] randHash;
	private byte[] finalRandHash;
	private int[] shuffleArray;
	
	/**
	 * Creates a RandNumprotocol object with a Block Signature, hash table to store the Miner ID info,
	 * random number that the miner generates, the hash of the random number, the final rand 
	 * which is used for shuffling miners and the array which contains the shuffled miner positions 
	 * for each miner for the next cycle.
	 */
	public ShuffleProtocol() {
		
		hashSigns = new BlockSignature();
		randomHash = new HashMap<Integer,ShuffleProtocol>();
		randNum = null;
		randHash = null;
		finalRandHash = null;
		shuffleArray = null;
	}

	/**
	 * This method returns the cryptographic-quality random number which the miner has
	 * @return random number which the miner has
	 */
	public Long getRandomNumber()
	{
		return randNum;
	}
	
	/**
	 * This method returns the miners signature on the hashed random number
	 * @return the miner signature on the hashed random number
	 */
	public BlockSignature getHashSignature()
	{
		return hashSigns;
	}
	
	
	/**
	 * This method returns the hash of the random number which the miner generates
	 * @return random number's hash
	 */
	public byte[] getRandHash() {
		return randHash;
	}
	

	/**
	 * This method adds a BlockSignature to the hashed random number which the miner has
	 * @param privateK the private key of the miner signing the random number
	 * @param publicK the public key added to the BlockSignature as identification
	 */
	public void addSignHashedRand(PrivateKey privateK, PublicKey publicK)
	{
		BlockSignature temp = new BlockSignature();
		temp.setPublicKey(publicK);
		try {
			SignedObject so = new SignedObject(randHash, privateK, Signature.getInstance("SHA256withRSA"));
			temp.setSign(so);
			this.hashSigns = temp;
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
	 * This method is used to update the hash table with the miner info. The table includes
	 * The Miner ID and its corresponding RandNumProtocol for lookup. The hash is created 
	 * by using SHA256.
	 * @param minerID the ID of the miner
	 */

	public void createRandHash(int minerID)
	{	
		try {
			    randomHash.put(minerID, this);
			    SecureRandom ran = new SecureRandom();
			    randNum = ran.nextLong();
				MessageDigest digest = MessageDigest.getInstance("SHA-256");
				digest.update(randNum.toString().getBytes());
				randHash = digest.digest();
				
			} catch (NoSuchAlgorithmException e) {
				// TODO Auto-generated catch block
				e.printStackTrace();
			}	
	}

	/**
	 * This method displays the hash in hex format.
	 * @return random number's hash in hex format
	 */
    public String displayRandHash(byte[] randHash) 
    {
    		StringBuffer sb = new StringBuffer();
            for (int i = 0; i < randHash.length; i++) 
            {
               sb.append(Integer.toString((randHash[i] & 0xff) + 0x100, 16).substring(1));
            }
    
            return sb.toString();
    
    }
	
    /**
	 * This method updates the miner info table.
	 * @param minerId the ID of the miner
	 * @param randProto the RandNumProtocol object of each miner
	 */
	
	public void updateTable(Integer minerID, ShuffleProtocol randProto)
	{
		randomHash.put(minerID, randProto);
	}
	
	/**
	 * This method prints the miner info table.
	 */
	public void printSignHashRandNum()
	{
		
		for (Entry<Integer, ShuffleProtocol> entry : randomHash.entrySet()) 
		{
		    Integer key = entry.getKey();
		    ShuffleProtocol valueobj = entry.getValue();    
		    System.out.println ("Miner id : " + key + " has the Random Num :  " + valueobj.getRandomNumber() + " and hash " 
		    + displayRandHash(valueobj.getRandHash())); 
		    //+ "\n" + " and the signed hash :" + valueobj.getHashSignature());
		    System.out.println();
		}
		
	} 
	
	/**
	 * Verifies the signature of the hashed random number that the miner generates is valid
	 * @return true if the signature is valid; false otherwise
	 */
	public boolean verifySignatures()
	{
		boolean verified = true;
		for (Entry<Integer, ShuffleProtocol> entry : randomHash.entrySet()) {
			
			Integer key = entry.getKey();
		    ShuffleProtocol valueobj = entry.getValue();  
			SignedObject so = valueobj.getHashSignature().getSign();
			try {
				if(so.verify(valueobj.getHashSignature().getPublicKey(), Signature.getInstance("SHA256withRSA")))
				{
					try {
						byte[] testHash = (byte[]) so.getObject();
						if(!Arrays.equals(testHash, valueobj.getRandHash()))
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
	
	/**
	 * Verifies the hashes of the random number sent by the miner corresponds to that
	 * miner's random number.
	 * @return true if the hashes are valid; false otherwise
	 */
	
	public boolean verifyRandHashes()
	{	
		boolean verified = true;
		for (Entry<Integer, ShuffleProtocol> entry : randomHash.entrySet()) 
		{
			
			Integer key = entry.getKey();
		    ShuffleProtocol valueobj = entry.getValue();  
		    try {
		    	
		    	  MessageDigest digest = MessageDigest.getInstance("SHA-256");
				  digest.update(String.valueOf(valueobj.getRandomNumber()).getBytes());
				  
				  if(!Arrays.equals(digest.digest(),valueobj.getRandHash()))
				  {
					  verified = false;
				  }
				  
				
			    } catch (NoSuchAlgorithmException e) {
				    // TODO Auto-generated catch block
				       e.printStackTrace();
			   }	
		
		   }
        
               return verified;
    
	 }

	/**
	 * This method calculates the final "rand" which is used for the shuffling miner protocol. Here the
	 * random numbers are concatenated in the order of miner IDs and hashed to get the final "rand".
	 * @return the final hashed random number which is used for shuffling miners.
	 */
	
	public byte[] concatRandNum()
	{
		
		long sumRand = 0L;
		for (Entry<Integer, ShuffleProtocol> entry : randomHash.entrySet()) 
		{
		    Integer key = entry.getKey();
		    ShuffleProtocol valueobj = entry.getValue();    
		    
		    sumRand = sumRand + valueobj.getRandomNumber();
		}
		
		try
		{
			MessageDigest digest = MessageDigest.getInstance("SHA-256");
			digest.update(String.valueOf(sumRand).getBytes());
			finalRandHash = digest.digest();
	    } 
		catch (NoSuchAlgorithmException e) {
		// TODO Auto-generated catch block
	    	e.printStackTrace();
	    }
		
		System.out.println("The final rand is : "+displayRandHash(finalRandHash));
		return finalRandHash;
	} 
	
	/**
	 * Sets the final rand number which is used for the shuffling protocol for the bench miners
	 */
	
	public void benchSetFinalRand(byte[] finalRand)
 	{
 		finalRandHash = finalRand;
 	}
	
	/**
	 * This method calculates the new shuffled positions for each miner for the next round. The array
	 * is initialized to -1 in the beginning. The array returns the new miner positions for each miner
	 * for the next round. The spillerArray is used to remove collision.
	 * We assign the first miner to position rand mod n, the second miner to position hash(rand) mod n,
     * the third miner to position hash(hash(rand)) mod n and so on.
	 * @param p the number of primary miners
	 * @param s the number of secondary miners
	 * @param b the number of bench miners
	 * @return the 2D array, networkShuffleArray which has the new shuffled positions.
	 */

	 public int[][] shufflePositions(int p, int s, int b) 
	 {
		 int i,j = 0;
		 int totalMiners = p*(1+s+b);
		 shuffleArray = new int[totalMiners];
		 List<Integer> spillerArray = new ArrayList<Integer>();
		 Arrays.fill(shuffleArray, -1);
		 ByteBuffer buffer = ByteBuffer.wrap(finalRandHash);
		 long current = buffer.getLong();
		 for(i=0;i<shuffleArray.length;i++)
		 {
			 	int newMinerPosition = (int) Math.abs((current % totalMiners));
			 	if(shuffleArray[newMinerPosition] == -1)
			 	{
			 		shuffleArray[newMinerPosition] = i;
			 	}  
			 	else
			 	{
			 		spillerArray.add(shuffleArray[newMinerPosition]);
			 		shuffleArray[newMinerPosition] = i;
			 	}
		    
			 	buffer = ByteBuffer.wrap(hashChainforShufflePositions(current));
			 	current = buffer.getLong();
		 }

		 for(i=0;i<shuffleArray.length;i++)
		 {
			 if(shuffleArray[i] == -1)
			 {
				 shuffleArray[i] = spillerArray.get(j);
				 j++;
			 }
		 }
		
		 int networkShuffleArray[][] = new int[p][1+s+b];
		 int k=0;
		 
		 for(i = 0; i < p; i++)
		 {
				for(j = 0; j < (1+s+b); j++)
				{
					networkShuffleArray[i][j] = shuffleArray[k];
					k++;
				}
		 }
		 
		 return networkShuffleArray;
		 
	}
	
	 	/**
	 	 * This method calculates the hash for finding the shuffled positions.
	 	 * @param current the current hash calculated for the shuffled position.
	 	 * @return calculates the hash on current
	 	 */
		 
	 	public byte[] hashChainforShufflePositions(long current) 
	 	{
	 		byte[] sha1hash = null;
	 		try {
	 			MessageDigest digest = MessageDigest.getInstance("SHA-256");
	 			digest.update(String.valueOf(current).getBytes());
	 			sha1hash = digest.digest();
	 			} catch (NoSuchAlgorithmException e) {
	 				e.printStackTrace();
	 			}
		
	 		return sha1hash;
	
	 	}
	 	
	 	
	
}



