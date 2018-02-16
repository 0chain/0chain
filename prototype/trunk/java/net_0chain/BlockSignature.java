package net_0chain;

import java.security.PublicKey;
import java.security.SignedObject;


public class BlockSignature{
	private PublicKey publicKey;
	private SignedObject sign;
	
	/**
	 * This creates a new block signature
	 */
	public BlockSignature()
	{
		publicKey = null;
		sign = null;
	}
	
	/**
	 * This method returns the public key of the block signature
	 * @return public key
	 */
	public PublicKey getPublicKey() {
		return publicKey;
	}
	
	/**
	 * This method sets the public key of the block signature for identifying 
	 * of who signed the block
	 * @param publicKey the public key
	 */
	public void setPublicKey(PublicKey publicKey) {
		this.publicKey = publicKey;
	}
	
	/**
	 * This method returns the Signed Object (the block's hash) from the block signature
	 * @return signed object of the block's hash
	 */
	public SignedObject getSign() {
		return sign;
	}
	
	/**
	 * This method sets the signed object to the signed object being provided
	 * @param sign the signed object of the block's hash
	 */
	public void setSign(SignedObject sign) {
		this.sign = sign;
	}
	
}
