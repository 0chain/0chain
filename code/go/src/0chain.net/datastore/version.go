package datastore

type VersionField struct {
	Version string `json:"version" msgpack:"_v"`
}
