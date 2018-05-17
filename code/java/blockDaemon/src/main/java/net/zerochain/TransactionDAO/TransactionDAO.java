package net.zerochain.TransactionDAO;

import javax.persistence.EntityManager;
import javax.persistence.PersistenceContext;

import javax.transaction.Transactional;
import org.springframework.stereotype.Repository;
import java.util.List;
import org.hibernate.Criteria;
import org.hibernate.Session;
import org.hibernate.SessionFactory;
import org.hibernate.cfg.Configuration;
import org.hibernate.criterion.Order;
import org.hibernate.criterion.Restrictions;
import net.zerochain.Transaction.TransactionEntity;
import java.util.List;

@Transactional
@Repository
public class TransactionDAO implements ITransactionDAO {
	
	@PersistenceContext
	private EntityManager entityManager;

	@Override
	public List<TransactionEntity> getTwoHundredTransactions()
	{
		Criteria crit = entityManager.unwrap(Session.class).createCriteria(TransactionEntity.class);
		crit.setMaxResults(200);
		crit.add(Restrictions.eq("status","free"));
		crit.addOrder(Order.desc("timestamp"));
		return crit.list();
	}

	@Override
	public void updateTransactionsToPending(List<TransactionEntity> transactions)
	{
		for(TransactionEntity t:transactions)
		{
			TransactionEntity temp = (TransactionEntity)entityManager.find(TransactionEntity.class,t.getHash_msg());
			temp.setStatus("pending");
		}
	}
}
