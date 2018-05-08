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
import org.hibernate.criterion.Restrictions;
import net.zerochain.Transaction.TransactionEntity;

@Transactional
@Repository
public class TransactionDAO implements ITransactionDAO {
	
	@PersistenceContext
	private EntityManager entityManager;
	
	@Override 
	public void saveTransaction(TransactionEntity transactionEntity) {
		entityManager.persist(transactionEntity);
	}

	@Override
	public boolean lookupTransaction(TransactionEntity transactionEntity)
	{
		Criteria crit = entityManager.unwrap(Session.class).createCriteria(TransactionEntity.class);
		crit.add(Restrictions.eq("hash_msg",transactionEntity.getHash_msg()));
		List<TransactionEntity> transactions = crit.list();
		return transactions.size() > 0;
	}
}
