package net.zerochain.component;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;
import org.springframework.boot.CommandLineRunner;
import net.zerochain.Block.BlockEntity;
import net.zerochain.Block.IBlockService;

@Component
public class BlockGenerationComponent implements CommandLineRunner
{
	@Autowired
	IBlockService iBlockService;

	@Override
	public void run(String... args) throws Exception
	{
		iBlockService.setMiner();
		long start = System.nanoTime();
		int i = 1;
		while(1==1)
		{
			BlockEntity blockEntity = iBlockService.generateBlock();
			iBlockService.sendBlock(blockEntity);
			System.out.print("\rBlocks generated: "+i);
			i++;
		}
		
	}
}