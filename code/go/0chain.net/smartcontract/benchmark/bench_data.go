package benchmark

//go:generate msgp -io=false -tests=false -unexported=true -v

type BenchDataMpt struct {
	Clients         []string `json:"clients"`
	PublicKeys      []string `json:"publicKeys"`
	PrivateKeys     []string `json:"privateKeys"`
	Sharders        []string `json:"sharders"`
	Owner           string   `json:"owner"`
	OwnerPublicKey  string   `json:"ownerPublicKey"`
	OwnerPrivateKey string   `json:"ownerPrivateKey"`
}
