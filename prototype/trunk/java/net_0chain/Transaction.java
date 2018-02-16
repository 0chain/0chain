package net_0chain;
import java.io.IOException;
import java.security.InvalidKeyException;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.PrivateKey;
import java.security.PublicKey;
import java.security.Signature;
import java.security.SignatureException;
import java.security.SignedObject;
import java.util.Arrays;

public class Transaction {
	private PublicKey to, from;
	private double creds;
	private SignedObject sign;
	private byte[] hash;
	
	/**
	 * Creates an uninitialized transaction with two public keys, one from and one to, 
	 * a balance, a signature, and a hash.
	 */
	public Transaction()
	{
		to = null;
		from = null;
		creds = 0.0;
		sign = null;
		hash = null;
	}
	
	/**
	 * Creates a transaction with two public keys, f from and t to, 
	 * a balance c, an uninitialized signature, and an uninitialized hash.
	 * @param t
	 * @param f
	 * @param c
	 */
	public Transaction(PublicKey t, PublicKey f, double c)
	{
		to = t;
		from = f;
		creds = c;
		sign = null;
		hash = null;
	}
	
	/**
	 * This method returns the public key the transaction is going to
	 * @return the public key 
	 */
	public PublicKey getTo() {
		return to;
	}
	
	/**
	 * This method sets the public key the transaction is going to
	 * @param to the public key
	 */
	public void setTo(PublicKey to) {
		this.to = to;
	}
	
	/**
	 * This method returns the public key of the client who created the transaction
	 * @return the public key
	 */
	public PublicKey getFrom() {
		return from;
	}
	
	/**
	 * This method sets the public key of the client who created the transction
	 * @param from the public key 
	 */
	public void setFrom(PublicKey from) {
		this.from = from;
	}
	
	/**
	 * This method returns the balance of the transaction
	 * @return balance of transaction
	 */
	public double getCreds() {
		return creds;
	}
	
	/**
	 * This method sets the balance of the transaction
	 * @param creds the balance of the transaction
	 */
	public void setCreds(double creds) {
		this.creds = creds;
	}
	
	/**
	 * This method returns the signed object of the transaction
	 * @return signed object of the hash of the transaction
	 */
	public SignedObject getSign() {
		return sign;
	}

	/**
	 * This method uses the private key of the client who created the transaction to signed the hash of the transaction
	 * @param pk the private key of the client who created the transaction
	 * @return true if the transaction was successfully signed; false otherwise
	 */
	public boolean setSign(PrivateKey pk) {
		boolean signed = false;
		try {
			this.sign = new SignedObject(this.getHash(),pk, Signature.getInstance("SHA256withRSA"));
			signed = true;
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
		return signed;
	}
	
	/**
	 * This method determines if the signature of the transaction is correct
	 * @return true if the signature of the transaction is correct
	 */
	public boolean isSignatureValid()
	{
		boolean valid = false;
		try {
			byte[] testHash = (byte[]) sign.getObject();
			if(sign.verify(from, Signature.getInstance("SHA256withRSA")) && Arrays.equals(this.getHash(), testHash))
			{
				valid = true;
			}
		} catch (ClassNotFoundException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		} catch (IOException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();	
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
		
		return valid;
	}
	
	/**
	 * This method compares the hashes of two transactions to determine if they are the same transaction
	 * @param t the transaction being compared
	 * @return true if the transactions are the same; false otherwise
	 */
	public boolean sameTransaction(Transaction t)
	{
		return Arrays.equals(hash, t.getHash());
	}
	
	/**
	 * This method returns the hash of the transaction
	 * @return hash of the transaction
	 */
	public byte[] getHash()
	{
		return hash;
	}
	
	/**
	 * This method creates the hash of a transaction by using 
	 * SHA256 to hash the public key to, the balance, and the 
	 * public key from.
	 */
	public void hashTransaction()
	{
		MessageDigest digest;
		try {
			digest = MessageDigest.getInstance("SHA-256");
			digest.update(to.getEncoded());
			digest.update(new Double(creds).byteValue());
			digest.update(from.getEncoded());
			hash = digest.digest();
		} catch (NoSuchAlgorithmException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}
	}
	
	/**
	 * This method determines if the hash in the transaction is valid
	 * @return true if the hash in the transaction is valid
	 */
	public boolean hashValid()
	{
		MessageDigest digest;
		boolean valid = false;
		try {
			byte[] hashTemp;
			digest = MessageDigest.getInstance("SHA-256");
			digest.update(to.getEncoded());
			digest.update(new Double(creds).byteValue());
			digest.update(from.getEncoded());
			hashTemp = digest.digest();
			valid = Arrays.equals(hashTemp, getHash());
		} catch (NoSuchAlgorithmException e) {
			// TODO Auto-generated catch block
			e.printStackTrace();
		}
		return valid;
	}
	
}
