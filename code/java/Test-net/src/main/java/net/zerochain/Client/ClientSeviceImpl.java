package net.zerochain.Client;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import net.zerochain.ClientDAO.IClientDAO;
import net.zerochain.resources.Utils;
import net.zerochain.resources.crypto.asymmetric.*;
import net.zerochain.Response.Response;

@Service("minerService")
public class ClientSeviceImpl implements IClientService {
	@Autowired 
	private IClientDAO iClientDAO;

	@Override
	public void saveRegistration(ClientEntity clientEntity) {
		// TODO Auto-generated method stub
		iClientDAO.saveRegistration(clientEntity);	
	}

	@Override
	public boolean lookupClient(ClientEntity clientEntity)
	{
		return iClientDAO.lookupClient(clientEntity);
	}

	@Override
	public Response verifyNewClient(ClientEntity clientEntity)
	{
		Response response = new Response();
		AsymmetricSigning algo = new EDDSA();
		boolean validKey = algo.verifyKey(clientEntity.getPublic_key());
		boolean correctHash = validKey && Utils.verifyHash(clientEntity.getPublic_key(), clientEntity.getClientID());
		boolean newClient = correctHash && !lookupClient(clientEntity);
		if(clientEntity.getPublic_key()==null || clientEntity.getClientID() == null)
		{
			response.setName("Error");
			response.setMessage("Bad json... BAD!!!");
		}
		else if(!validKey)
		{
			response.setName("Error");
			response.setMessage("Invalid key");
		}
		else if(!correctHash)
		{
			response.setName("Error");
			response.setMessage("Invalid clientID");
		}
		else if(!newClient)
		{
			response.setName("Error");
			response.setMessage("Client already exists");
		}
		else
		{
			response.setName("Success");
			response.setMessage("Client registration completed");
		}

        return response;
	}

}
