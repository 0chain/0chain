package encryption

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/herumi/bls/ffi/go/bls"
)

var GenG2 *bls.G2

func init() {
	err := bls.Init(bls.CurveFp254BNb)
	if err != nil {
		panic(err)
	}
	GenG2 = &bls.G2{}
	/* The following string is obtained by serializing the generator of G2 using temporary go binding as follows
		func (pub1 *PublicKey) GenG2() (pub2 *PublicKey) {
	        pub2 = new(PublicKey)
	        C.blsGetGeneratorOfG2(pub2.getPointer())
	        return pub2
	} */
	bytes, err := hex.DecodeString("28b1ce2dbb7eccc8ba6b0d29615ac81e33be4d5909602ac35d2cac774eb4cc119a0deec914a95ffcd4cdbe685608602e7f82de7651a2e95ba0c4dabb144a200f")
	if err != nil {
		panic(err)
	}
	GenG2.Deserialize(bytes)
}

//BLS0ChainScheme - a signature scheme for BLS0Chain Signature
type BLS0ChainScheme struct {
	privateKey []byte
	publicKey  []byte
}

//NewBLS0ChainScheme - create a BLS0ChainScheme object
func NewBLS0ChainScheme() *BLS0ChainScheme {
	return &BLS0ChainScheme{}
}

//GenerateKeys - implement interface
func (b0 *BLS0ChainScheme) GenerateKeys() error {
	var skey bls.SecretKey
	skey.SetByCSPRNG()
	b0.privateKey = skey.GetLittleEndian()
	b0.publicKey = skey.GetPublicKey().Serialize()
	return nil
}

//ReadKeys - implement interface
func (b0 *BLS0ChainScheme) ReadKeys(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	result := scanner.Scan()
	if result == false {
		return ErrKeyRead
	}
	publicKey := scanner.Text()
	b0.SetPublicKey(publicKey)
	result = scanner.Scan()
	if result == false {
		return ErrKeyRead
	}
	privateKey := scanner.Text()
	privateKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		return err
	}
	b0.privateKey = privateKeyBytes
	return nil
}

//WriteKeys - implement interface
func (b0 *BLS0ChainScheme) WriteKeys(writer io.Writer) error {
	publicKey := hex.EncodeToString(b0.publicKey)
	privateKey := hex.EncodeToString(b0.privateKey)
	_, err := fmt.Fprintf(writer, "%v\n%v\n", publicKey, privateKey)
	return err
}

//SetPublicKey - implement interface
func (b0 *BLS0ChainScheme) SetPublicKey(publicKey string) error {
	if len(b0.privateKey) > 0 {
		return errors.New("cannot set public key when there is a private key")
	}
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return err
	}
	b0.publicKey = publicKeyBytes
	return nil
}

//GetPublicKey - implement interface
func (b0 *BLS0ChainScheme) GetPublicKey() string {
	return hex.EncodeToString(b0.publicKey)
}

//Sign - implement interface
func (b0 *BLS0ChainScheme) Sign(hash interface{}) (string, error) {
	var sk bls.SecretKey
	sk.SetLittleEndian(b0.privateKey)
	rawHash, err := GetRawHash(hash)
	if err != nil {
		return "", err
	}
	sig := sk.Sign(string(rawHash))
	return sig.SerializeToHexStr(), nil
}

//Verify - implement interface
func (b0 *BLS0ChainScheme) Verify(signature string, hash string) (bool, error) {
	pk, err := b0.getPublicKey()
	if err != nil {
		return false, err
	}
	sign, err := b0.GetSignature(signature)
	if err != nil {
		return false, err
	}
	rawHash, err := hex.DecodeString(hash)
	if err != nil {
		return false, err
	}
	return sign.Verify(pk, string(rawHash)), nil
}

//GetSignature - given a string return the signature object
func (b0 *BLS0ChainScheme) GetSignature(signature string) (*bls.Sign, error) {
	if signature == "" {
		return nil, errors.New("empty signature")
	}
	var sign bls.Sign
	err := sign.DeserializeHexStr(signature)
	if err != nil {
		return nil, err
	}
	return &sign, nil
}

func (b0 *BLS0ChainScheme) getPublicKey() (*bls.PublicKey, error) {
	var pk = &bls.PublicKey{}
	err := pk.Deserialize(b0.publicKey)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

//PairMessageHash - Pair a given message hash
func (b0 *BLS0ChainScheme) PairMessageHash(hash string) (*bls.GT, error) {
	g2 := &bls.G2{}
	err := g2.Deserialize(b0.publicKey)
	if err != nil {
		return nil, err
	}
	var g1 = &bls.G1{}
	rawHash, err := hex.DecodeString(hash)
	if err != nil {
		return nil, err
	}
	g1.HashAndMapTo(rawHash)
	var gt = &bls.GT{}
	bls.Pairing(gt, g1, g2)
	return gt, nil
}

//GenerateSplitKeys - implement interface
func (b0 *BLS0ChainScheme) GenerateSplitKeys(numSplits int) ([]SignatureScheme, error) {
	var primarySk bls.Fr
	primarySk.SetLittleEndian(b0.privateKey)

	splitKeys := make([]SignatureScheme, numSplits)
	var sk bls.SecretKey

	/*key := NewBLS0ChainScheme()
	key.GenerateKeys()
	splitKeys[0] = key
	sk.SetLittleEndian(key.privateKey)
	*/

	//Generate all but one split keys and add the secret keys
	for i := 0; i < numSplits-1; i++ {
		key := NewBLS0ChainScheme()
		key.GenerateKeys()
		splitKeys[i] = key
		var sk2 bls.SecretKey
		sk2.SetLittleEndian(key.privateKey)
		sk.Add(&sk2)
	}
	var aggregateSk bls.Fr
	aggregateSk.SetLittleEndian(sk.GetLittleEndian())

	//Subtract the aggregated private key from the primary private key to derive the last split private key
	var lastSk bls.Fr
	bls.FrSub(&lastSk, &primarySk, &aggregateSk)

	lastKey := NewBLS0ChainScheme()
	var lastSecretKey bls.SecretKey
	lastSecretKey.SetLittleEndian(lastSk.Serialize())
	lastKey.privateKey = lastSecretKey.GetLittleEndian()
	lastSecretKey.SetLittleEndian(lastKey.privateKey)
	lastKey.publicKey = lastSecretKey.GetPublicKey().Serialize()
	splitKeys[numSplits-1] = lastKey
	return splitKeys, nil
}

//AggregateSignatures - implement interface
func (b0 *BLS0ChainScheme) AggregateSignatures(signatures []string) (string, error) {
	var aggSign bls.Sign
	for _, signature := range signatures {
		var sign bls.Sign
		sign.DeserializeHexStr(signature)
		aggSign.Add(&sign)
	}
	return aggSign.SerializeToHexStr(), nil
}
