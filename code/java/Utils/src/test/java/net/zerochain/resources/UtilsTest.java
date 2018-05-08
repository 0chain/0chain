package net.zerochain.resources;

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
public class UtilsTest extends TestCase
{
    /**
     * Create the test case
     *
     * @param testName name of the test case
     */
    public UtilsTest( String testName )
    {
        super( testName );
    }

    /**
     * @return the suite of tests being tested
     */
    public static Test suite()
    {
        return new TestSuite( UtilsTest.class );
    }

    public void testStringTOHexAndBackLossless()
    {
        String test = "zcdjklpe";
        String hexTest = Utils.stringToHexString(test);
        assertEquals(test, Utils.hexStringToString(hexTest));
    }

    /**
    * Makes sure createHash creates the appropriate hash
    */
    public void testCreateHashHashesCorrectly()
    {
        DigestSHA3 sha3 = new Digest256();
        sha3.update(Utils.fromHexString(Utils.stringToHexString("hello")));
        assertEquals(Utils.toHexString(sha3.digest()),Utils.createHash(Utils.stringToHexString("hello")));
    }

    /**
    * Makes sure hashes are different from different strings
    */
    public void testCreateHashDifferentInputs()
    {
        String hash1 = Utils.createHash(Utils.stringToHexString("hello"));
        String hash2 = Utils.createHash(Utils.stringToHexString("Hello"));
        assertFalse(hash1.equals(hash2));
    }

    public void testCreateHashDifferentInputsOnlyIfHashedCorrectly()
    {
        String hash1 = Utils.createHash(Utils.stringToHexString("hello"));
        String hash2 = Utils.createHash(Utils.stringToHexString("Hello"));
        String hash3 = Utils.createHash("Hello");
        String hash4 = Utils.createHash("hello");
        assertFalse(hash1.equals(hash2) && !(hash3.equals(hash4)));
    }

    /**
    * Tests verifyHash works with a correct hash
    */
    public void testVerifyHashWorksForSameInput()
    {
        assertTrue(Utils.verifyHash(Utils.stringToHexString("hello"),Utils.createHash(Utils.stringToHexString("hello"))));
    }

    /**
    * Test verifyHash works with a wrong hash
    */
    public void testVerifyHashWorksForDifferentInput()
    {
        assertFalse(Utils.verifyHash(Utils.stringToHexString("Hello"),Utils.createHash(Utils.stringToHexString("hello"))));
    }

    public void testInTimeTrue()
    {
        Timestamp clientTime = Utils.getTimestamp();
        try
        {
            Thread.sleep(1);
        }
        catch(Exception e)
        {
            assertTrue(false);
        }
        Timestamp minerTime = Utils.getTimestamp();
        assertTrue(Utils.inTime(minerTime,clientTime));
    }

    public void testInTimeClientTooEarly()
    {
        Timestamp minerTime = Utils.getTimestamp();
        try
        {
            Thread.sleep(1);
        }
        catch(Exception e)
        {
            assertTrue(false);
        }
        Timestamp clientTime = Utils.getTimestamp();
        assertFalse(Utils.inTime(minerTime,clientTime));
    }

    public void testInTimeClientTooLate()
    {
        Timestamp clientTime = Utils.getTimestamp();
        try
        {
            Thread.sleep(5000);
        }
        catch(Exception e)
        {
            assertTrue(false);
        }
        
        Timestamp minerTime = Utils.getTimestamp();
        assertFalse(Utils.inTime(minerTime,clientTime));
    }

    public void testTimestampToString()
    {
        Timestamp timestamp = Utils.getTimestamp();
        String timestampString = Utils.timestampToString(timestamp);
        Timestamp timestamp1 = Utils.stringToTimestamp(timestampString);
        assertTrue((timestamp.getTime() - timestamp1.getTime()) == 0);
    }
}