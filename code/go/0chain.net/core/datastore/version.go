package datastore

//var (
//	ErrPreviousProtoDecoderNotFound = errors.New("no previous protocol decoder available")
//)

type VersionField struct {
	Version string `json:"version" msgpack:"_v"`
}

// GetVersion returns the version
func (v *VersionField) GetVersion() string {
	return v.Version
}

func (v *VersionField) SetVersion(version string) {
	v.Version = version
}

// NoProtocolChange is a placeholder for protocol messages
// that indicates there's no protocol changes yet, but
// need to meet the Entity interface
type NoProtocolChange struct{}

func (n *NoProtocolChange) ToPreviousVersion(origin Entity) Entity {
	return origin
}

//func (n *NoProtocolChange) PreviousProtoDecode(data []byte) (Entity, error) {
//	return nil, ErrPreviousProtoDecoderNotFound
//}

// NotProtocol is a placeholder that indicates the Entity
// is not a protocol message while it still needs to meet the
// Entity interface
type NotProtocol struct {
	NoProtocolChange
}

func (n *NotProtocol) GetVersion() string {
	return ""
}

func (n *NotProtocol) SetVersion(_ string) {
}
