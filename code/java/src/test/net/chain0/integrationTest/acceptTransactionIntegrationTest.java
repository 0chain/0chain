package net.chain0.integrationTest;

import junit.framework.Test;
import junit.framework.TestCase;
import junit.framework.TestSuite;
import org.bouncycastle.jcajce.provider.digest.SHA3.DigestSHA3;
import org.bouncycastle.jcajce.provider.digest.SHA3.Digest256;

//import net.chain0.client.registration.*;
//import net.chain0.client.registration.auth.*;
import net.chain0.client.registration.model.Client;
import net.chain0.client.registration.api.MinerAcceptClientApi;
import net.chain0.resources.utils;

import net.chain0.client.transaction.*;
import net.chain0.client.transaction.auth.*;
import net.chain0.client.transaction.model.*;
import net.chain0.client.transaction.api.MinerAcceptTransactionApi;

import java.security.KeyPair;

import java.util.*;

public class acceptTransactionIntegrationTest extends TestCase
{
    /**
     * Create the test case
     *
     * @param testName name of the test case
     */
    public acceptTransactionIntegrationTest( String testName )
    {
        super( testName );
    }

    /**
     * @return the suite of tests being tested
     */
    public static Test suite()
    {
        return new TestSuite( acceptTransactionIntegrationTest.class );
    }

    /**
    * This method tests the accept_transaction GET method
    */
    public void testTransactionGetAPI()
    {
        MinerAcceptTransactionApi apiInstance = new MinerAcceptTransactionApi();
        try {
            List<Transaction> result = apiInstance.acceptTransactionGet();
            assertTrue(true);
        } catch (Exception e) {
            assertTrue(false);
        }
    }   

    /**
    * This method tests sending an empty transaction
    * json to the accept_transction POST method
    */
    public void testTransactionAcceptEmptyJson()
    {
        MinerAcceptTransactionApi apiInstance = new MinerAcceptTransactionApi();
        Transaction body = new Transaction(); // Transaction | 
        try {
            Reply result = apiInstance.acceptTransactionPost(body);
            assertTrue(true);
        } catch (Exception e) {
            assertTrue(false);
        }
    }

