package net.chain0.miner.transaction.api.impl;

import net.chain0.miner.transaction.api.*;
import net.chain0.miner.transaction.model.*;
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
public class AcceptTransactionApiServiceImpl extends AcceptTransactionApiService {
    @Override
    public Response acceptTransactionGet(SecurityContext securityContext)
    throws NotFoundException {
        List<Transaction> list = new ArrayList<Transaction>();
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
                    ResultSet rs = stmt.executeQuery( "SELECT * FROM \"transaction\";" );

                    while ( rs.next() ) 
                    {
                        String hashKey = rs.getString("client_id");
                        String data  = rs.getString("data");
                        Timestamp timestamp = rs.getTimestamp("timestamp");
                        String hash = rs.getString("hash_msg");
                        String signature = rs.getString("sign");
                        Transaction temp = new Transaction();
                        temp.setClientID(hashKey);
                        temp.setData(data);
                        temp.setTimestamp(utils.timestampToString(timestamp));
                        temp.setSign(signature);
                        temp.setHashMsg(hash);
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
        } catch (SQLException ex) 
        {
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
            }
        }
        return Response.status(200).entity(list).build();
    }
    @Override
    public Response acceptTransactionPost(Transaction body, SecurityContext securityContext)
    throws NotFoundException {
        // do some magic!
        Reply response = new Reply();
        Timestamp minerTime = utils.getTimestamp();
        String hashKey = body.getClientID();
        String data = body.getData();
        String timestampS = body.getTimestamp();
        String hash = body.getHashMsg();
        String signature = body.getSign();
        Timestamp timestamp = utils.stringToTimestamp(timestampS);
        int statusCode = 0;
        Connection conn = null;

        try
        {
            Context initialContext = new InitialContext();
            Context environmentContext = (Context) initialContext.lookup("java:comp/env");
            DataSource dataSource = (DataSource) environmentContext.lookup("jdbc/postgres");
            conn = dataSource.getConnection();
            boolean isRegistered = utils.alreadyClient(hashKey, conn);
            String publicKey = "";
            if(isRegistered)
            {
                publicKey = utils.getPublicKeyFromHash(hashKey, conn);
            }
            boolean correctTransactionHash = isRegistered && verifyTransactionHash(publicKey,hashKey, data, timestamp, hash, signature);
            boolean freshTransaction = correctTransactionHash && utils.freshTransaction(hash, conn);
            boolean validTransaction = freshTransaction && utils.inTime(minerTime,timestamp);
            if(conn != null && validTransaction)
            {
                Statement stmt = null;
                try
                {
                    String query = "INSERT into \"transaction\" (client_id, data, timestamp, hash_msg, sign) VALUES ('"+ hashKey+"','"+data+"','"+timestamp+"','"+hash+"','"+signature+"');";
                    stmt = conn.createStatement();
                    int oid = stmt. executeUpdate(query);
                    statusCode = 201;
                    response.setName(oid+"");
                    response.setMessage("Worked");
                    stmt.close();          
                } catch ( Exception e ) 
                {
                    statusCode = 418;
                    response.setName(e.getClass().getName());
                    response.setMessage(e.getMessage());
                }
            }
            else if(conn != null && !validTransaction)
            {
                statusCode = 200;
                response.setName("ERROR");
                response.setMessage("Not an authorized transaction");
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

    public static boolean verifyTransactionHash(String publicKey, String hashKey,String data, Timestamp timestamp, String hash, String signature)
    {
        boolean sameHash = utils.verifyHash(utils.fromHexString(hashKey+data+utils.timestampToString(timestamp)), hash);

        boolean signedCorrectly = false;
        if(utils.verifyKey(publicKey))
        {
            signedCorrectly = utils.verifySignature(publicKey, signature, hash);
        }

        return sameHash && signedCorrectly;
    }
}
