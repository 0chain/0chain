import java.io.IOException;

public class Test {
	public static void main(String args[]) throws ClassNotFoundException, IOException
	{
		int i;
		int numBlocks = 10;
		
		//MinerNetwork mn = new MinerNetwork(3, 2, 0.5, new int[]{1,2,3,6, 4});
		MinerNetwork mn = new MinerNetwork(3, 2, 0.5);
		
		for(i = 0; i < 8300; i++)
		{
			Transaction temp = mn.createTransaction(0, 10, 0.0001);
			mn.acceptTransaction(temp);
		}
		
		long start = System.currentTimeMillis();
		for(i = 0; i < numBlocks; i++)
		{
			mn.singleRound(0);
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
