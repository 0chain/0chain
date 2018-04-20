package net.chain0.resources;

import junit.framework.Test;
import junit.framework.TestCase;
import junit.framework.TestSuite;
import org.bouncycastle.jcajce.provider.digest.SHA3.DigestSHA3;
import org.bouncycastle.jcajce.provider.digest.SHA3.Digest256;

import java.security.KeyPair;
import java.security.PublicKey;
import java.security.PrivateKey;

import java.io.File;
import java.util.*;
import java.sql.Timestamp;

/**
 * Unit test for simple App.
 */
public class utilsTest extends TestCase
{
    /**
     * Create the test case
     *
     * @param testName name of the test case
     */
    public utilsTest( String testName )
    {
        super( testName );
    }

    /**
     * @return the suite of tests being tested
     */
    public static Test suite()
    {
        return new TestSuite( utilsTest.class );
    }

    public void testStringTOHexAndBackLossless()
    {
        String test = "zcdjklpe";
        String hexTest = utils.stringToHexString(test);
        assertEquals(test, utils.hexStringToString(hexTest));
    }

    /**
    * Makes sure createPublicKey doesn't return an empty string
    */
    public void testCreatePublicKeyProducesString()
    {
        assertFalse("".equals(utils.createPublicKey()));
    }

    /**
    * Makes sure getPublicKey doesn't return a null
    */
    public void testGetPublicKeyReturnsAKey()
    {
        assertFalse(null == utils.getPublicKey(utils.createPublicKey()));
    }

    /**
    * Makes sure verifyKey returns true since getPublicKey doesn't return null
    */
    public void testVerifyKeyVerifiesRealKey()
    {
        String key = utils.createPublicKey();
        assertTrue((null != utils.getPublicKey(key)) && utils.verifyKey(key));
    }

    /**
    * Makes sure verifyKey returns false since the string isn't a public key
    */
    public void testVerifyKeyRejectsBadKey()
    {
        assertFalse(utils.verifyKey("rraeaasgrage"));
    }

    /**
    * Makes sure createHash creates the appropriate hash
    */
    public void testCreateHashHashesCorrectly()
    {
        DigestSHA3 sha3 = new Digest256();
        sha3.update(utils.fromHexString(utils.stringToHexString("hello")));
        assertEquals(utils.toHexString(sha3.digest()),utils.createHash(utils.stringToHexString("hello")));
    }

    /**
    * Makes sure hashes are different from different strings
    */
    public void testCreateHashDifferentInputs()
    {
        String hash1 = utils.createHash(utils.stringToHexString("hello"));
        String hash2 = utils.createHash(utils.stringToHexString("Hello"));
        assertFalse(hash1.equals(hash2));
    }

    public void testCreateHashDifferentInputsOnlyIfHashedCorrectly()
    {
        String hash1 = utils.createHash(utils.stringToHexString("hello"));
        String hash2 = utils.createHash(utils.stringToHexString("Hello"));
        String hash3 = utils.createHash("Hello");
        String hash4 = utils.createHash("hello");
        assertFalse(hash1.equals(hash2) && !(hash3.equals(hash4)));
    }

    /**
    * Tests verifyHash works with a correct hash
    */
    public void testVerifyHashWorksForSameInput()
    {
        assertTrue(utils.verifyHash(utils.stringToHexString("hello"),utils.createHash(utils.stringToHexString("hello"))));
    }

    /**
    * Test verifyHash works with a wrong hash
    */
    public void testVerifyHashWorksForDifferentInput()
    {
        assertFalse(utils.verifyHash(utils.stringToHexString("Hello"),utils.createHash(utils.stringToHexString("hello"))));
    }

    public void testSignHash()
    {
        KeyPair keys = utils.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = utils.toHexString(publicK.getEncoded());
        String privateString = utils.toHexString(privateK.getEncoded());
        String data = utils.stringToHexString("Random stuff");
        String hash = utils.createHash(data);
        String sign = utils.signHash(privateString,hash);
        assertFalse(sign.equals(""));
    }

