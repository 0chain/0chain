package net.chain0.client.api.impl;

import net.chain0.client.api.*;
import net.chain0.client.model.*;
import net.chain0.client.postClient;

import com.sun.jersey.multipart.FormDataParam;

import net.chain0.client.model.Client;
import net.chain0.client.model.Reply;

import java.util.List;
import net.chain0.client.api.NotFoundException;

import java.io.InputStream;

import com.sun.jersey.core.header.FormDataContentDisposition;
import com.sun.jersey.multipart.FormDataParam;

import javax.ws.rs.core.Response;
import javax.ws.rs.core.SecurityContext;
import javax.validation.constraints.*;
@javax.annotation.Generated(value = "io.swagger.codegen.languages.JavaJerseyServerCodegen", date = "2018-05-01T09:04:58.044-07:00")
public class ClientApiServiceImpl extends ClientApiService {
    @Override
    public Response clientPost(Client body, SecurityContext securityContext)
    throws NotFoundException {
        // do some magic!
        return postClient.postClient(body);
    }
}
