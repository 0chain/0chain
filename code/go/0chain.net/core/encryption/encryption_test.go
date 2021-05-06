package encryption

import (
	"testing"
)

var data = "0chain.net rocks"
var expectedHash = "6cb51770083ba34e046bc6c953f9f05b64e16a0956d4e496758b97c9cf5687d5"

func TestHash(t *testing.T) {
	if Hash(data) != expectedHash {
		t.Fatal("invalid hash")
	}
}
