package net.chain0.miner.registration.api.impl;

import net.chain0.miner.registration.api.*;
import net.chain0.miner.registration.model.*;
import net.chain0.resources.utils;

import com.sun.jersey.multipart.FormDataParam;

import java.util.List;

import java.io.InputStream;

import com.sun.jersey.core.header.FormDataContentDisposition;
import com.sun.jersey.multipart.FormDataParam;

import javax.ws.rs.core.Response;
import javax.ws.rs.core.SecurityContext;
import javax.validation.constraints.*;

import org.bouncycastle.jcajce.provider.digest.SHA3.DigestSHA3;
import org.bouncycastle.jcajce.provider.digest.SHA3.Digest256;
import org.bouncycastle.jce.provider.BouncyCastleProvider;
import org.bouncycastle.pqc.math.linearalgebra.ByteUtils;
import java.security.KeyFactory;
import java.security.PublicKey;
import java.security.spec.X509EncodedKeySpec;
import java.security.Security;

import javax.naming.Context;
import javax.naming.InitialContext;
import javax.naming.NamingException;
import javax.sql.DataSource;

import java.io.StringWriter;
import java.io.PrintWriter;
import java.util.ArrayList;
import java.util.List;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.Statement;
import java.sql.Timestamp;
import java.sql.SQLException;
import java.sql.ResultSet;

@javax.annotation.Generated(value = "io.swagger.codegen.languages.JavaJerseyServerCodegen", date = "2018-04-09T22:36:37.747Z")
public class AcceptClientApiServiceImpl extends AcceptClientApiService {
    @Override
    public Response acceptClientGet(SecurityContext securityContext)
    throws NotFoundException {
        List<Client> list = new ArrayList<Client>();
        Connection conn = null;
 
        try 
        {
            Context initialContext = new InitialContext();
            Context environmentContext = (Context) initialContext.lookup("java:comp/env");
            DataSource dataSource = (DataSource) environmentContext.lookup("jdbc/postgres");
            conn = dataSource.getConnection();
            if (conn != null) 
            {
                Statement stmt = null;
                try 
                {
                    stmt = conn.createStatement();
                    ResultSet rs = stmt.executeQuery( "SELECT * FROM clients;" );
                    
                    while ( rs.next() ) 
                    {
                        String public_key = rs.getString("public_key");
                        String hash_key = rs.getString("hash_key");
                        String sign = rs.getString("sign");
                        Client temp = new Client();
                        temp.setPublicKey(public_key);
                        temp.setClientID(hash_key);
                        temp.setSignature(sign);
                        list.add(temp);
                    }

                    rs.close();     
                    stmt.close();          
                } catch ( Exception e ) 
                {
                }                 
            }
        } catch ( NamingException ne) 
        {
        } catch (SQLException ex) {
            StringWriter sw = new StringWriter();
            PrintWriter pw = new PrintWriter(sw);
            ex.printStackTrace(pw);
            String sStackTrace = sw.toString(); // stack trace as a string
        } finally 
        {
            try 
            {
                if (conn != null && !conn.isClosed()) 
                {
                    conn.close();
                }
            } catch (SQLException ex) 
            {
                ex.printStackTrace();
            }
        }

        return Response.status(200).entity(list).build();
    }
    @Override
    public Response acceptClientPost(Client body, SecurityContext securityContext)
    throws NotFoundException {
        // do some magic!
        int statusCode = 0;
        String public_key = body.getPublicKey();
        String hash = body.getClientID();
        String sign = body.getSignature();
        Connection conn = null;
        Reply response = new Reply();
        response.setName("");
        response.setMessage("");

        try
        {
            Context initialContext = new InitialContext();
            Context environmentContext = (Context) initialContext.lookup("java:comp/env");
            DataSource dataSource = (DataSource) environmentContext.lookup("jdbc/postgres");
            conn = dataSource.getConnection();
            boolean correctHash = utils.verifyHash(public_key, hash);
            boolean newClient = !utils.alreadyClient(hash, conn);
            boolean validKey = utils.verifyKey(public_key);
            boolean signedCorrectly = false;
            if(validKey)
            {
                signedCorrectly = utils.verifySignature(public_key, sign, hash);
            }
            if(conn != null && correctHash && newClient && validKey && signedCorrectly)
            {
                Statement stmt = null;
                try
                {
                    String query = "INSERT into clients (public_key, hash_key, sign) VALUES ('"+public_key+"','"+hash+"','"+sign+"');";
                    stmt = conn.createStatement();
                    int oid = stmt. executeUpdate(query);
                    if(oid == 1)
                    {
                        response.setName("Status");
                        response.setMessage("Client Accepted");
                        statusCode = 201;
                    }    
                    stmt.close();          
                } catch ( Exception e ) 
                {
                    statusCode = 418;
                    response.setName(e.getClass().getName());
                    response.setMessage(e.getMessage());
                }
            }
            else if(!validKey)
            {
                statusCode = 200;
                response.setName("ERROR");
                response.setMessage("FAILED! That was not an EC public key");
            }
            else if(!correctHash)
            {
                statusCode = 200;
                response.setName("ERROR");
                response.setMessage("FAILED! Hash was wrong");
            }
            else if(!newClient) 
            {
                statusCode = 208;
                response.setName("ERROR");
                response.setMessage("Already registered");
            }
            else if(!signedCorrectly)
            {
                statusCode = 208;
                response.setName("ERROR");
                response.setMessage("Signature is bad");
            }
        } catch ( NamingException ne) 
        {
            statusCode = 418;
            response.setName(ne.getClass().getName());
            response.setMessage(ne.getMessage());
        } catch (SQLException ex) 
        {
            StringWriter sw = new StringWriter();
            PrintWriter pw = new PrintWriter(sw);
            ex.printStackTrace(pw);
            statusCode = 418;
            response.setName("SQL Exception");
            response.setMessage(sw.toString());
        } finally 
        {
            try 
            {
                if (conn != null && !conn.isClosed()) 
                {
                    conn.close();
                }
            } catch (SQLException ex) 
            {
                StringWriter sw = new StringWriter();
                PrintWriter pw = new PrintWriter(sw);
                ex.printStackTrace(pw);
                statusCode = 418;
                response.setName("SQL Exception");
                response.setMessage(sw.toString());
            }
        }
        return Response.status(statusCode).entity(response).build();
    }
}
