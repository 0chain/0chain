package util

//go:generate msgp -v -io=false -tests=false -unexported=true

type deadNodes struct {
	Nodes map[string]bool `json:"n"` // value as bool type to pass msgp build error
}

func (d *deadNodes) decode(data []byte) error {
	_, err := d.UnmarshalMsg(data)
	return err
}

func (d *deadNodes) encode() ([]byte, error) {
	return d.MarshalMsg(nil)
}
