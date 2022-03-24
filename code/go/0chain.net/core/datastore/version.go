package datastore

//go:generate msgp -io=false -tests=false -v
type VersionField struct {
	Version string `json:"version" msgpack:"_v"`
}
