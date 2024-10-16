package bls

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"0chain.net/chaincore/wallet"
	"github.com/0chain/common/core/logging"
	"github.com/herumi/bls-go-binary/bls"
)

type DKGID = bls.ID

type DKGKeyShareImpl struct {
	wallet *wallet.Wallet
	id     DKGID

	//This is private to the party
	Msk []bls.SecretKey // aik, coefficients of the polynomial, Si(x). ai0 = secret value of i

	//This is public information
	mpk []bls.PublicKey // Aik (Fik in some papers) = (g^aik), coefficients of the polynomial, Pi(x). Ai0 = public value of i

	//This is aggregate private info to the party from others
	sij map[bls.ID]bls.SecretKey // Sij = Si(j) for j in all parties

	//Below info is from Qual Set

	//This is arregate private info computed from sij
	si bls.SecretKey // Secret key share - Sigma(Sij) , for j in qual parties

	//This is publicly computable information for others and for self, it's the public key of the private share key
	pi *bls.PublicKey // Public key share - g^Si

	//This is the group public key
	gmpk []bls.PublicKey // Sigma(Aik) , for i in qual parties
}

type DKGSignatureShare struct {
	id        DKGID
	signature bls.Sign
}

type DKGSignature struct {
	shares []DKGSignatureShare
}

var wallets []*wallet.Wallet

var dkgShares []*DKGKeyShareImpl

// GenerateWallets - generate the wallets used to participate in DKG
func GenerateWallets(n int) {
	for i := 0; i < n; i++ {
		w := &wallet.Wallet{}
		if err := w.Initialize("bls0chain"); err != nil {
			panic(err)
		}
		wallets = append(wallets, w)
	}
}

// InitializeDKGShares - initialize DKG Share structures
func InitializeDKGShares() {
	dkgShares = dkgShares[:0]
	for _, w := range wallets {
		dkgShare := &DKGKeyShareImpl{wallet: w}
		if err := dkgShare.id.SetHexString("1" + w.ClientID[:31]); err != nil {
			panic(err)
		}
		dkgShare.sij = make(map[bls.ID]bls.SecretKey)
		dkgShares = append(dkgShares, dkgShare)
	}
}

// GenerateDKGKeyShare - create Si(x) and corresponding Pi(x) polynomial coefficients
func (dkgs *DKGKeyShareImpl) GenerateDKGKeyShare(t int) {
	var dsk bls.SecretKey
	dsk.SetByCSPRNG()
	dkgs.Msk = dsk.GetMasterSecretKey(t)
	dkgs.mpk = bls.GetMasterPublicKey(dkgs.Msk)
}

// GenerateSij - generate secret key shares from i for each party j
func (dkgs *DKGKeyShareImpl) GenerateSij(ids []DKGID) {
	dkgs.sij = make(map[bls.ID]bls.SecretKey)
	for _, id := range ids {
		var sij bls.SecretKey
		if err := sij.Set(dkgs.Msk, &id); err != nil {
			panic(err)
		}
		dkgs.sij[id] = sij
	}
}

// ValidateShare - validate Sij using Pj coefficients
func (dkgs *DKGKeyShareImpl) ValidateShare(jpk []bls.PublicKey, sij bls.SecretKey) bool {
	var expectedSijPK bls.PublicKey
	if err := expectedSijPK.Set(jpk, &dkgs.id); err != nil {
		panic(err)
	}
	sijPK := sij.GetPublicKey()
	return expectedSijPK.IsEqual(sijPK)
}

// AggregateSecretShares - compute Si = Sigma(Sij), j in qual and Pi = g^Si
// Useful to compute self secret key share and associated public key share
// For other parties, the public key share can be derived using the Pj(x) coefficients
func (dkgs *DKGKeyShareImpl) AggregateSecretKeyShares(qual []DKGID, dkgShares map[bls.ID]*DKGKeyShareImpl) {
	var sk bls.SecretKey
	for _, id := range qual {
		dkgsj, ok := dkgShares[id]
		if !ok {
			panic("no share")
		}
		sij := dkgsj.sij[dkgs.id]
		sk.Add(&sij)
	}
	dkgs.si = sk
	dkgs.pi = dkgs.si.GetPublicKey()
}

