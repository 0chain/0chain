package zcnsc_test

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"0chain.net/chaincore/currency"

	"0chain.net/core/encryption"

	cstate "0chain.net/chaincore/chain/state"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	logging.Logger = zap.NewNop()
}

func Test_ShouldSign(t *testing.T) {
	bytes, err := json.Marshal("sample string")
	require.NoError(t, err)

	signatureScheme := encryption.NewBLS0ChainScheme()
	err = signatureScheme.GenerateKeys()
	require.NoError(t, err)

	_, err = signatureScheme.Sign(hex.EncodeToString(bytes))
	require.NoError(t, err)
}

func Test_ShouldSignAndVerify(t *testing.T) {
	bytes, err := json.Marshal("sample string")
	require.NoError(t, err)

	signatureScheme := encryption.NewBLS0ChainScheme()
	err = signatureScheme.GenerateKeys()
	require.NoError(t, err)

	hash := hex.EncodeToString(bytes)
	sig, err := signatureScheme.Sign(hash)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	ok, err := signatureScheme.Verify(sig, hash)
	require.NoError(t, err)
	require.Equal(t, true, ok)
}

func Test_ShouldSignAndVerifyUsingPublicKey(t *testing.T) {
	bytes, err := json.Marshal("sample string")
	require.NoError(t, err)

	signatureScheme := encryption.NewBLS0ChainScheme()
	err = signatureScheme.GenerateKeys()
	require.NoError(t, err)

	hash := hex.EncodeToString(bytes)
	sig, err := signatureScheme.Sign(hash)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	pk := signatureScheme.GetPublicKey()
	signatureScheme = encryption.NewBLS0ChainScheme()
	err = signatureScheme.SetPublicKey(pk)
	require.NoError(t, err)

	ok, err := signatureScheme.Verify(sig, hash)
	require.NoError(t, err)
	require.Equal(t, ok, true)
}

//func Test_ShouldVerifySignature(t *testing.T) {
//	mp, err := CreateMintPayload(defaultClient)
//	require.NoError(t, err)
//
//	toSign := mp.GetStringToSign()
//	for _, sig := range mp.Signatures {
//		auth := authorizers[sig.ID]
//		ok, err := auth.Verify(sig.Signature, toSign)
//		require.NoError(t, err)
//		require.Equal(t, true, ok)
//	}
//}

func Test_ShouldSaveGlobalNode(t *testing.T) {
	_, _, err := createStateAndNodeAndAddNodeToState()
	require.NoError(t, err, "must Save the global node in state")
}

func Test_ShouldGetGlobalNode(t *testing.T) {
	balances, node, err := createStateAndNodeAndAddNodeToState()
	require.NoError(t, err, "must Save the global node in state")

	expected, _ := GetGlobalNode(balances)

	require.Equal(t, node.ID, expected.ID)
	require.Equal(t, node.MinBurnAmount, expected.MinBurnAmount)
}

func Test_GlobalNodeEncodeAndDecode(t *testing.T) {
	node := CreateSmartContractGlobalNode()
	node.BurnAddress = "11"
	node.MinMintAmount = 12
	node.MinBurnAmount = 13

	expected := CreateSmartContractGlobalNode()

	bytes := node.Encode()
	err := expected.Decode(bytes)

	require.NoError(t, err, "must Save the global node in state")

	expected.BurnAddress = "11"
	expected.MinMintAmount = 12
	expected.MinBurnAmount = 13
}

func Test_PublicKey(t *testing.T) {
	pk := AddAuthorizerPayload{}

	err := pk.Decode(nil)
	require.Error(t, err)

	var data []byte
	err = pk.Decode(data)
	require.Error(t, err)

	data = []byte("")
	err = pk.Decode(data)
	require.Error(t, err)

	pk.PublicKey = "public key"

	bytes, err := json.Marshal(pk)
	require.NoError(t, err)

	expected := AddAuthorizerPayload{}
	err = expected.Decode(bytes)
	require.NoError(t, err)
	require.Equal(t, expected.PublicKey, pk.PublicKey)
}

func Test_AuthorizerPartialUpSizeSerialization(t *testing.T) {
	type PartialState struct {
		ID     string            `json:"id"`
		Config *AuthorizerConfig `json:"config"`
	}

	target := &AuthorizerNode{}
	source := &PartialState{
		Config: &AuthorizerConfig{
			Fee: currency.Coin(222),
		},
	}

	bytes, err := json.Marshal(source)
	require.NoError(t, err)

	err = json.Unmarshal(bytes, target)
	require.NoError(t, err)

	require.Equal(t, currency.Coin(222), target.Config.Fee)
}

func Test_AuthorizerPartialDownSizeSerialization(t *testing.T) {
	type PartialState struct {
		ID     string            `json:"id"`
		Config *AuthorizerConfig `json:"config"`
	}

	source := &AuthorizerNode{
		Config: &AuthorizerConfig{
			Fee: currency.Coin(222),
		},
	}

	target := &PartialState{}
	err := json.Unmarshal(source.Encode(), target)

	require.NoError(t, err)
	require.Equal(t, currency.Coin(222), target.Config.Fee)
}

func Test_AuthorizerSettings_ShouldBeSerializable(t *testing.T) {
	source := &AuthorizerNode{
		Config: &AuthorizerConfig{
			Fee: currency.Coin(222),
		},
	}

	target := &AuthorizerNode{}
	err := target.Decode(source.Encode())
	require.NoError(t, err)
	require.Equal(t, currency.Coin(222), target.Config.Fee)
}

func Test_AuthorizerNodeSerialization(t *testing.T) {
	source := &AuthorizerNode{
		ID:        "aaa",
		PublicKey: "bbb",
		URL:       "ddd",
		Config: &AuthorizerConfig{
			Fee: 222,
		},
	}

	target := &AuthorizerNode{}

	err := target.Decode(source.Encode())
	require.NoError(t, err)
}

func Test_UpdateAuthorizerConfigTest(t *testing.T) {
	type AuthorizerConfigSource struct {
		Fee currency.Coin `json:"fee"`
	}

	type AuthorizerNodeSource struct {
		ID     string                  `json:"id"`
		Config *AuthorizerConfigSource `json:"config"`
	}

	source := &AuthorizerNodeSource{
		ID: "12345678",
		Config: &AuthorizerConfigSource{
			Fee: currency.Coin(999),
		},
	}
	target := &AuthorizerNode{}

	bytes, err := json.Marshal(source)
	require.NoError(t, err)

	err = target.Decode(bytes)
	require.NoError(t, err)

	err = target.Decode(bytes)
	require.NoError(t, err)

	require.Equal(t, "", target.URL)
	require.Equal(t, "", target.PublicKey)
	require.Equal(t, "12345678", target.ID)
	require.Equal(t, currency.Coin(999), target.Config.Fee)
}

func createStateAndNodeAndAddNodeToState() (cstate.StateContextI, *GlobalNode, error) {
	node := CreateSmartContractGlobalNode()
	node.MinBurnAmount = 111
	balances := MakeMockStateContext()
	err := node.Save(balances)
	return balances, node, err
}
