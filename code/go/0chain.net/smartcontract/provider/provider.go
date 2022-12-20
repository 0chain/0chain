package provider

import (
	"0chain.net/core/datastore"
)

//go:generate msgp -io=false -tests=false -v

func GetKey(id string) datastore.Key {
	return "provider:" + id
}

type AbstractProvider interface {
}
