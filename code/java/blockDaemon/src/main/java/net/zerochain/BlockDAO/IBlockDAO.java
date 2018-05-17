package net.zerochain.BlockDAO;

import net.zerochain.Block.BlockEntity;
import net.zerochain.Block.BlockTransactionEntity;

public interface IBlockDAO {
	void saveBlock(BlockEntity blockEntity);
	String getLastFinalizedBlockHash();
	int getLastFinalizedRound();
	boolean isBlockEmpty();
	void saveBlockTransaction(BlockTransactionEntity blockTransactionEntity);
}