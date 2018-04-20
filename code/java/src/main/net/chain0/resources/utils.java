package net.chain0.resources;

import org.bouncycastle.jcajce.provider.digest.SHA3.DigestSHA3;
import org.bouncycastle.jcajce.provider.digest.SHA3.Digest256;
import org.bouncycastle.jce.provider.BouncyCastleProvider;
import org.bouncycastle.pqc.math.linearalgebra.ByteUtils;
import java.security.KeyFactory;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.security.KeyPair;
import java.security.PublicKey;
import java.security.PrivateKey;
import java.security.spec.X509EncodedKeySpec;
import java.security.spec.PKCS8EncodedKeySpec;
import java.security.Security;
import java.util.ArrayList;
import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.Statement;
import java.sql.Timestamp;
import java.sql.SQLException;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.Calendar;
import java.security.Signature;
import java.nio.ByteBuffer;
import java.security.NoSuchProviderException;
import java.security.spec.InvalidKeySpecException;
import java.security.spec.*;
import java.security.*;
import java.lang.Throwable;
import java.io.PrintWriter;

public class utils
{
	public static String toHexString(byte[] bytes)
	{
		return ByteUtils.toHexString(bytes);
	}

	public static byte[] fromHexString(String hash)
	{
		return ByteUtils.fromHexString(hash);
	}

    public static String stringToHexString(String data)
    {
        return toHexString(data.getBytes());
    }

    public static String hexStringToString(String hexString)
    {
        return new String(fromHexString(hexString));
    }

    public static byte[] doubleToByteArray(double value) 
    {
        byte[] bytes = new byte[8];
        ByteBuffer.wrap(bytes).putDouble(value);
        return bytes;
    }

    public static PublicKey getPublicKey(String public_key)
    {
        PublicKey key = null;
        Security.addProvider(new BouncyCastleProvider());
        try{
            byte[] hash = fromHexString(public_key);
            X509EncodedKeySpec X509publicKey = new X509EncodedKeySpec(hash);
            KeyFactory kf = KeyFactory.getInstance("EC");
            key =  kf.generatePublic(X509publicKey);
        }
        catch(Exception e){
        }
        return key;
    }

	public static PublicKey getPublicKey(String public_key, PrintWriter writer)
	{
        PublicKey key = null;
        writer.println("in get key");
		Security.addProvider(new BouncyCastleProvider());
        writer.println("Got provider");
    	try{
    		byte[] hash = fromHexString(public_key);
    		X509EncodedKeySpec X509publicKey = new X509EncodedKeySpec(hash);
    		KeyFactory kf = KeyFactory.getInstance("EC");
    		key =  kf.generatePublic(X509publicKey);
            writer.println("here");
    	}
    	catch(Exception e){
             writer.println(e.getClass().getName()+": "+ e.getMessage() );
    	}
    	return key;
	}

