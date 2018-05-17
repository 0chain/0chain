package net.zerochain.ClientDAO;

import net.zerochain.Client.ClientEntity;

public interface IClientDAO {
	String getClientPublic_key(String client_id);
}
