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

import net.zerochain.resources.Utils;
import net.zerochain.resources.crypto.asymmetric.*;
import net.zerochain.Client.ClientEntity;
import net.zerochain.Response.Response;
import net.zerochain.Application;

@RunWith(SpringRunner.class)
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
public class ClientTest {

	@Autowired
    private TestRestTemplate restTemplate;

	@Test
	public void testEmptyClientPost() {
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", new ClientEntity(), Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Bad json... BAD!!!",client.getMessage());
	}

	@Test 
	public void testPublicKeyOnlyPost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		ClientEntity clientEntity = new ClientEntity();
		clientEntity.setPublic_key(public_key);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", clientEntity, Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Bad json... BAD!!!",client.getMessage());
	}

	@Test
	public void testClientIDOnlyPost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		ClientEntity clientEntity = new ClientEntity();
		clientEntity.setHash_key(client_id);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", clientEntity, Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Bad json... BAD!!!",client.getMessage());
	}

	@Test
	public void testSignOnlyPost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		ClientEntity clientEntity = new ClientEntity();
		clientEntity.setSign(sign);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", clientEntity, Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Bad json... BAD!!!",client.getMessage());
	}

	@Test
	public void testWrongHashPost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key+"03AF");
		String sign = algo.createSignature(private_key,client_id);
		ClientEntity clientEntity = new ClientEntity();
		clientEntity.setPublic_key(public_key);
		clientEntity.setHash_key(client_id);
		clientEntity.setSign(sign);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", clientEntity, Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Invalid clientID",client.getMessage());
	}

	@Test
	public void testWrongPublicKeyPost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		ClientEntity clientEntity = new ClientEntity();
		clientEntity.setPublic_key(public_key+"0B2C");
		clientEntity.setHash_key(client_id);
		clientEntity.setSign(sign);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", clientEntity, Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Invalid key",client.getMessage());
	}

	@Test
	public void testSentPrivateKeyPost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(private_key);
		String sign = algo.createSignature(private_key,client_id);
		ClientEntity clientEntity = new ClientEntity();
		clientEntity.setPublic_key(private_key);
		clientEntity.setHash_key(client_id);
		clientEntity.setSign(sign);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", clientEntity, Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Invalid key",client.getMessage());
	}

	@Test
	public void testWrongSignaturePost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		KeyPair keys1 = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(Utils.toHexString(keys1.getPrivate().getEncoded()),client_id);
		ClientEntity clientEntity = new ClientEntity();
		clientEntity.setPublic_key(public_key);
		clientEntity.setHash_key(client_id);
		clientEntity.setSign(sign);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", clientEntity, Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Invalid signature",client.getMessage());
	}

	@Test
	public void testWrongAsymmetricAlgoPost()
	{
		AsymmetricSigning algo = new ECDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Invalid key",client.getMessage());
	}

	@Test
	public void testRegisteredClientPost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Error",client.getName());
        assertEquals("Client already exists",client.getMessage());
	}

	@Test
	public void testGoodClientPost()
	{
		AsymmetricSigning algo = new EDDSA();
		KeyPair keys = algo.createKeys();
		String private_key = Utils.toHexString(keys.getPrivate().getEncoded());
		String public_key = Utils.toHexString(keys.getPublic().getEncoded());
		String client_id = Utils.createHash(public_key);
		String sign = algo.createSignature(private_key,client_id);
		ResponseEntity<Response> responseEntity = restTemplate.postForEntity("/v1/registration", new ClientEntity(public_key,client_id,sign), Response.class);
        Response client = responseEntity.getBody();
        assertEquals("Success",client.getName());
        assertEquals("Client registration completed",client.getMessage());
	}

}
