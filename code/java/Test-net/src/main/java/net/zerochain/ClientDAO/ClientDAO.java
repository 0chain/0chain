package net.zerochain.ClientDAO;

import javax.persistence.EntityManager;
import javax.persistence.PersistenceContext;

import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;
import java.util.List;
import org.hibernate.Criteria;
import org.hibernate.Session;
import org.hibernate.SessionFactory;
import org.hibernate.cfg.Configuration;
import org.hibernate.criterion.Restrictions;
import net.zerochain.Client.ClientEntity;

@Transactional
@Repository
public class ClientDAO implements IClientDAO {
	
	@PersistenceContext 
	private EntityManager entityManager;
	
	@Override
	public void saveRegistration(ClientEntity clientEntity) {
		entityManager.persist(clientEntity);
	}

	@Override
	public boolean lookupClient(ClientEntity clientEntity)
	{
		Criteria crit = entityManager.unwrap(Session.class).createCriteria(ClientEntity.class);
		crit.add(Restrictions.eq("hash_key",clientEntity.getHash_key()));
		List<ClientEntity> clients = crit.list();
		return clients.size() > 0;
	}

	@Override
	public String getClientPublic_key(ClientEntity clientEntity)
	{
		String public_key = "";
		Criteria crit = entityManager.unwrap(Session.class).createCriteria(ClientEntity.class);
		crit.add(Restrictions.eq("hash_key",clientEntity.getHash_key()));
		List<ClientEntity> clients = crit.list();
		if(clients.size() > 0)
		{
			public_key = clients.get(0).getPublic_key();
		}
		return public_key;
	}
}
