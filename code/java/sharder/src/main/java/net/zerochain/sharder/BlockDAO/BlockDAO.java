package net.zerochain.sharder.BlockDAO;

import javax.persistence.EntityManager;
import javax.persistence.PersistenceContext;
import javax.transaction.Transactional;

import org.springframework.stereotype.Repository;

import net.zerochain.sharder.Block.BlockEntity;

@Transactional
@Repository
public class BlockDAO implements IBlockDAO{
	
	@PersistenceContext
	private EntityManager entityManager;
	
	@Override
	public void saveBlock(BlockEntity blockEntity) {
		entityManager.persist(blockEntity);
	}

}
