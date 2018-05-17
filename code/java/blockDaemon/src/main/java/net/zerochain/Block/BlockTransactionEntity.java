package net.zerochain.Block;

import java.io.Serializable;

import javax.persistence.Column;
import javax.persistence.MapKeyColumn;
import javax.persistence.Entity;
import javax.persistence.Id;
import javax.persistence.IdClass;
import javax.persistence.Table;

@Entity
@IdClass(BlockTransactionEntity.class)
@Table(name= "blocktransactions")
public class BlockTransactionEntity implements Serializable {
	private static final long serialVersionUID = 1L;

	@Id
	@Column(name="block_hash")
	private String block_hash;

	@Id
	@Column(name="hash_msg")
	private String hash_msg;
	
	public BlockTransactionEntity()
	{
		this.block_hash = "";
		this.hash_msg = "";
	}

	public BlockTransactionEntity(String block_hash, String hash_msg)
	{
		this.block_hash = block_hash;
		this.hash_msg = hash_msg;
	}

	public String getBlock_hash()
	{
		return block_hash;
	}

	public String getHash_msg()
	{
		return hash_msg;
	}

	public void setBlock_hash(String block_hash)
	{
		this.block_hash = block_hash;
	}

	public void setHash_msg(String hash_msg)
	{
		this.hash_msg = hash_msg;
	}

	@Override
	public int hashCode()
	{
		String s = block_hash+hash_msg;
		return s.hashCode();
	}

	@Override
	public boolean equals(Object obj)
	{
		boolean equals = false;
		if (this == obj)
		{
			equals = true;
		}
		else if (obj == null)
		{
			equals = false;
		}
		else if (getClass() != obj.getClass())
		{
			equals = false;
		}
		else
		{
			BlockTransactionEntity other = (BlockTransactionEntity) obj;
			equals = hash_msg.equals(other.getHash_msg()) && block_hash.equals(other.getBlock_hash());
		}

		return equals;
	}

}
