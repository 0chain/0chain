package provider

import "0chain.net/core/datastore"

func GetKey(id string) datastore.Key {
	return "provider:" + id
}
