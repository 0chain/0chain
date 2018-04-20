package net.chain0.integrationTest;

import junit.framework.Test;
import junit.framework.TestCase;
import junit.framework.TestSuite;
import org.bouncycastle.jcajce.provider.digest.SHA3.DigestSHA3;
import org.bouncycastle.jcajce.provider.digest.SHA3.Digest256;

import net.chain0.client.registration.*;
import net.chain0.client.registration.auth.*;
import net.chain0.client.registration.model.*;
import net.chain0.client.registration.api.MinerAcceptClientApi;
import net.chain0.resources.utils;

import java.security.KeyPair;
import java.security.KeyPairGenerator;

import java.io.File;
import java.util.*;

/**
 * Unit test for simple App.
 */
public class acceptClientIntegrationTest extends TestCase
{
    /**
     * Create the test case
     *
     * @param testName name of the test case
     */
    public acceptClientIntegrationTest( String testName )
    {
        super( testName );
    }

    /**
     * @return the suite of tests being tested
     */
    public static Test suite()
    {
        return new TestSuite( acceptClientIntegrationTest.class );
    }

    /**
    * This method tests the accept_client GET method
    */
    public void testGetClientAPI()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        try {
            List<Client> result = apiInstance.acceptClientGet();
            assertTrue( true );
        } catch (ApiException e) {
            System.err.println("Exception when calling DefaultApi#acceptClientGet");
            e.printStackTrace();
            assertTrue( false );
        }
    }
    
    /*
    * This method tests sending an empty client json 
    * to the accpet_client POST method
    */
    public void testPostClientEmptyClientJson()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        Client body = new Client(); // Client | 
        try {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"FAILED! That was not an EC public key");
        } catch (ApiException e) {
            System.out.println("Code: "+e.getCode());
            System.out.println("Body: "+e.getResponseBody());
            assertTrue(false);
        }
    }

    /*
    * This method tests sending a client json with only 
    * the public key to the accpet_client POST method
    */
    public void testPostClientPublicKeyOnlyJson()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        String publicKey = utils.createPublicKey();
        Client body = new Client(); // Client | 
        body.setPublicKey(publicKey);
        try {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"FAILED! Hash was wrong");
        } catch (ApiException e) {
            System.out.println("Code: "+e.getCode());
            System.out.println("Body: "+e.getResponseBody());
            assertTrue(false);
        }
    }

    /*
    * This method tests sending a client json with only 
    * the clientID to the accpet_client POST method
    */
    public void testPostClientClientIDOnlyJson()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        String publicKey = utils.createPublicKey();
        String clientID = utils.createHash(publicKey);
        Client body = new Client(); // Client | 
        body.setClientID(clientID);
        try {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"FAILED! That was not an EC public key");
        } catch (ApiException e) {
            System.out.println("Code: "+e.getCode());
            System.out.println("Body: "+e.getResponseBody());
            assertTrue(false);
        }
    }

    /*
    * This method tests sending a client json filled 
    * out correctly to the accpet_client POST method
    */
    public void testPostClientCorrectJson()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        KeyPair keys = utils.createKeys();
        String publicKey = utils.toHexString(keys.getPublic().getEncoded());
        String privateKey = utils.toHexString(keys.getPrivate().getEncoded());
        String clientID = utils.createHash(publicKey);
        String sign = utils.signHash(privateKey,clientID);
        Client body = new Client(); // Client | 
        body.setPublicKey(publicKey);
        body.setSignature(sign);
        body.setClientID(clientID);
        try {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"Client Accepted");
        } catch (ApiException e) {
            e.printStackTrace();
            assertTrue(false);
        }
    }

    /*
    * This method tests sending a client json with a correct 
    * public key and a bad hash to the accpet_client POST method
    */
    public void testPostClientCorrectPublicKeyWrongHash()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        String publicKey = utils.createPublicKey();
        String clientID = "this hash is wrong";
        Client body = new Client(); // Client | 
        body.setPublicKey(publicKey);
        body.setClientID(clientID);
        try {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"FAILED! Hash was wrong");
        } catch (ApiException e) {
            e.printStackTrace();
            assertTrue(false);
        }
    }

    /*
    * This method tests sending a client json with a bad public key and 
    * a hash of a correct publick key hash to the accpet_client POST method
    */
    public void testPostClientCorrectHashWrongPublicKey()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        String publicKey = utils.createPublicKey();
        String clientID = utils.createHash(publicKey);
        Client body = new Client(); // Client | 
        body.setPublicKey("wrong public key");
        body.setClientID(clientID);
        try {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"FAILED! That was not an EC public key");
        } catch (ApiException e) {
            e.printStackTrace();
            assertTrue(false);
        }
    }

    public void testPostClientWrongPublicKey()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        String publicKey = utils.createPublicKey();
        String publicKey1 = utils.createPublicKey();
        String clientID = utils.createHash(publicKey1);
        Client body = new Client(); // Client | 
        body.setPublicKey(publicKey);
        body.setClientID(clientID);
        try {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"FAILED! Hash was wrong");
        } catch (ApiException e) {
            e.printStackTrace();
            assertTrue(false);
        }
    }

    public void testPostClientSomeoneAccidentlySendsPrivateKey()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        KeyPair keys = utils.createKeys();
        String privateKey = utils.toHexString(keys.getPrivate().getEncoded());
        String wrongClientID = utils.createHash(privateKey);
        Client body = new Client(); // Client | 
        body.setPublicKey(privateKey);
        body.setClientID(wrongClientID);
        try {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"FAILED! That was not an EC public key");
        } catch (ApiException e) {
            e.printStackTrace();
            assertTrue(false);
        }
    }

    public void testWrongSignature()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        KeyPair keys = utils.createKeys();
        String publicKey = utils.toHexString(keys.getPublic().getEncoded());
        String privateKey = utils.toHexString(keys.getPrivate().getEncoded());
        String clientID = utils.createHash(publicKey);
        String sign = utils.signHash(privateKey,"12032ace314");
        Client body = new Client(); // Client | 
        body.setPublicKey(publicKey);
        body.setSignature(sign);
        body.setClientID(clientID);
        try
        {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"Signature is bad");
        } catch (ApiException e) {
            e.printStackTrace();
            assertTrue(false);
        }
    }

    public void testPostClientSomeoneSendsRSAKey()
    {
        MinerAcceptClientApi apiInstance = new MinerAcceptClientApi();
        String publicKey = "";
        try {
            KeyPairGenerator keyGen = KeyPairGenerator.getInstance("RSA");
            KeyPair keypair = keyGen.generateKeyPair();
            publicKey = utils.toHexString(keypair.getPublic().getEncoded());
        } catch (Exception e) {
            // TODO Auto-generated catch block
            e.printStackTrace();
            assertTrue(false);
        }
        String clientID = utils.createHash(publicKey);
        Client body = new Client(); // Client | 
        body.setPublicKey(publicKey);
        body.setClientID(clientID);
        try {
            Reply result = apiInstance.acceptClientPost(body);
            assertEquals(result.getMessage(),"FAILED! That was not an EC public key");
        } catch (ApiException e) {
            e.printStackTrace();
            assertTrue(false);
        }
    }
}