package net.zerochain.Client;

import java.io.Serializable;

public class ClientEntity implements Serializable {

	private static final long serialVersionUID = 1L;

	private String public_key;

	private String clientid;

	public ClientEntity(String public_key, String clientid) {
		this.public_key = public_key;
		this.clientid = clientid;
	}

	public ClientEntity() {
		this.public_key = "";
		this.clientid = "";
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
		return "{" +
				"\"public_key\":\"" + public_key + '\"' +
				",\"clientid\":\"" + clientid + '\"' +
				'}';
	}
}