	public static boolean verifyKey(String public_key)
    {
        boolean isKey = false;
        try{
            byte[] hash = fromHexString(public_key);
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

    public static boolean verifyKey(String public_key, PrintWriter writer)
    {
        boolean isKey = false;
        writer.println("In HERE");
        try{
            byte[] hash = fromHexString(public_key);
            writer.println("to byte array");
            X509EncodedKeySpec X509publicKey = new X509EncodedKeySpec(hash);
            writer.println("x509");
            KeyFactory kf = KeyFactory.getInstance("EC");
            writer.println("key factory");
            try
            {
                kf.generatePublic(X509publicKey);
                isKey = true;
            }catch(Throwable e)
            {
                isKey = false;
                writer.println(e.getClass().getName()+": "+ e.getMessage() );
            }
            
            writer.println("generate");
            
        }
        catch(Exception ne){
            isKey = false;
            writer.println(ne.getClass().getName()+": "+ ne.getMessage() );
        }
        return isKey;
    }

    public static String createHash(ArrayList<byte[]> thingsToHash)
    {
    	DigestSHA3 sha3 = new Digest256();
    	int i, size = thingsToHash.size();
    	for(i = 0; i < size; i++)
    	{
    		sha3.update(thingsToHash.get(i));
    	}
    	return toHexString(sha3.digest());
    }

    public static String createHash(byte[] thingToHash)
    {
    	DigestSHA3 sha3 = new Digest256();
    	sha3.update(thingToHash);
    	return toHexString(sha3.digest());
    }

    public static String createHash(String thingToHash)
    {
    	return createHash(fromHexString(thingToHash));
    }

    public static boolean verifyHash(ArrayList<byte[]> thingsToHash, String hash)
    {
    	return createHash(thingsToHash).equals(hash);
    }

    public static boolean verifyHash(byte[] thingToHash, String hash)
    {
    	return createHash(thingToHash).equals(hash);
    }

    public static boolean verifyHash(String thingToHash, String hash)
    {
    	return verifyHash(fromHexString(thingToHash),hash);
    }

    public static boolean alreadyClient(String hash, Connection conn, PrintWriter writer)
    {
    	boolean isClient = false;
        if(conn != null)
        {

            Statement stmt = null;
            try{
                String query = "select * from clients where hash_key = '"+hash+"';";
                stmt = conn.createStatement();
                ResultSet rs = stmt.executeQuery(query);  
                if(rs.next() && rs.getString("hash_key").equals(hash))
                {
                	isClient = true;
                }  
                stmt.close();          
            } catch ( Exception e ) {
                writer.println(e.getClass().getName()+": "+ e.getMessage());
            }
        }
        return isClient;
    }

    public static boolean alreadyClient(String hash, Connection conn)
    {
        boolean isClient = false;
        if(conn != null)
        {

            Statement stmt = null;
            try{
                String query = "select * from clients where hash_key = '"+hash+"';";
                stmt = conn.createStatement();
                ResultSet rs = stmt.executeQuery(query);  
                if(rs.next() && rs.getString("hash_key").equals(hash))
                {
                    isClient = true;
                }  
                stmt.close();          
            } catch ( Exception e ) {
            }
        }
        return isClient;
    }

    public static boolean freshTransaction(String hash, Connection conn)
    {
        boolean freshTransaction = false;
        if(conn != null)
        {
            Statement stmt = null;
            try
            {
                String query = "Select * from \"transaction\" where hash_msg = '"+hash+"';";
                stmt = conn.createStatement();
                ResultSet rs = stmt.executeQuery(query);
                if(!rs.next())
                {
                    freshTransaction = true;
                }
            } catch(Exception e)
            {
                freshTransaction = false;
            }
        }
        return freshTransaction;
    }

    public static String getPublicKeyFromHash(String hashKey, Connection conn)
    {
        String publicKey = "";
        if(conn != null)
        {
            Statement stmt = null;
            try{
                String query = "select public_key from clients where hash_key = '"+hashKey+"';";
                stmt = conn.createStatement();
                ResultSet rs = stmt.executeQuery(query); 
                if(rs.next())
                {
                    publicKey = rs.getString("public_key");
                }  
                stmt.close();          
            } catch ( Exception e ) {

            }
        }
        return publicKey;
    }

    public static String createPublicKey()
    {
        String key = "";
        try {
            KeyPairGenerator keyGen = KeyPairGenerator.getInstance("EC");
            KeyPair keypair = keyGen.generateKeyPair();
            //System.out.println(toHexString(keypair.getPrivate().getEncoded()));
            key = ByteUtils.toHexString(keypair.getPublic().getEncoded());
        } catch (Exception e) {
            // TODO Auto-generated catch block
        }
        return key;
    }

    public static KeyPair createKeys()
    {
        KeyPair key = null;
        try {
            KeyPairGenerator keyGen = KeyPairGenerator.getInstance("EC");
            key = keyGen.generateKeyPair();
            //System.out.println(toHexString(keypair.getPrivate().getEncoded()));
        } catch (Exception e) {
            // TODO Auto-generated catch block
        }
        return key;
    }

    public static String signHash(String privateKey, String hash)
    {
        String signature = "";
        try{
            Signature sign = getECDSA();
            KeyFactory kf = KeyFactory.getInstance("EC");
            sign.initSign(kf.generatePrivate(new PKCS8EncodedKeySpec(fromHexString(privateKey))));
            sign.update(fromHexString(hash));
            signature = toHexString(sign.sign());
        }catch(Exception e)
        {
        }
        return signature;
    }


    public static boolean verifySignature(String publicKey, String signature, String hash)
    {
        boolean signedCorrectly = false;
        PublicKey key = null;
        try
        {
            key = getPublicKey(publicKey);
        } catch(Throwable e)
        {

        }

        Signature sign = getECDSA();
        if(key != null)
        {
            try
            {
                sign.initVerify(key);
                sign.update(fromHexString(hash));
                signedCorrectly = sign.verify(fromHexString(signature));
            }catch(Exception e)
            {
                signedCorrectly = false;
            }
        }
        return signedCorrectly;
    }

    public static Signature getECDSA()
    {
        Signature ecdsa = null;
        try {
            ecdsa = Signature.getInstance("SHA3-256WITHECDSA", new BouncyCastleProvider());
        } catch (Exception e) {

        } 
        return ecdsa;
    }

    public static boolean inTime(Timestamp minerTime, Timestamp timestamp)
    {
        boolean inTime = ((minerTime.getTime() - timestamp.getTime()) >= 0) && ((minerTime.getTime() - timestamp.getTime()) < 5000);
        return inTime;
    }

    public static Timestamp stringToTimestamp(String timestamp)
    {
        SimpleDateFormat dateFormat = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS");
        long justInCase = 0;
        Timestamp timeStamp = null;
        try
        {
            Date parsedTimestamp = dateFormat.parse(timestamp);
            timeStamp = new Timestamp(parsedTimestamp.getTime());
        }
        catch(Exception e)
        { 
            timeStamp = new Timestamp(justInCase);
        }
        return timeStamp;
    }

    public static Timestamp getTimestamp()
    {
        Calendar c = Calendar.getInstance();
        return new Timestamp(c.getTime().getTime());
    }

    public static String timestampToString(Timestamp timestamp)
    {
        SimpleDateFormat dateFormat = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS");
        return dateFormat.format(timestamp);
    }

    public static String getTimestampAsString()
    {
        SimpleDateFormat dateFormat = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS");
        return dateFormat.format(getTimestamp());
    }

}