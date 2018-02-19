package net_0chain;

import java.io.IOException;

public class Test {
	public static void main(String args[]) throws ClassNotFoundException, IOException
	{
		int i;
		int numBlocks = 1;
		MinerNetworkProtocols mn = new MinerNetworkProtocols(3, 2, 2, 0.50, new int[]{}, new int[]{6});
		//MinerNetwork mn = new MinerNetwork(3, 2, 2, .50);

		for(i = 0; i < 1000; i++)
		{
			mn.acceptTransaction(mn.createTransaction(0, 10, 0.0001));
		}
		
		long start = System.currentTimeMillis();
		for(i = 0; i < numBlocks; i++)
		{
			mn.singleRoundProtocol2(6);
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
		
		System.out.println();
		System.out.println("The miners generate the cryptographic-quality random number");
			
		mn.runRandProtocol();
			
	}
}
