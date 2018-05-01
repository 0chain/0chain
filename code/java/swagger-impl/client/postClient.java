package net.chain0.client;

import javax.ws.rs.core.Response;
import javax.ws.rs.core.SecurityContext;
import javax.validation.constraints.*;

public class postClient
{
        public static Response postClient(net.chain0.client.model.Client body)
        {
        int statusCode = 0;
        String public_key = body.getPublicKey();
        String hash = body.getClientID();
        String sign = body.getSignature();
        net.chain0.client.model.Reply response = new net.chain0.client.model.Reply();
        response.setName("TEST");
        response.setMessage("TESTING");

        boolean correctHash = net.chain0.resources.utils.verifyHash(public_key, hash);
        return Response.status(statusCode).entity(response).build();
        }
}