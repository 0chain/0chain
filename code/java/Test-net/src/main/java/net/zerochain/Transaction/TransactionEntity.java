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
@Table(name = "transaction")
public class TransactionEntity implements Serializable{

	
	private static final long serialVersionUID = 1L;
	
	@Column(name = "client_id")
	private String client_id;
	
	@Column(name = "data")
	private String data;
	
	//@Temporal(TemporalType.TIMESTAMP)
	//@Column(name = "timestamp", nullable = false)
	@Column(name = "timestamp")
	private Timestamp timestamp;
	
	@Id
	@Column(name = "hash_msg")
	private String hash_msg;
	
	@Column(name = "sign")
	private String sign;
	
	public TransactionEntity (String client_id,String data, Timestamp timestamp,String hash_msg, String sign) {
		this.client_id = client_id;
		this.data = data;
		this.timestamp = timestamp;
		this.hash_msg = hash_msg;
		this.sign = sign;
	}
	
	public TransactionEntity() {
		this.client_id = "";
		this.data = "";
		this.timestamp = new Timestamp(1L);
		this.hash_msg = "";
		this.sign = "";
	}
	
	public String getClient_id() {
		return client_id;
	}
	public void setClient_id(String client_id) {
		this.client_id = client_id;
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
				" client_id='" + client_id + '\'' +
				", data='" + data + '\'' +
				", timestamp='" + Utils.timestampToString(timestamp) +'\'' +
				", hash_msg='" + hash_msg + '\'' +
				", sign='" + sign + '\'' +
				'}'; 
	}

}
