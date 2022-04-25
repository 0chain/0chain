package cmd

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"0chain.net/core/encryption"

	"github.com/herumi/bls/ffi/go/bls"
)

var GenG2 *bls.G2
var ErrKeyRead = errors.New("error reading the keys")

func init() {
	err := bls.Init(int(bls.CurveFp254BNb))
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
	err = GenG2.Deserialize(bytes)
	if err != nil {
		panic(err)
	}
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
	if !result {
		return ErrKeyRead
	}
	publicKey := scanner.Text()
	err := b0.SetPublicKey(publicKey)
	if err != nil {
		return err
	}
	result = scanner.Scan()
	if !result {
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

// Converts public key 'pk' to format that the herumi/bls library likes.
// It's possible to get a MIRACL PublicKey which is of much longer format
// (See below example), as wallets are using MIRACL library not herumi lib.
// If 'pk' is not in MIRACL format, we just return the original 'pk' then.
//
// This is an example of the raw public key we expect from MIRACL
var miraclExamplePK = `0418a02c6bd223ae0dfda1d2f9a3c81726ab436ce5e9d17c531ff0a385a13a0b491bdfed3a85690775ee35c61678957aaba7b1a1899438829f1dc94248d87ed36817f6dfafec19bfa87bf791a4d694f43fec227ae6f5a867490e30328cac05eaff039ac7dfc3364e851ebd2631ea6f1685609fc66d50223cc696cb59ff2fee47ac`

//
// This is an example of the same MIRACL public key serialized with ToString().
// pk ([1bdfed3a85690775ee35c61678957aaba7b1a1899438829f1dc94248d87ed368,18a02c6bd223ae0dfda1d2f9a3c81726ab436ce5e9d17c531ff0a385a13a0b49],[039ac7dfc3364e851ebd2631ea6f1685609fc66d50223cc696cb59ff2fee47ac,17f6dfafec19bfa87bf791a4d694f43fec227ae6f5a867490e30328cac05eaff])
func MiraclToHerumiPK(pk string) string {
	if len(pk) != len(miraclExamplePK) {
		return pk
	}
	n1 := pk[2:66]
	n2 := pk[66:(66 + 64)]
	n3 := pk[(66 + 64):(66 + 64 + 64)]
	n4 := pk[(66 + 64 + 64):(66 + 64 + 64 + 64)]
	var p bls.PublicKey
	err := p.SetHexString("1 " + n2 + " " + n1 + " " + n4 + " " + n3)
	if err != nil {
		panic(err)
	}
	return p.SerializeToHexStr()
}

// Converts signature 'sig' to format that the herumi/bls library likes.
// zwallets are using MIRACL library which send a MIRACL signature not herumi
// lib.
//
// If the 'sig' was not in MIRACL format, we just return the original sig.
var miraclExampleSig = `(0d4dbad6d2586d5e01b6b7fbad77e4adfa81212c52b4a0b885e19c58e0944764,110061aa16d5ba36eef0ad4503be346908d3513c0a2aedfd0d2923411b420eca)`

func MiraclToHerumiSig(sig string) string {
	if len(sig) <= 2 {
		return sig
	}
	if sig[0] != miraclExampleSig[0] {
		return sig
	}
	withoutParens := sig[1:(len(sig) - 1)]
	comma := strings.Index(withoutParens, ",")
	if comma < 0 {
		return "00"
	}
	n1 := withoutParens[0:comma]
	n2 := withoutParens[(comma + 1):]
	var sign bls.Sign
	err := sign.SetHexString("1 " + n1 + " " + n2)
	if err != nil {
		panic(err)
	}
	return sign.SerializeToHexStr()
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
	publicKey = MiraclToHerumiPK(publicKey)
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		return err
	}
	b0.publicKey = publicKeyBytes
	return nil
}

func (b0 *BLS0ChainScheme) SetPrivateKey(privateKey string) {
	b0.privateKey = []byte(privateKey)
}

//GetPublicKey - implement interface
func (b0 *BLS0ChainScheme) GetPublicKey() string {
	return hex.EncodeToString(b0.publicKey)
}

func (b0 *BLS0ChainScheme) GetPrivateKey() string {
	return string(b0.privateKey)
}

//Sign - implement interface
func (b0 *BLS0ChainScheme) Sign(hash interface{}) (string, error) {
	var sk bls.SecretKey
	if err := sk.SetLittleEndian(b0.privateKey); err != nil {
		return "", err
	}
	rawHash, err := encryption.GetRawHash(hash)
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
	err := sign.DeserializeHexStr(MiraclToHerumiSig(signature))
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
	if err := g1.HashAndMapTo(rawHash); err != nil {
		return nil, err
	}
	var gt = &bls.GT{}
	bls.Pairing(gt, g1, g2)
	return gt, nil
}

//GenerateSplitKeys - implement interface
func (b0 *BLS0ChainScheme) GenerateSplitKeys(numSplits int) ([]encryption.SignatureScheme, error) {
	var primarySk bls.Fr
	err := primarySk.SetLittleEndian(b0.privateKey)
	if err != nil {
		return nil, err
	}

	splitKeys := make([]encryption.SignatureScheme, numSplits)
	var sk bls.SecretKey

	//Generate all but one split keys and add the secret keys
	for i := 0; i < numSplits-1; i++ {
		key := NewBLS0ChainScheme()
		err := key.GenerateKeys()
		if err != nil {
			return nil, err
		}
		splitKeys[i] = key
		var sk2 bls.SecretKey
		err = sk2.SetLittleEndian(key.privateKey)
		if err != nil {
			return nil, err
		}
		sk.Add(&sk2)
	}
	var aggregateSk bls.Fr
	err = aggregateSk.SetLittleEndian(sk.GetLittleEndian())
	if err != nil {
		return nil, err
	}

	//Subtract the aggregated private key from the primary private key to derive the last split private key
	var lastSk bls.Fr
	bls.FrSub(&lastSk, &primarySk, &aggregateSk)

	lastKey := NewBLS0ChainScheme()
	var lastSecretKey bls.SecretKey
	err = lastSecretKey.SetLittleEndian(lastSk.Serialize())
	if err != nil {
		return nil, err
	}
	lastKey.privateKey = lastSecretKey.GetLittleEndian()
	err = lastSecretKey.SetLittleEndian(lastKey.privateKey)
	if err != nil {
		return nil, err
	}
	lastKey.publicKey = lastSecretKey.GetPublicKey().Serialize()
	splitKeys[numSplits-1] = lastKey
	return splitKeys, nil
}

//AggregateSignatures - implement interface
func (b0 *BLS0ChainScheme) AggregateSignatures(signatures []string) (string, error) {
	var aggSign bls.Sign
	for _, signature := range signatures {
		var sign bls.Sign
		if err := sign.DeserializeHexStr(MiraclToHerumiSig(signature)); err != nil {
			return "", err
		}
		aggSign.Add(&sign)
	}
	return aggSign.SerializeToHexStr(), nil
}
