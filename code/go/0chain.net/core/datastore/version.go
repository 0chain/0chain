package datastore

//go:generate msgp -io=false -tests=false -v
// swagger:model
type VersionField struct {
	// Version of the entity
	Version string `json:"version" msgpack:"_v"`
}
