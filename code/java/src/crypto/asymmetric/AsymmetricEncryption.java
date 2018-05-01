package net.chain0.resources.crypto.asymmetric;

import java.security.KeyPair;
import java.security.PublicKey;
import java.security.Signature;

public interface AsymmetricEncryption
{
	KeyPair createKeys();
	String createSignature(String privateKey, String hash);
	boolean verifySignature(String publicKey, String signature, String hash);
	Signature getSignature();
	boolean verifyKey(String public_key);
	PublicKey getPublicKey(String public_key);
}
