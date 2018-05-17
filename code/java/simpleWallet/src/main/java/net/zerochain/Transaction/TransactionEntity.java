package net.zerochain.Transaction;

import java.io.Serializable;
import java.sql.Date;
import java.sql.Timestamp;

import net.zerochain.resources.Utils;

public class TransactionEntity implements Serializable{

	
	private static final long serialVersionUID = 1L;
	
	private String clientid;

	private String data;

	private Timestamp timestamp;

	private String hash_msg;

	private String sign;
	
	public TransactionEntity (String clientid,String data, Timestamp timestamp,String hash_msg, String sign) {
		this.clientid = clientid;
		this.data = data;
		this.timestamp = timestamp;
		this.hash_msg = hash_msg;
		this.sign = sign;
	}
	
	public TransactionEntity() {
		this.clientid = "";
		this.data = "";
		this.timestamp = new Timestamp(1L);
		this.hash_msg = "";
		this.sign = "";
	}
	
	public String getClientID() {
		return clientid;
	}
	public void setClientID(String clientid) {
		this.clientid = clientid;
	}
	
	public String getData() {
		return data;
	}
	public void setData(String data) {
		this.data = data;
	}
	
	public Timestamp getTimestamp() {
		return timestamp;
	}
	public void setTimestamp(Timestamp timestamp) {
		this.timestamp = timestamp;
	}
	
	public String getHash_msg() {
		return hash_msg;
	}
	
	public void setHash_msg(String hash_msg) {
		this.hash_msg = hash_msg;
	}
	
	public String getSign() {
		return sign;
	}
	public void setSign(String sign) {
		this.sign = sign;
	}

	@Override
	public String toString()
	{
		return "TransactionEntity{" +
				" clientid='" + clientid + '\'' +
				", transaction_data='" + data + '\'' +
				", timestamp='" + Utils.timestampToString(timestamp) +'\'' +
				", hash='" + hash_msg + '\'' +
				", signature='" + sign + '\'' +
				'}'; 
	}

}