    public void testTransactionAcceptCorrectTransaction()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        MinerAcceptTransactionApi apiInstance1 = new MinerAcceptTransactionApi();
        KeyPair keys = utils.createKeys();
        String publicKey = utils.toHexString(keys.getPublic().getEncoded());
        String privateKey = utils.toHexString(keys.getPrivate().getEncoded());
        String clientID = utils.createHash(publicKey);
        String signedClientID = utils.signHash(privateKey,clientID);
        String data = utils.stringToHexString("abc...xyz");
        String timeStamp = utils.getTimestampAsString();
        String hash = utils.createHash(utils.fromHexString(clientID+data+timeStamp));
        String sign = utils.signHash(privateKey,hash);
        try
        {
            Client c = new Client();
            c.setClientID(clientID);
            c.setPublicKey(publicKey);
            c.setSignature(signedClientID);
            net.chain0.client.registration.model.Reply r = apiInstance.acceptClientPost(c);
            Transaction body = new Transaction();
            body.setClientID(clientID);
            body.setData(data);
            body.setTimestamp(timeStamp);
            body.setHashMsg(hash);
            body.setSign(sign);
            Reply result = apiInstance1.acceptTransactionPost(body);
            assertEquals("Worked",result.getMessage());
        }catch(Exception e)
        {
            assertTrue(false);
        }   
    }

    public void testTransactionAcceptBadHash()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        MinerAcceptTransactionApi apiInstance1 = new MinerAcceptTransactionApi();
        KeyPair keys = utils.createKeys();
        String publicKey = utils.toHexString(keys.getPublic().getEncoded());
        String privateKey = utils.toHexString(keys.getPrivate().getEncoded());
        String clientID = utils.createHash(publicKey);
        String data = utils.stringToHexString("abc...xyz");
        String timeStamp = utils.getTimestampAsString();
        String hash = utils.createHash(utils.fromHexString(clientID+data+timeStamp));
        String sign = utils.signHash(privateKey,hash+"10031ae3");
        try
        {
            Client c = new Client();
            c.setClientID(clientID);
            c.setPublicKey(publicKey);
            net.chain0.client.registration.model.Reply r = apiInstance.acceptClientPost(c);
            Transaction body = new Transaction();
            body.setClientID(clientID);
            body.setData(data);
            body.setTimestamp(timeStamp);
            body.setHashMsg(hash);
            body.setSign(sign);
            Reply result = apiInstance1.acceptTransactionPost(body);
            assertEquals("Not an authorized transaction",result.getMessage());
        }catch(Exception e)
        {
            assertTrue(false);
        }   
    }

    public void testTransactionAcceptNotRegisteredClient()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        MinerAcceptTransactionApi apiInstance1 = new MinerAcceptTransactionApi();
        KeyPair keys = utils.createKeys();
        String publicKey = utils.toHexString(keys.getPublic().getEncoded());
        String privateKey = utils.toHexString(keys.getPrivate().getEncoded());
        String clientID = utils.createHash(publicKey);
        String data = utils.stringToHexString("abc...xyz");
        String timeStamp = utils.getTimestampAsString();
        String hash = utils.createHash(utils.fromHexString(clientID+data+timeStamp));
        String sign = utils.signHash(privateKey,hash);
        try
        {
            Transaction body = new Transaction();
            body.setClientID(clientID);
            body.setData(data);
            body.setTimestamp(timeStamp);
            body.setHashMsg(hash);
            body.setSign(sign);
            Reply result = apiInstance1.acceptTransactionPost(body);
            assertEquals("Not an authorized transaction",result.getMessage());
        }catch(Exception e)
        {
            assertTrue(false);
        }   
    }

    public void testTransactionAcceptWrongPrivateKeyToSign()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        MinerAcceptTransactionApi apiInstance1 = new MinerAcceptTransactionApi();
        KeyPair keys = utils.createKeys();
        KeyPair keys1 = utils.createKeys();
        String publicKey = utils.toHexString(keys.getPublic().getEncoded());
        String privateKey = utils.toHexString(keys1.getPrivate().getEncoded());
        String clientID = utils.createHash(publicKey);
        String data = utils.stringToHexString("abc...xyz");
        String timeStamp = utils.getTimestampAsString();
        String hash = utils.createHash(utils.fromHexString(clientID+data+timeStamp));
        String sign = utils.signHash(privateKey,hash);
        try
        {
            Client c = new Client();
            c.setClientID(clientID);
            c.setPublicKey(publicKey);
            net.chain0.client.registration.model.Reply r = apiInstance.acceptClientPost(c);
            Transaction body = new Transaction();
            body.setClientID(clientID);
            body.setData(data);
            body.setTimestamp(timeStamp);
            body.setHashMsg(hash);
            body.setSign(sign);
            Reply result = apiInstance1.acceptTransactionPost(body);
            assertEquals("Not an authorized transaction",result.getMessage());
        }catch(Exception e)
        {
            assertTrue(false);
        }   
    }

    public void testTransactionAcceptTransactionReplay()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        MinerAcceptTransactionApi apiInstance1 = new MinerAcceptTransactionApi();
        KeyPair keys = utils.createKeys();
        String publicKey = utils.toHexString(keys.getPublic().getEncoded());
        String privateKey = utils.toHexString(keys.getPrivate().getEncoded());
        String clientID = utils.createHash(publicKey);
        String data = utils.stringToHexString("abc...xyz");
        String timeStamp = utils.getTimestampAsString();
        String hash = utils.createHash(utils.fromHexString(clientID+data+timeStamp));
        String sign = utils.signHash(privateKey,hash);
        try
        {
            Client c = new Client();
            c.setClientID(clientID);
            c.setPublicKey(publicKey);
            net.chain0.client.registration.model.Reply r = apiInstance.acceptClientPost(c);
            Transaction body = new Transaction();
            body.setClientID(clientID);
            body.setData(data);
            body.setTimestamp(timeStamp);
            body.setHashMsg(hash);
            body.setSign(sign);
            apiInstance1.acceptTransactionPost(body);
            Reply result = apiInstance1.acceptTransactionPost(body);
            assertEquals("Not an authorized transaction",result.getMessage());
        }catch(Exception e)
        {
            assertTrue(false);
        }   
    }

    public void testTransactionAcceptTransactionTooLate()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        MinerAcceptTransactionApi apiInstance1 = new MinerAcceptTransactionApi();
        KeyPair keys = utils.createKeys();
        String publicKey = utils.toHexString(keys.getPublic().getEncoded());
        String privateKey = utils.toHexString(keys.getPrivate().getEncoded());
        String clientID = utils.createHash(publicKey);
        String data = utils.stringToHexString("abc...xyz");
        String timeStamp = utils.getTimestampAsString();
        String hash = utils.createHash(utils.fromHexString(clientID+data+timeStamp));
        String sign = utils.signHash(privateKey,hash);
        try
        {
            Client c = new Client();
            c.setClientID(clientID);
            c.setPublicKey(publicKey);
            net.chain0.client.registration.model.Reply r = apiInstance.acceptClientPost(c);
            Transaction body = new Transaction();
            body.setClientID(clientID);
            body.setData(data);
            body.setTimestamp(timeStamp);
            body.setHashMsg(hash);
            body.setSign(sign);
            try
            {
                Thread.sleep(5000);
            }
            catch(Exception e)
            {
                assertTrue(false);
            }
            Reply result = apiInstance1.acceptTransactionPost(body);
            assertEquals("Not an authorized transaction",result.getMessage());
        }catch(Exception e)
        {
            assertTrue(false);
        }   
    }
}

