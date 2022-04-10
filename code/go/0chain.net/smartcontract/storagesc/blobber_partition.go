package storagesc

//go:generate msgp -io=false -tests=false -unexported=true -v

type BlobberNode struct {
	ID  string `json:"id"`
	Url string `json:"url"`
}

func (bn *BlobberNode) GetID() string {
	return bn.ID
}
