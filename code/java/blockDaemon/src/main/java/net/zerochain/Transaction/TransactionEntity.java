package net.zerochain.Transaction;

import java.io.Serializable;
import java.sql.Date;
import java.sql.Timestamp;

import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import javax.persistence.Id;
import javax.persistence.Table;
import javax.persistence.Temporal;
import javax.persistence.TemporalType;

import net.zerochain.resources.Utils;


@Entity
@Table(name = "transactions")
public class TransactionEntity implements Serializable{

	
	private static final long serialVersionUID = 1L;
	
	@Column(name = "clientid")
	private String clientid;
	
	@Column(name = "data")
	private String data;
	
	@Column(name = "timestamp")
	private Timestamp timestamp;
	
	@Id
	@Column(name = "hash_msg")
	private String hash_msg;
	
	@Column(name = "sign")
	private String sign;

	@Column(name = "status")
	private String status;
	
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

	public String getStatus()
	{
		return status;
	}

	public void setStatus(String status)
	{
		this.status = status;
	}

	@Override
	public String toString()
	{
		return "TransactionEntity{" +
				" clientid='" + clientid + '\'' +
				", data='" + data + '\'' +
				", timestamp='" + Utils.timestampToString(timestamp) +'\'' +
				", hash_msg='" + hash_msg + '\'' +
				", sign='" + sign + '\'' +
				'}'; 
	}

}
