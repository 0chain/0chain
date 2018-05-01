package net.chain0.resources.crypto.asymmetric;

import org.bouncycastle.jcajce.provider.digest.SHA3.DigestSHA3;
import org.bouncycastle.jcajce.provider.digest.SHA3.Digest256;
import org.bouncycastle.jce.provider.BouncyCastleProvider;

import java.security.InvalidAlgorithmParameterException;
import java.security.InvalidKeyException;
import java.security.KeyPair;
import java.security.KeyFactory;
import java.security.spec.X509EncodedKeySpec;
import java.security.spec.PKCS8EncodedKeySpec;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.security.NoSuchProviderException;
import java.security.PrivateKey;
import java.security.PublicKey;
import java.security.Security;
import java.security.Signature;
import java.security.spec.X509EncodedKeySpec;
import net.chain0.resources.utils;

public class ECDSA implements AsymmetricEncryption
{
    @Override
	public KeyPair createKeys()
	{
		KeyPair key = null;
        try {
            KeyPairGenerator keyGen = KeyPairGenerator.getInstance("EC");
            key = keyGen.generateKeyPair();
        } catch (Exception e) {
        	e.printStackTrace();
        }
        return key;
	}

    @Override
	public String createSignature(String privateKey, String hash)
	{
		String signature = "";
        try{
            Signature sign = getSignature();
            KeyFactory kf = KeyFactory.getInstance("EC");
            sign.initSign(kf.generatePrivate(new PKCS8EncodedKeySpec(utils.fromHexString(privateKey))));
            sign.update(utils.fromHexString(hash));
            signature = utils.toHexString(sign.sign());
        }catch(Exception e)
        {
        }
        return signature;
	}

    @Override
	public boolean verifySignature(String publicKey, String signature, String hash)
	{
		boolean signedCorrectly = false;
        PublicKey key = null;
        try
        {
            key = getPublicKey(publicKey);
        } catch(Throwable e)
        {

        }

        Signature sign = getSignature();
        if(key != null)
        {
            try
            {
                sign.initVerify(key);
                sign.update(utils.fromHexString(hash));
                signedCorrectly = sign.verify(utils.fromHexString(signature));
            }catch(Exception e)
            {
                signedCorrectly = false;
            }
        }
        return signedCorrectly;
	}

    @Override
	public Signature getSignature()
	{
		Signature signature = null;
        try {
            signature = Signature.getInstance("SHA3-256WITHECDSA", new BouncyCastleProvider());
        } catch (Exception e) {

        } 
        return signature;

	}

    @Override
	public boolean verifyKey(String public_key)
	{
		boolean isKey = false;
        try{
            byte[] hash = utils.fromHexString(public_key);
            X509EncodedKeySpec X509publicKey = new X509EncodedKeySpec(hash);
            KeyFactory kf = KeyFactory.getInstance("EC");
            try
            {
                kf.generatePublic(X509publicKey);
                isKey = true;
            }catch(Throwable e)
            {
                isKey = false;
            }
            
        }
        catch(Exception ne){
            isKey = false;
        }
        return isKey;
	}

    @Override
	public PublicKey getPublicKey(String public_key)
	{
        PublicKey key = null;
        try{
            byte[] hash = utils.fromHexString(public_key);
            X509EncodedKeySpec X509publicKey = new X509EncodedKeySpec(hash);
            KeyFactory kf = KeyFactory.getInstance("EC");
            key =  kf.generatePublic(X509publicKey);
        }
        catch(Exception e){
        }
        return key;
	}
}