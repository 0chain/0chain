package chain

import (
	"testing"
)

func TestVersionsEntity_Get(t *testing.T) {
	vs := VersionsEntity{}
	vs.Version = "1.0.0"
	vs.Sign = "abcdefe"
	vs.Versions = map[string]string{
		"sc_version":    "1.0.0",
		"proto_version": "1.0.0",
	}

	vs.Hash()
	//d, err := json.MarshalIndent(&vs, "", "\t")
	//require.NoError(t, err)
	//fmt.Println(string(d))
}
