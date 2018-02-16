package net_0chain;
import java.security.PublicKey;
import java.util.ArrayList;

public class Ledger {
	private ArrayList<Account> ledger;
	
	/**
	 * Creates a new ledger by creating an arraylist of accounts
	 */
	public Ledger()
	{
		ledger = new ArrayList<Account>();
	}
	
	/**
	 * This method adds a new account to the ledger
	 * @param a new account
	 */
	public void addAccount(Account a)
	{
		Account temp = new Account();
		temp.setCreds(a.getCreds());
		temp.setPublicKey(a.getPublicKey());
		ledger.add(temp);
	}
	
	/**
	 * This method returns the ledger
	 * @return ledger
	 */
	public ArrayList<Account> getAccounts()
	{
		return ledger;
	}
	
	/**
	 * This method adds an arraylist of accounts to the ledger
	 * as long as the new account doesn't already exist in the ledger
	 * @param newAccounts the list of new accounts
	 */
	public void addAccounts(ArrayList<Account> newAccounts)
	{
		for(int i = 0; i < newAccounts.size(); i++)
		{
			if(!ledger.contains(newAccounts.get(i)))
			{
				addAccount(newAccounts.get(i));
			}
		}
	}
	
	/**
	 * This method updates the ledger from a list of transactions
	 * @param transactions the list use to update the ledger
	 */
	public void updateAll(ArrayList<Transaction> transactions)
	{
		int size = transactions.size();
		for(int i = 0; i < size; i++)
		{
			update(transactions.get(i));
		}
	}
	
	/**
	 * This method updates the ledger from a single transaction
	 * @param t the transaction used to update the ledger
	 */
	public void update(Transaction t)
	{
		if(validTransaction(t))
		{
			int indexOfT = indexOfAccount(t.getTo());
			int indexOfF = indexOfAccount(t.getFrom());
			double tCreds = ledger.get(indexOfT).getCreds();
			double fCreds = ledger.get(indexOfF).getCreds();
			ledger.get(indexOfT).setCreds(tCreds + t.getCreds());
			ledger.get(indexOfF).setCreds(fCreds - t.getCreds());
		}
	}
	
	/**
	 * This method returns true or false if the transaction is valid 
	 * according to the ledger. If both the to and from account exist
	 * and there are sufficient funds in the from account for the
	 * transfer
	 * @param t the transaction
	 * @return true if the transaction is valid; false otherwise
	 */
	public boolean validTransaction(Transaction t)
	{
		boolean valid = false;
		int indexOfT = indexOfAccount(t.getTo());
		int indexOfF = indexOfAccount(t.getFrom());
		if(indexOfT >= 0 && indexOfF >=0 && t.getCreds() <= ledger.get(indexOfF).getCreds())
		{
			valid = true;
		}
		return valid;
	}
	
	/**
	 * This method returns the account from the ledger
	 * identified by the public key
	 * @param pk the public key
	 * @return account associated with the public key
	 */
	public Account getAccount(PublicKey pk)
	{
		return ledger.get(indexOfAccount(pk));
	}
	
	/**
	 * This method returns the index of the account
	 * associated with the public key.
	 * @param pk the public key
	 * @return index of the account in the ledger
	 */
	public int indexOfAccount(PublicKey pk)
	{
		int size = ledger.size();
		int index = -1;
		for(int i = 0; i < size && index < 0; i++)
		{
			if(ledger.get(i).getPublicKey().equals(pk))
			{
				index = i;
			}
		}
		return index;
	}
	
	/**
	 * This method prints all the accounts in the ledger.
	 * For each account the public key and the credits in the
	 * account are printed
	 */
	public void printLedger()
	{
		System.out.println("Accounts in Ledger:");
		for(int i = 0; i < ledger.size();i++)
		{
			System.out.println("Account: "+i);
			System.out.println("\tCreds: "+ledger.get(i).getCreds());
		}
	}
}
