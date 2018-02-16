package net_0chain;

import java.util.ArrayList;
import java.util.Stack;

public class MinerProtocol1 extends Miner {

	private Stack<Block> currentVerify;
	private Stack<Block> nextVerify;
	private ArrayList<Block> waitingConfirmation;
	private int maxCreationTime;
	
	public MinerProtocol1(int mt)
	{
		super();
		currentVerify = new Stack<Block>();
		nextVerify = new Stack<Block>();
		waitingConfirmation = new ArrayList<Block>();
		maxCreationTime = mt;
	}
	
	public MinerProtocol1(double d, int mt)
	{
		super(d);
		currentVerify = new Stack<Block>();
		nextVerify = new Stack<Block>();
		waitingConfirmation = new ArrayList<Block>();
		maxCreationTime = mt;
	}
	
	public MinerProtocol1(Block b, double d, int mt)
	{
		super(b,d);
		currentVerify = new Stack<Block>();
		nextVerify = new Stack<Block>();
		waitingConfirmation = new ArrayList<Block>();
		maxCreationTime = mt;
	}
	
	public void addBlockToStack(Block b)
	{
		nextVerify.add(b);
	}
	
	public Block getBlockToVerify()
	{
		return currentVerify.pop();
	}
	
	public Block createBlock()
	{
		long start = System.currentTimeMillis();
		Block b = new Block();
		int i = 0;
		while(i < getTransactionPool().size() && (System.currentTimeMillis() - start) < maxCreationTime/2)
		{
			Transaction temp = getTransactionPool().get(i);
			if(getClient().getLedger().validTransaction(temp) && temp.isSignatureValid() && temp.hashValid())
			{
				b.addTransaction(temp);
			}
			i++;
		}
		moveTransactionsToPending(b);
		b.setPreviousBlock(waitingConfirmation.get(waitingConfirmation.size() - 1).getCurrentHash());
		b.createCurrentHash();
		signBlock(b);
		//waitingConfirmation.add(b);
		while ((System.currentTimeMillis() - start) < maxCreationTime);
		return b;
	}
	
	public Block verifyBlock()
	{
		Block verify = getBlockToVerify();
		
		return verify;
	}
	
	
}
