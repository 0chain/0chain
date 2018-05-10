package net.zerochain.sharder.Block;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import net.zerochain.sharder.BlockDAO.IBlockDAO;

@Service("blockservice")
public class BlockServiceImpl implements IBlockService {

	@Autowired
	private IBlockDAO iBlockDAO;

	@Override
	public void saveBlock(BlockEntity blockEntity) {
		iBlockDAO.saveBlock(blockEntity);

	}

}
