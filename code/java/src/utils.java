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
import java.nio.ByteBuffer;
import java.security.NoSuchProviderException;
import java.security.spec.InvalidKeySpecException;
import java.security.spec.*;
import java.security.*;
import java.security.Signature;

import java.util.ArrayList;
import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.Statement;
import java.sql.Timestamp;
import java.sql.SQLException;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.Calendar;

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

    public static boolean verifyHash(byte[] thingToHash, String hash)
    {
    	return createHash(thingToHash).equals(hash);
    }

    public static boolean verifyHash(String thingToHash, String hash)
    {
    	return verifyHash(fromHexString(thingToHash),hash);
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