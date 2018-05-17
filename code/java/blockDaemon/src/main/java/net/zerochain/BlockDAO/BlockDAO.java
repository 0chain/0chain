package net.zerochain.BlockDAO;

import javax.persistence.EntityManager;
import javax.persistence.PersistenceContext;

import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;
import java.util.List;
import org.hibernate.Criteria;
import org.hibernate.Session;
import org.hibernate.SessionFactory;
import org.hibernate.criterion.Order;
import org.hibernate.cfg.Configuration;
import org.hibernate.criterion.Restrictions;
import net.zerochain.Block.BlockEntity;
import net.zerochain.Block.BlockTransactionEntity;

@Transactional
@Repository
public class BlockDAO implements IBlockDAO {
	
	@PersistenceContext 
	private EntityManager entityManager;
	
	@Override
	public void saveBlock(BlockEntity blockEntity) {
		entityManager.persist(blockEntity);
	}

	@Override
	public void saveBlockTransaction(BlockTransactionEntity blockTransactionEntity)
	{
		entityManager.persist(blockTransactionEntity);
	}

	@Override
	public String getLastFinalizedBlockHash()
	{
		Criteria crit = entityManager.unwrap(Session.class).createCriteria(BlockEntity.class);
		crit.addOrder(Order.desc("round"));
		List<BlockEntity> blocksOrdered = crit.list();
		return blocksOrdered.get(0).getBlock_hash();
	}

	@Override
	public int getLastFinalizedRound()
	{
		Criteria crit = entityManager.unwrap(Session.class).createCriteria(BlockEntity.class);
		crit.addOrder(Order.desc("round"));
		List<BlockEntity> blocksOrdered = crit.list();
		return blocksOrdered.get(0).getRound();
	}

	@Override
	public boolean isBlockEmpty()
	{
		Criteria crit = entityManager.unwrap(Session.class).createCriteria(BlockEntity.class);
		return crit.list().size() == 0;
	}
}