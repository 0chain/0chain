package net.zerochain.Block;

import net.zerochain.Block.BlockEntity;
import net.zerochain.Block.BlockTransactionEntity;
import java.util.List;
import net.zerochain.Transaction.TransactionEntity;

public interface IBlockService
{
	BlockEntity generateBlock();
	void sendBlock(BlockEntity blockEntity);
	void saveTransactions(BlockEntity blockEntity, List<String> hash_msg);
	BlockEntity createBlockEntity(List<TransactionEntity> transactions);
	String createBlockHash(List<TransactionEntity> transactions, String prev_block_hash);
	void setMiner();

}