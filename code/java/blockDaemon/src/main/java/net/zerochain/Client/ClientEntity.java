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
	@Column(name = "clientid")
	private String clientid;

	public ClientEntity(String public_key, String clientid) {
		this.public_key = public_key;
		this.clientid = clientid;
	}

	public ClientEntity() {
	}

	public String getPublic_key() {
		return public_key;
	}
	public void setPublic_key(String public_key) {
		this.public_key = public_key;
	}

	public String getClientID() {
		return clientid;
	}
	public void setClientID(String clientid) {
		this.clientid = clientid;
	}

	@Override
	public String toString() {
		return "MinerEntity{" +
				" public_key='" + public_key + '\'' +
				", clientid='" + clientid + '\'' +
				'}';
	}
}
