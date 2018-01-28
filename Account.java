import java.security.PublicKey;

public class Account {
	private PublicKey publicKey;
	private double creds;
	
	/**
	 * Creates an Account without a public key and a balance of 0.0
	 */
	public Account()
	{
		publicKey = null;
		creds = 0.0;
	}
	
	/**
	 * Creates an Account with a public key and a balance of 0.0
	 * @param pk public key used for this account
	 */
	public Account(PublicKey pk)
	{
		publicKey = pk;
		creds = 0.0;
	}
	
	/**
	 * Creates an Account without a public key and a balance of c
	 * @param c the amount used to initialize the balance
	 */
	public Account(double c)
	{
		publicKey = null;
		creds = c;
	}
	
	/**
	 * Creates an account with a public key and a balance of c
	 * @param pk the public key used for this account
	 * @param c	the amount used to initialize balance
	 */
	public Account(PublicKey pk, double c)
	{
		publicKey = pk;
		creds = c;
	}
	
	/**
	 * Returns the public key for the account
	 * @return public key
	 */
	public PublicKey getPublicKey() {
		return publicKey;
	}
	
	/**
	 * Sets the public key to a new public key
	 * @param publicKey the new public key used for the account
	 */
	public void setPublicKey(PublicKey publicKey) {
		this.publicKey = publicKey;
	}
	
	/**
	 * Returns the balance for the account
	 * @return balance
	 */
	public double getCreds() {
		return creds;
	}
	
	/**
	 * Sets the accounts balance to a new balance
	 * @param creds the new balance for the account
	 */
	public void setCreds(double creds) {
		this.creds = creds;
	}
	
	
}
