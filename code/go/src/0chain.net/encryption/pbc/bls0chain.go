package pbc

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"0chain.net/encryption"
	"github.com/Nik-U/pbc"
)

//BLS0ChainInitialSetup - initial setup to create parameters for the signatures
func BLS0ChainInitialSetup() (*pbc.Params, *pbc.Element) {
	params := pbc.GenerateA(160, 512)
	pairing := params.NewPairing()
	g := pairing.NewG2().Rand()
	return params, g
}

//BLS0ChainParams - parameters associated with the BLS0Chain signature scheme
type BLS0ChainParams struct {
	Params  string `json:"params"`
	SharedG []byte `json:"shared_g"`
}

//BLS0ChainSerialize signature parameters
func BLS0ChainSerialize(params *pbc.Params, g *pbc.Element, writer io.Writer) error {
	sigParams := &BLS0ChainParams{}
	sigParams.Params = params.String()
	sigParams.SharedG = g.Bytes()
	return json.NewEncoder(writer).Encode(sigParams)
}

//BLS0ChainDeserialize - deserialize the signature parameters
func BLS0ChainDeserialize(reader io.Reader) (*BLS0ChainParams, error) {
	sigParams := &BLS0ChainParams{}
	err := json.NewDecoder(reader).Decode(&sigParams)
	if err != nil {
		return nil, err
	}
	return sigParams, nil
}

//BLS0ChainSetup - given the params string and shared G from the initial setup, recreate the pairing and G
func BLS0ChainSetup(bls0ChainParams *BLS0ChainParams) (*pbc.Pairing, *pbc.Element) {
	params, err := pbc.NewParamsFromString(bls0ChainParams.Params)
	if err != nil {
		panic(err)
	}
	pairing := params.NewPairing()
	g := pairing.NewG2()
	g.SetBytes(bls0ChainParams.SharedG)
	return pairing, g
}

//BLS0ChainScheme - a signature scheme based on BLS0Chain
type BLS0ChainScheme struct {
	privateKey *pbc.Element
	publicKey  *pbc.Element
	pairing    *pbc.Pairing
	sharedG    *pbc.Element
}

//NewBLS0ChainScheme - given the bls0chain params, create the associated signature scheme object
func NewBLS0ChainScheme(bls0chainParams *BLS0ChainParams) *BLS0ChainScheme {
	b0 := &BLS0ChainScheme{}
	pairing, g := BLS0ChainSetup(bls0chainParams)
	b0.pairing = pairing
	b0.sharedG = g
	return b0
}

//GenerateKeys - implement interface
func (b0 *BLS0ChainScheme) GenerateKeys() error {
	b0.privateKey = b0.pairing.NewZr().Rand()
	b0.publicKey = b0.pairing.NewG2().PowZn(b0.sharedG, b0.privateKey)
	return nil
}

//ReadKeys - implement interface
func (b0 *BLS0ChainScheme) ReadKeys(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	result := scanner.Scan()
	if result == false {
		return encryption.ErrKeyRead
	}
	publicKey := scanner.Text()
	b0.SetPublicKey(publicKey)
	result = scanner.Scan()
	if result == false {
		return encryption.ErrKeyRead
	}
	privateKey := scanner.Text()
	privateKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return err
	}
	b0.privateKey = b0.pairing.NewZr().SetBytes(privateKeyBytes)
	return nil
}

//WriteKeys - implement interface
func (b0 *BLS0ChainScheme) WriteKeys(writer io.Writer) error {
	publicKey := hex.EncodeToString(b0.publicKey.Bytes())
	privateKey := hex.EncodeToString(b0.privateKey.Bytes())
	_, err := fmt.Fprintf(writer, "%v\n%v\n", publicKey, privateKey)
	return err
}

//SetPublicKey - implement interface
func (b0 *BLS0ChainScheme) SetPublicKey(publicKey string) error {
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return err
	}
	b0.publicKey = b0.pairing.NewG2().SetBytes(publicKeyBytes)
	return nil
}

//GetPublicKey - implement interface
func (b0 *BLS0ChainScheme) GetPublicKey() string {
	return hex.EncodeToString(b0.publicKey.Bytes())
}

//Sign - implement interface
func (b0 *BLS0ChainScheme) Sign(hash interface{}) (string, error) {
	rawHash, err := encryption.GetRawHash(hash)
	if err != nil {
		return "", err
	}
	h := b0.pairing.NewG1().SetFromHash(rawHash)
	signature := b0.pairing.NewG2().PowZn(h, b0.privateKey)
	return hex.EncodeToString(signature.Bytes()), nil
}

//Verify - implement interface
func (b0 *BLS0ChainScheme) Verify(signature string, hash string) (bool, error) {
	s1, err := b0.PairMessageHash(hash)
	if err != nil {
		return false, err
	}
	rawSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false, err
	}

	sig := b0.pairing.NewG2().SetBytes([]byte(rawSignature))
	s2 := b0.pairing.NewGT().Pair(sig, b0.sharedG)
	return s1.Equals(s2), nil
}

//PairMessageHash - Pair a given message hash
func (b0 *BLS0ChainScheme) PairMessageHash(hash string) (*pbc.Element, error) {
	rawHash, err := hex.DecodeString(hash)
	if err != nil {
		return nil, err
	}
	h := b0.pairing.NewG1().SetFromHash(rawHash)
	s1 := b0.pairing.NewGT().Pair(h, b0.publicKey)
	return s1, nil
}
