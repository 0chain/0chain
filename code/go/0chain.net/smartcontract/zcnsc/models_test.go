package zcnsc_test

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"0chain.net/chaincore/state"

	"0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/logging"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	chain.ServerChain = new(chain.Chain)
	chain.ServerChain.Config = chain.NewConfigImpl(&chain.ConfigData{ClientSignatureScheme: "bls0chain"})

	logging.Logger = zap.NewNop()
}

func Test_ShouldSign(t *testing.T) {
	bytes, err := json.Marshal("sample string")
	require.NoError(t, err)

	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	err = signatureScheme.GenerateKeys()
	require.NoError(t, err)

	_, err = signatureScheme.Sign(hex.EncodeToString(bytes))
	require.NoError(t, err)
}

func Test_ShouldSignAndVerify(t *testing.T) {
	bytes, err := json.Marshal("sample string")
	require.NoError(t, err)

	signatureScheme := chain.GetServerChain().GetSignatureScheme()
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

	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	err = signatureScheme.GenerateKeys()
	require.NoError(t, err)

	hash := hex.EncodeToString(bytes)
	sig, err := signatureScheme.Sign(hash)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	pk := signatureScheme.GetPublicKey()
	signatureScheme = chain.GetServerChain().GetSignatureScheme()
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
	pk := AuthorizerParameter{}

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

	expected := AuthorizerParameter{}
	err = expected.Decode(bytes)
	require.NoError(t, err)
	require.Equal(t, expected.PublicKey, pk.PublicKey)
}

func Test_ZcnLockingPool_ShouldBeSerializable(t *testing.T) {
	pool := &tokenpool.ZcnLockingPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      "id",
				Balance: 100,
			},
		},
		TokenLockInterface: &TokenLock{
			StartTime: 0,
			Duration:  0,
			Owner:     "id",
		},
	}

	target := &tokenpool.ZcnLockingPool{}

	err := target.Decode(pool.Encode(), &TokenLock{})
	require.NoError(t, err)
	require.Equal(t, int(target.Balance), 100)
}

func Test_AuthorizerPartialUpSizeSerialization(t *testing.T) {
	type PartialState struct {
		ID     string            `json:"id"`
		Config *AuthorizerConfig `json:"config"`
	}

	target := &AuthorizerNode{}
	source := &PartialState{
		Config: &AuthorizerConfig{
			Fee: state.Balance(222),
		},
	}

	bytes, err := json.Marshal(source)
	require.NoError(t, err)

	err = json.Unmarshal(bytes, target)
	require.NoError(t, err)

	require.Equal(t, state.Balance(222), target.Config.Fee)
}

func Test_AuthorizerPartialDownSizeSerialization(t *testing.T) {
	type PartialState struct {
		ID     string            `json:"id"`
		Config *AuthorizerConfig `json:"config"`
	}

	source := &AuthorizerNode{
		Config: &AuthorizerConfig{
			Fee: state.Balance(222),
		},
	}

	target := &PartialState{}
	err := json.Unmarshal(source.Encode(), target)

	require.NoError(t, err)
	require.Equal(t, state.Balance(222), target.Config.Fee)
}

func Test_AuthorizerSettings_ShouldBeSerializable(t *testing.T) {
	source := &AuthorizerNode{
		Config: &AuthorizerConfig{
			Fee: state.Balance(222),
		},
	}

	target := &AuthorizerNode{}
	err := target.Decode(source.Encode())
	require.NoError(t, err)
	require.NotNil(t, target.Staking)
	require.Equal(t, state.Balance(222), target.Config.Fee)
}

func Test_AuthorizerNode_ShouldBeSerializableWithTokenLock(t *testing.T) {
	// Create authorizer node
	tr := CreateDefaultTransactionToZcnsc()
	node := NewAuthorizer(tr.ClientID, tr.PublicKey, "https://localhost:9876")
	_, _, _ = node.Staking.DigPool(tr.Hash, tr)
	node.Staking.ID = "11"

	// Deserialize it into new instance
	target := &AuthorizerNode{}

	err := target.Decode(node.Encode())
	require.NoError(t, err)
	require.Equal(t, target.Staking.ID, "11")
	require.Equal(t, int64(target.Staking.Balance), tr.Value)
}

func Test_AuthorizerNodeSerialization(t *testing.T) {
	source := &AuthorizerNode{
		ID:        "aaa",
		PublicKey: "bbb",
		Staking: &tokenpool.ZcnLockingPool{
			ZcnPool: tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{
					ID:      "ccc",
					Balance: 111,
				},
			},
			TokenLockInterface: nil,
		},
		URL: "ddd",
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
		Fee state.Balance `json:"fee"`
	}

	type AuthorizerNodeSource struct {
		ID     string                  `json:"id"`
		Config *AuthorizerConfigSource `json:"config"`
	}

	source := &AuthorizerNodeSource{
		ID: "12345678",
		Config: &AuthorizerConfigSource{
			Fee: state.Balance(999),
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
	require.Equal(t, state.Balance(999), target.Config.Fee)
}

func createStateAndNodeAndAddNodeToState() (cstate.StateContextI, *GlobalNode, error) {
	node := CreateSmartContractGlobalNode()
	node.MinBurnAmount = 111
	balances := MakeMockStateContext()
	err := node.Save(balances)
	return balances, node, err
}