    public void testVerifySignature()
    {
        KeyPair keys = utils.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = utils.toHexString(publicK.getEncoded());
        String privateString = utils.toHexString(privateK.getEncoded());
        String data = utils.stringToHexString("Random stuff");
        String hash = utils.createHash(data);
        String sign = utils.signHash(privateString,hash);
        assertTrue(utils.verifySignature(publicString,sign,hash));
    }

    public void testVerifySignatureWrongHash()
    {
        KeyPair keys = utils.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = utils.toHexString(publicK.getEncoded());
        String privateString = utils.toHexString(privateK.getEncoded());
        String data = utils.stringToHexString("Random stuff");
        String hash = utils.createHash(data);
        String hash1 = utils.createHash(utils.stringToHexString("Other random stuff"));
        String sign = utils.signHash(privateString,hash);
        assertFalse(utils.verifySignature(publicString,sign,hash1));
    }

    public void testVerifySignatureWrongHash1()
    {
        KeyPair keys = utils.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = utils.toHexString(publicK.getEncoded());
        String privateString = utils.toHexString(privateK.getEncoded());
        String data = utils.stringToHexString("Random stuff");
        String hash = utils.createHash(data);
        String hash1 = utils.createHash(utils.stringToHexString("random stuff"));
        String sign = utils.signHash(privateString,hash);
        assertFalse(utils.verifySignature(publicString,sign,hash1));
    }

    public void testVerifySignatureWrongHashWrongWayToHash()
    {
        KeyPair keys = utils.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = utils.toHexString(publicK.getEncoded());
        String privateString = utils.toHexString(privateK.getEncoded());
        String hash = utils.createHash("hello");
        String hash1 = utils.createHash("Hello");
        String sign = utils.signHash(privateString,hash);
        assertTrue(utils.verifySignature(publicString,sign,hash1));
    }

    public void testVerifySignatureWrongPublicKey()
    {
        KeyPair keys = utils.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = utils.createPublicKey();
        String privateString = utils.toHexString(privateK.getEncoded());
        String data = utils.stringToHexString("Random stuff");
        String hash = utils.createHash(data);
        String sign = utils.signHash(privateString,hash);
        assertFalse(utils.verifySignature(publicString,sign,hash));
    }

    public void testVerifySignatureWrongSign()
    {
        KeyPair keys = utils.createKeys();
        KeyPair keys1 = utils.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        PrivateKey privateK1 = keys1.getPrivate();
        String publicString = utils.toHexString(publicK.getEncoded());
        String privateString = utils.toHexString(privateK1.getEncoded());
        String data = utils.stringToHexString("Random stuff");
        String hash = utils.createHash(data);
        String sign = utils.signHash(privateString,hash);
        assertFalse(utils.verifySignature(publicString,sign,hash));
    }

    public void testInTimeTrue()
    {
        Timestamp clientTime = utils.getTimestamp();
        try
        {
            Thread.sleep(1);
        }
        catch(Exception e)
        {
            assertTrue(false);
        }
        Timestamp minerTime = utils.getTimestamp();
        assertTrue(utils.inTime(minerTime,clientTime));
    }

    public void testInTimeClientTooEarly()
    {
        Timestamp minerTime = utils.getTimestamp();
        try
        {
            Thread.sleep(1);
        }
        catch(Exception e)
        {
            assertTrue(false);
        }
        Timestamp clientTime = utils.getTimestamp();
        assertFalse(utils.inTime(minerTime,clientTime));
    }

    public void testInTimeClientTooLate()
    {
        Timestamp clientTime = utils.getTimestamp();
        try
        {
            Thread.sleep(5000);
        }
        catch(Exception e)
        {
            assertTrue(false);
        }
        
        Timestamp minerTime = utils.getTimestamp();
        assertFalse(utils.inTime(minerTime,clientTime));
    }

    public void testTimestampToString()
    {
        Timestamp timestamp = utils.getTimestamp();
        String timestampString = utils.timestampToString(timestamp);
        Timestamp timestamp1 = utils.stringToTimestamp(timestampString);
        assertTrue((timestamp.getTime() - timestamp1.getTime()) == 0);
    }
}
