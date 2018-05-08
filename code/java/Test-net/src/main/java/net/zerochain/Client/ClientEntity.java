package net.zerochain.Client;

import javax.persistence.Column;
import javax.persistence.Entity;
import javax.persistence.Id;
import javax.persistence.Table;
import javax.persistence.GeneratedValue;
import javax.persistence.GenerationType;
import java.io.Serializable;

@Entity
@Table(name="clients")
public class ClientEntity implements Serializable {

	private static final long serialVersionUID = 1L;

	@Column(name = "public_key")
	private String public_key;

	@Id
	@Column(name = "hash_key")
	private String hash_key;

	public ClientEntity(String public_key, String hash_key) {
		this.public_key = public_key;
		this.hash_key = hash_key;
	}

	public ClientEntity() {
	}

	public String getPublic_key() {
		return public_key;
	}
	public void setPublic_key(String public_key) {
		this.public_key = public_key;
	}

	public String getHash_key() {
		return hash_key;
	}
	public void setHash_key(String hash_key) {
		this.hash_key = hash_key;
	}

	@Override
	public String toString() {
		return "MinerEntity{" +
				" public_key='" + public_key + '\'' +
				", hash_key='" + hash_key + '\'' +
				'}';
	}
}
