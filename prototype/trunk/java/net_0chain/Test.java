package net_0chain;

import java.io.IOException;

public class Test {
	public static void main(String args[]) throws ClassNotFoundException, IOException
	{
		int i;
		int numBlocks = 3;
		MinerNetworkProtocols mn = new MinerNetworkProtocols(3, 2, 0, 0.50, new int[]{1,3,6}, new int[]{});
		//MinerNetwork mn = new MinerNetwork(3, 2, 2, .50);

		for(i = 0; i < 2000; i++)
		{
			mn.acceptTransaction(mn.createTransaction(0, 1, 0.0001));
		}
		
		long start = System.currentTimeMillis();
		for(i = 0; i < numBlocks; i++)
		{
			mn.singleRoundProtocol0(i);
			//mn.getChain().printCurrentBlock();
		}
		long finish = System.currentTimeMillis();
		mn.getChain().printHashes();
		mn.getMiner(0).getClient().getLedger().printLedger();
		System.out.println("Time to create "+(mn.getChain().getLength()-1)+": " + (finish-start)+" milliseconds");
		if(mn.getChain().getLength() > 1)
		{
			System.out.println("Block rate: 1 block per "+ ((finish-start)/(mn.getChain().getLength()-1))+" milliseconds");
		}
		mn.printMinerBlocks();
	}
}
