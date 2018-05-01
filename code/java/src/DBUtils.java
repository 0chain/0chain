package net.chain0.resources;

import java.util.ArrayList;
import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.Statement;
import java.sql.Timestamp;
import java.sql.SQLException;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.Calendar;

import java.io.PrintWriter;

public class DBUtils
{
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

    public static boolean blockchainStarted(Connection conn)
    {
        boolean started = false;
        if(conn != null)
        {
            Statement stmt = null;
            try
            {
                String query = "Select count(*) from block;";
                stmt = conn.createStatement();
                ResultSet rs = stmt.executeQuery(query);
                if(rs.next())
                {
                    started = true;
                }
            } catch(Exception e)
            {
                started = false;
            }
        }
        return started;
    }

    public static boolean startBlockchain(Connection conn)
    {
        boolean start = false;
        if(conn != null && !blockchainStarted(conn))
        {
            Statement stmt = null;
            try
            {
                String query = "INSERT into block (block_hash, prev_block_hash, block_signature, miner_id, timestamp) VALUES ('0001','0000','aaa','ddd','"+utils.getTimestampAsString()+"');";
                stmt = conn.createStatement();
                int oid = stmt.executeUpdate(query);
                if(oid == 1)
                {
                    start = true;
                }
            } catch(Exception e)
            {
                start = false;
            }
        }
        else
        {
            start = blockchainStarted(conn);
        }
        return start;
    }

    public static String getCurrentBlockHash(Connection conn)
    {
        String currentBlockHash = "";
        if(conn != null)
        {
            Statement stmt = null;
            try
            {
                String query = "SELECT * from block order by timestamp desc limit 1;";
                stmt = conn.createStatement();
                ResultSet rs = stmt.executeQuery(query);
                if(rs.next())
                {
                    currentBlockHash = rs.getString("block_hash");
                }
            } catch(Exception e)
            {
            }
        }
        return currentBlockHash;
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
}