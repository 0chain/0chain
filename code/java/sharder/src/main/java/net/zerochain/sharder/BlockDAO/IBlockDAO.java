package net.zerochain.sharder.BlockDAO;

import net.zerochain.sharder.Block.BlockEntity;

public interface IBlockDAO {
	void saveBlock(BlockEntity blockEntity);

}
