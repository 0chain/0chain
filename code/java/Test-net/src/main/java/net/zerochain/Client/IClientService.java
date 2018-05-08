package net.zerochain.Client;

import net.zerochain.Response.Response;

public interface IClientService {
	void saveRegistration(ClientEntity clientEntity);
	boolean lookupClient(ClientEntity clientEntity);
	Response verifyNewClient(ClientEntity clientEntity);
}
