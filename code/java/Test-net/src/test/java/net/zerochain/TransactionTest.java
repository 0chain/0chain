package net.zerochain;

import org.junit.Test;
import static org.junit.Assert.*;
import org.junit.runner.RunWith;
import org.skyscreamer.jsonassert.JSONAssert;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.context.embedded.LocalServerPort;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpStatus;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.http.ResponseEntity;
import org.springframework.test.context.junit4.SpringRunner;
import org.apache.log4j.Logger;

import java.security.PrivateKey;
import java.security.PublicKey;
import java.security.KeyPair;
import java.sql.Timestamp;

import net.zerochain.Application;
import net.zerochain.resources.Utils;
import net.zerochain.resources.crypto.asymmetric.*;
import net.zerochain.Client.ClientEntity;
import net.zerochain.Transaction.TransactionEntity;
import net.zerochain.Response.Response;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
public class TransactionTest {

	@Autowired
    private TestRestTemplate restTemplate;

    @Test
    public void testEmptyTransactionPost()
    {
    	AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);

		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/transaction", new TransactionEntity(), Response.class);

		Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("JSON not filled in correctly",client.getMessage());		
    }

    @Test
    public void testBashHashTransactionPost()
    {
    	AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);

		String data = "test";
		Timestamp timestamp = Utils.getTimestamp();
		String hash = Utils.createHash(client_id+data+Utils.timestampToString(timestamp)+"BBAA007842");
		String sign1 = algo.createSignature(private_key, hash);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/transaction", new TransactionEntity(client_id,data,timestamp,hash,sign1), Response.class);

		Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Bad transaction hash",client.getMessage());
    }

    @Test
    public void testUnregisteredClientTransactionPost()
    {
    	AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);

		String data = "test";
		Timestamp timestamp = Utils.getTimestamp();
		String hash = Utils.createHash(client_id+data+Utils.timestampToString(timestamp));
		String sign1 = algo.createSignature(private_key, hash);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/transaction", new TransactionEntity(client_id,data,timestamp,hash,sign1), Response.class);

		Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Not registered",client.getMessage());
    }

    @Test
    public void testWrongPrivateKeySignedTransactionPost()
    {
    	AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);

		String wrongPrivate_key = Utils.toHexString(algo.createKeys().getPrivate().getEncoded());

		String data = "test";
		Timestamp timestamp = Utils.getTimestamp();
		String hash = Utils.createHash(client_id+data+Utils.timestampToString(timestamp));
		String sign1 = algo.createSignature(wrongPrivate_key, hash);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/transaction", new TransactionEntity(client_id,data,timestamp,hash,sign1), Response.class);

		Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Signature is just an X",client.getMessage());
    }

    @Test
    public void testReplayTransactionPost()
    {

		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);

		String data = "test";
		Timestamp timestamp = Utils.getTimestamp();
		String hash = Utils.createHash(client_id+data+Utils.timestampToString(timestamp));
		String sign1 = algo.createSignature(private_key, hash);
		restTemplate.postForEntity("/v1/transaction", new TransactionEntity(client_id,data,timestamp,hash,sign1), Response.class);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/transaction", new TransactionEntity(client_id,data,timestamp,hash,sign1), Response.class);

		Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("I've heard this one before",client.getMessage());
    }

    @Test
    public void testTooLateTransactionPost()
    {
    	AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);

		String data = "test";
		Timestamp timestamp = Utils.getTimestamp();
		try
        {
            Thread.sleep(5000);
        }
        catch(Exception e)
        {
            assertTrue(false);
        }
		String hash = Utils.createHash(client_id+data+Utils.timestampToString(timestamp));
		String sign1 = algo.createSignature(private_key, hash);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/transaction", new TransactionEntity(client_id,data,timestamp,hash,sign1), Response.class);

		Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Too old",client.getMessage());
    }

	@Test
	public void testGoodTransactionPost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);

		String data = "test";
		Timestamp timestamp = Utils.getTimestamp();
		String hash = Utils.createHash(client_id+data+Utils.timestampToString(timestamp));
		String sign1 = algo.createSignature(private_key, hash);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/transaction", new TransactionEntity(client_id,data,timestamp,hash,sign1), Response.class);

		Response client = responseEntity.getBody();
        assertEquals("Success",client.getName());
        assertEquals("Transaction accepted",client.getMessage());
	}

}
