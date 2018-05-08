package net.zerochain.ClientDAO;

import net.zerochain.Client.ClientEntity;

public interface IClientDAO {
	void saveRegistration(ClientEntity clientEntity);
	boolean lookupClient(ClientEntity clientEntity);
	String getClientPublic_key(ClientEntity clientEntity);
}
