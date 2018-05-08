package net.chain0.resources.asymmetric;

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

import net.zerochain.resources.Utils;
import net.zerochain.resources.crypto.asymmetric.*;

/**
 * Unit test for simple App.
 */
public class EDDSATest extends TestCase
{
    /**
     * Create the test case
     *
     * @param testName name of the test case
     */
    public EDDSATest( String testName )
    {
        super( testName );
    }

    /**
     * @return the suite of tests being tested
     */
    public static Test suite()
    {
        return new TestSuite( EDDSATest.class );
    }

    public void testSignHash()
    {
        AsymmetricSigning algo = new EDDSA();
        KeyPair keys = algo.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = Utils.toHexString(publicK.getEncoded());
        String privateString = Utils.toHexString(privateK.getEncoded());
        String data = Utils.stringToHexString("Random stuff");
        String hash = Utils.createHash(data);
        String sign = algo.createSignature(privateString,hash);
        assertFalse(sign.equals(""));
    }

    public void testVerifySignature()
    {
        AsymmetricSigning algo = new EDDSA();
        KeyPair keys = algo.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = Utils.toHexString(publicK.getEncoded());
        String privateString = Utils.toHexString(privateK.getEncoded());
        String data = Utils.stringToHexString("Random stuff");
        String hash = Utils.createHash(data);
        String sign = algo.createSignature(privateString,hash);
        assertTrue(algo.verifySignature(publicString,sign,hash));
    }

    public void testVerifySignatureWrongHash()
    {
        AsymmetricSigning algo = new EDDSA();
        KeyPair keys = algo.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = Utils.toHexString(publicK.getEncoded());
        String privateString = Utils.toHexString(privateK.getEncoded());
        String data = Utils.stringToHexString("Random stuff");
        String hash = Utils.createHash(data);
        String hash1 = Utils.createHash(Utils.stringToHexString("Other random stuff"));
        String sign = algo.createSignature(privateString,hash);
        assertFalse(algo.verifySignature(publicString,sign,hash1));
    }

    public void testVerifySignatureWrongHash1()
    {
        AsymmetricSigning algo = new EDDSA();
        KeyPair keys = algo.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = Utils.toHexString(publicK.getEncoded());
        String privateString = Utils.toHexString(privateK.getEncoded());
        String data = Utils.stringToHexString("Random stuff");
        String hash = Utils.createHash(data);
        String hash1 = Utils.createHash(Utils.stringToHexString("random stuff"));
        String sign = algo.createSignature(privateString,hash);
        assertFalse(algo.verifySignature(publicString,sign,hash1));
    }

    public void testVerifySignatureWrongHashWrongWayToHash()
    {
        AsymmetricSigning algo = new EDDSA();
        KeyPair keys = algo.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = Utils.toHexString(publicK.getEncoded());
        String privateString = Utils.toHexString(privateK.getEncoded());
        String hash = Utils.createHash("hello");
        String hash1 = Utils.createHash("Hello");
        String sign = algo.createSignature(privateString,hash);
        assertTrue(algo.verifySignature(publicString,sign,hash1));
    }

    public void testVerifySignatureWrongPublicKey()
    {
        AsymmetricSigning algo = new EDDSA();
        KeyPair keys = algo.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        String publicString = Utils.toHexString(algo.createKeys().getPublic().getEncoded());
        String privateString = Utils.toHexString(privateK.getEncoded());
        String data = Utils.stringToHexString("Random stuff");
        String hash = Utils.createHash(data);
        String sign = algo.createSignature(privateString,hash);
        assertFalse(algo.verifySignature(publicString,sign,hash));
    }

    public void testVerifySignatureWrongSign()
    {
        AsymmetricSigning algo = new EDDSA();
        KeyPair keys = algo.createKeys();
        KeyPair keys1 = algo.createKeys();
        PublicKey publicK = keys.getPublic();
        PrivateKey privateK = keys.getPrivate();
        PrivateKey privateK1 = keys1.getPrivate();
        String publicString = Utils.toHexString(publicK.getEncoded());
        String privateString = Utils.toHexString(privateK1.getEncoded());
        String data = Utils.stringToHexString("Random stuff");
        String hash = Utils.createHash(data);
        String sign = algo.createSignature(privateString,hash);
        assertFalse(algo.verifySignature(publicString,sign,hash));
    }

}