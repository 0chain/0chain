package net.zerochain.Block;

import java.io.Serializable;
import java.sql.Timestamp;
import java.text.SimpleDateFormat;
import java.util.Date;

import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.Id;
import javax.persistence.Table;

@Entity
@Table(name= "block")
public class BlockEntity implements Serializable {

	private static final long serialVersionUID = 1L;

	@Id
	@Column(name= "block_hash")
	private String block_hash;

	@Column(name="prev_block_hash")
	private String prev_block_hash;

	@Column(name = "block_signature")
	private String block_signature;

	@Column(name = "miner_id")
	private String miner_id;
	
	
	@Column(name = "timestamp")
	private Timestamp timestamp;

	@Column(name = "round")
	private int round;
	
    public static String timestampToString(Timestamp timestamp)
    {
        SimpleDateFormat dateFormat = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS");
        return dateFormat.format(timestamp);
    }
	public BlockEntity(String block_hash, String prev_block_hash, String block_signature, String miner_id,Timestamp timestamp, int round) {
		this.block_hash = block_hash;
		this.prev_block_hash = prev_block_hash;
		this.block_signature = block_signature;
		this.miner_id = miner_id;
		this.timestamp = timestamp;
		this.round = round;
	}

	public BlockEntity() {
		this.block_hash = "";
		this.prev_block_hash = "";
		this.block_signature = "";
		this.miner_id = "";
		this.timestamp = new Timestamp(1L);
		this.round= 0;
	}

	public String getBlock_hash() {
		return block_hash;
	}
	public void setBlock_hash(String block_hash) {
		this.block_hash = block_hash;
	}

	public String getPrev_block_hash() {
		return prev_block_hash;
	}
	public void setPrev_block_hash(String prev_block_hash) {
		this.prev_block_hash = prev_block_hash;
	}

	public String getBlock_signature() {
		return block_signature;
	}
	public void setBlock_signature(String block_signature) {
		this.block_signature= block_signature;
	}

	public String getMiner_id() {
		return miner_id;
	}
	public void setMiner_id(String miner_id) {
		this.miner_id = miner_id;
	}

	public Timestamp getTimestamp() {
		return timestamp;
	}
	public void setTimestamp(Timestamp timestamp) {
		this.timestamp = timestamp;
	}

	public int getRound() {
		return round;
	}
	public void setRound(int round) {
		this.round = round;
	}

	@Override
	public String toString() {
		return "{"+
				" \"block_hash\":\"" + block_hash + '\"' +
				", \"prev_block_hash\":\"" + prev_block_hash + '\"' +
				", \"block_signature\":\"" + block_signature + '\"' +
				", \"miner_id\":\"" + miner_id + '\"' +
				", \"timestamp\":\"" + timestampToString(timestamp) +'\"' +
				", \"round\":\"" + round + '\"' +
				'}';
	}


}
