package benchmark

//go:generate msgp -io=false -tests=false -unexported=true -v

type BenchDataMpt struct {
	Clients             []string `json:"clients"`
	PublicKeys          []string `json:"publicKeys"`
	PrivateKeys         []string `json:"privateKeys"`
	Sharders            []string `json:"sharders"`
	AuthorizerId        string   `json:"authorizerId"`
	AuthorizerPublicKey string   `json:"authorizerPublicKey"`
	Owner               string   `json:"owner"`
	OwnerPublicKey      string   `json:"ownerPublicKey"`
	OwnerPrivateKey     string   `json:"ownerPrivateKey"`
}
