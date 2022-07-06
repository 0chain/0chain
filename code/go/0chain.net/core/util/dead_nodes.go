package util

//go:generate msgp -v -io=false -tests=false -unexported=true

type deadNodes struct {
	Nodes map[string]int64 `json:"nodes"`
}

func (d *deadNodes) decode(data []byte) error {
	_, err := d.UnmarshalMsg(data)
	return err
}

func (d *deadNodes) encode() ([]byte, error) {
	return d.MarshalMsg(nil)
}