// ComputePublicKeyShare - compute the public key share of any party j, based on the coefficients of Pj(x)
func (dkgs *DKGKeyShareImpl) ComputePublicKeyShare(qual []DKGID, dkgShares map[bls.ID]*DKGKeyShareImpl) bls.PublicKey {
	var pk bls.PublicKey
	for _, id := range qual {
		dkgsj, ok := dkgShares[id]
		if !ok {
			panic("no share")
		}
		var pkj bls.PublicKey
		if err := pkj.Set(dkgsj.mpk, &dkgs.id); err != nil {
			panic(err)
		}
		pk.Add(&pkj)
	}
	return pk
}

// AggregatePublicKeyShares - compute Sigma(Aik, i in qual)
func (dkgs *DKGKeyShareImpl) AggregatePublicKeyShares(qual []DKGID, dkgShares map[bls.ID]*DKGKeyShareImpl) {
	dkgs.gmpk = dkgs.gmpk[:0]
	for k := 0; k < len(dkgs.mpk); k++ {
		var pk bls.PublicKey
		for _, id := range qual {
			dkgsj, ok := dkgShares[id]
			if !ok {
				panic("no share")
			}
			pk.Add(&dkgsj.mpk[k])
		}
		dkgs.gmpk = append(dkgs.gmpk, pk)
	}
}

// Sign - sign using the group secret key share
func (dkgs *DKGKeyShareImpl) Sign(msg string) string {
	return dkgs.si.Sign(msg).GetHexString()
}

// VerifySignature - verify the signature using the group public key share
func (dkgs *DKGKeyShareImpl) VerifySignature(msg string, sig *bls.Sign) bool {
	return sig.Verify(dkgs.pi, msg)
}

// VerifyGroupSignature - verify group signature using group public key
func (dkgs *DKGKeyShareImpl) VerifyGroupSignature(msg string, sig *bls.Sign) bool {
	return sig.Verify(&dkgs.gmpk[0], msg)
}

// Recover - given t signature shares, recover the group signature (using lagrange interpolation)
func (dkgs *DKGKeyShareImpl) Recover(dkgSigShares []DKGSignatureShare) (*bls.Sign, error) {
	var aggSig bls.Sign
	var signatures []Sign
	var ids []bls.ID
	t := len(dkgSigShares)
	if t > len(dkgs.Msk) {
		t = len(dkgs.Msk)
	}
	for k := 0; k < t; k++ {
		ids = append(ids, dkgSigShares[k].id)
		signatures = append(signatures, dkgSigShares[k].signature)
	}
	if err := aggSig.Recover(signatures, ids); err != nil {
		return nil, err
	}
	return &aggSig, nil
}

func init() {
	logging.InitLogging("development", "")
}

func TestGenerateDKG(tt *testing.T) {
	n := 20                                 //total participants at the beginning
	t := int(math.Round(0.67 * float64(n))) // threshold number of parties required to create aggregate signature
	q := int(math.Round(0.85 * float64(n))) // qualified to compute dkg based on DKG protocol execution
	if q == t && t < n {
		q++
	}
	GenerateWallets(n)
	InitializeDKGShares()

	var ids []DKGID
	var qualIDs []DKGID
	var qualDKGSharesMap map[bls.ID]*DKGKeyShareImpl

	for _, dkgs := range dkgShares {
		ids = append(ids, dkgs.id)
	}

	//Generate aik for each party (the polynomial for sharing the secret)
	for _, dkgs := range dkgShares {
		dkgs.GenerateDKGKeyShare(t)
	}

	//Generate Sij for each party (the p(id) value for a given id)
	for _, dkgs := range dkgShares {
		dkgs.GenerateSij(ids)
	}

	//Validate Sij shares received fromm others using P(x)
	for _, dkgsi := range dkgShares {
		for _, dkgsj := range dkgShares {
			sij := dkgsj.sij[dkgsi.id]
			valid := dkgsi.ValidateShare(dkgsj.mpk, sij)
			if !valid {
				tt.Errorf("%v -> %v share valid = %v\n", dkgsi.wallet.ClientID[:7], dkgsj.wallet.ClientID[:7], valid)
			}
		}
	}

	//Simulate Qual Set
	shuffled := make([]*DKGKeyShareImpl, n)
	perm := rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Perm(len(shuffled))
	for i, v := range perm {
		shuffled[v] = dkgShares[i]
	}
	qualDKGSharesMap = make(map[bls.ID]*DKGKeyShareImpl)
	for i := 0; i < q; i++ {
		qualIDs = append(qualIDs, shuffled[i].id)
		qualDKGSharesMap[shuffled[i].id] = shuffled[i]
	}

	//Compute si = Sigma sij, aggregate secret share of each qualified party
	for _, dkgsi := range qualDKGSharesMap {
		dkgsi.AggregateSecretKeyShares(qualIDs, qualDKGSharesMap)
		dkgsi.AggregatePublicKeyShares(qualIDs, qualDKGSharesMap)
	}
	for _, dkgsi := range qualDKGSharesMap {
		for _, dkgsj := range qualDKGSharesMap {
			if dkgsi == dkgsj {
				continue
			}
			pk := dkgsi.ComputePublicKeyShare(qualIDs, qualDKGSharesMap)
			if !pk.IsEqual(dkgsi.pi) {
				panic("public key share not valid")
			}
		}
	}

	msg := fmt.Sprintf("Hello 0Chain World %v", time.Now())
	falseMsg := fmt.Sprintf("Hello 0Chain World %v", time.Now())
	var falseCount int
	var signatures []DKGSignatureShare

	//Sign a message
	prng := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	for idx, id := range qualIDs {
		dkgsi, ok := qualDKGSharesMap[id]
		if !ok {
			panic(fmt.Sprintf("no share: %v\n", idx))
		}
		var sign string
		//if rand.Float64() < float64(t)/float64(n) {
		if prng.Float64() < float64(t)/float64(q) {
			sign = dkgsi.Sign(msg)
		} else {
			sign = dkgsi.Sign(falseMsg)
			falseCount++
		}
		var blsSig bls.Sign
		if err := blsSig.SetHexString(sign); err != nil {
			panic(err)
		}
		signature := DKGSignatureShare{signature: blsSig, id: dkgsi.id}
		signatures = append(signatures, signature)
	}
	//Aggregate Signatures
	count := 0
	for _, id := range qualIDs {
		count++
		dkgsi, ok := qualDKGSharesMap[id]
		if !ok {
			panic("no share")
		}
		var dkgSignature DKGSignature
		for _, signature := range signatures {
			if signature.id != dkgsi.id {
				if prng.Float64() < 0.10 {
					//To simulate network/byzantine condition of not getting the shares
					continue
				}
				dkgsj, ok := qualDKGSharesMap[signature.id]
				if !ok {
					panic("no share")
				}
				if !dkgsj.VerifySignature(msg, &signature.signature) {
					continue
				}
			}
			dkgSignature.shares = append(dkgSignature.shares, signature)
		}
		if len(dkgSignature.shares) < t {
			continue
		}
		shuffled := make([]DKGSignatureShare, len(dkgSignature.shares))
		perm := rand.New(rand.NewSource(int64(time.Now().Nanosecond()))).Perm(len(shuffled))
		//To simulate network condition and also self not having the share yet
		for i, v := range perm {
			shuffled[v] = dkgSignature.shares[i]
		}
		_, err := dkgsi.Recover(shuffled)
		if err != nil {
			fmt.Printf("Error recovering signature %v\n", err)
			continue
		}
	}
}
