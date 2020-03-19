package bls

/* DKG implementation */

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"github.com/herumi/bls/ffi/go/bls"
)

/*DKG - to manage DKG process */
type DKG struct {
	T  int
	N  int
	ID PartyID

	Msk []Key

	Sij                  map[PartyID]Key
	sijMutex             *sync.Mutex
	receivedSecretShares map[PartyID]Key
	secretSharesMutex    *sync.RWMutex

	Si Key
	Pi *PublicKey

	Mpk []PublicKey

	gmpkMutex *sync.RWMutex
	gmpk      map[PartyID]PublicKey

	MagicBlockNumber int64
	StartingRound    int64
}

type DKGSummary struct {
	datastore.IDField
	StartingRound int64             `json:"starting_round"`
	SecretShares  map[string]string `json:"secret_shares"`
}

var dkgSummaryMetadata *datastore.EntityMetadataImpl

/* init -  To initialize a point on the curve */
func init() {
	err := bls.Init(bls.CurveFp254BNb)
	if err != nil {
		panic(fmt.Errorf("bls initialization error: %v", err))
	}
}

/*MakeDKG - to create a dkg object */
func MakeDKG(t, n int, id string) *DKG {
	dkg := &DKG{
		T:                    t,
		N:                    n,
		Sij:                  make(map[PartyID]Key),
		receivedSecretShares: make(map[PartyID]Key),
		secretSharesMutex:    &sync.RWMutex{},
		sijMutex:             &sync.Mutex{},
		Si:                   Key{},
		ID:                   PartyID{},
		gmpkMutex:            &sync.RWMutex{},
	}
	var secKey Key
	secKey.SetByCSPRNG()

	dkg.ID = ComputeIDdkg(id)
	dkg.Msk = secKey.GetMasterSecretKey(t)
	dkg.Mpk = bls.GetMasterPublicKey(dkg.Msk)
	return dkg
}

/*MakeDKG - to create a dkg object */
func SetDKG(t, n int, shares map[string]string, msk []string, mpks map[PartyID][]PublicKey, id string) *DKG {
	dkg := &DKG{
		T:                    t,
		N:                    n,
		Sij:                  make(map[PartyID]Key),
		receivedSecretShares: make(map[PartyID]Key),
		secretSharesMutex:    &sync.RWMutex{},
		sijMutex:             &sync.Mutex{},
		Si:                   Key{},
		ID:                   PartyID{},
		gmpkMutex:            &sync.RWMutex{},
	}
	dkg.ID = ComputeIDdkg(id)
	for _, v := range msk {
		var secretKey Key
		err := secretKey.SetHexString(v)
		if err != nil {
			panic(err.Error())
		}
		dkg.Msk = append(dkg.Msk, secretKey)
	}
	dkg.Mpk = bls.GetMasterPublicKey(dkg.Msk)
	dkg.AggregatePublicKeyShares(mpks)
	for k, v := range shares {
		var secreteShare Key
		err := secreteShare.SetHexString(v)
		if err != nil {
			panic(err.Error())
		}
		if dkg.ValidateShare(mpks[ComputeIDdkg(k)], secreteShare) {
			id := ComputeIDdkg(k)
			dkg.receivedSecretShares[id] = secreteShare
		} else {
			panic("failed to verify secret share")
		}
	}
	dkg.AggregateSecretKeyShares()
	return dkg
}

/*ComputeIDdkg - to create an ID of party of type PartyID */
func ComputeIDdkg(minerID string) PartyID {
	var forID PartyID
	if err := forID.SetHexString("1" + minerID[:31]); err != nil {
		fmt.Printf("Error while computing ID %s\n", forID.GetHexString())
	}
	return forID
}

/*ComputeDKGKeyShare - Derive the share for each miner through polynomial substitution method */
func (dkg *DKG) ComputeDKGKeyShare(forID PartyID) (Key, error) {
	var secVec Key
	err := secVec.Set(dkg.Msk, &forID)
	if err != nil {
		return Key{}, err
	}
	dkg.Sij[forID] = secVec
	return secVec, nil
}

/*GetKeyShareForOther - Get the DKGKeyShare for this Miner specified by the PartyID */
func (dkg *DKG) GetKeyShareForOther(to PartyID) *DKGKeyShare {
	dkg.sijMutex.Lock()
	defer dkg.sijMutex.Unlock()
	indivShare, ok := dkg.Sij[to]
	if !ok {
		return nil
	}
	dShare := &DKGKeyShare{Share: indivShare.GetHexString()}
	dShare.SetKey(to.GetHexString())
	return dShare
}

/*AggregateShares - Each party aggregates the received shares from other party which is calculated for that party */
func (dkg *DKG) AggregateSecretKeyShares() {
	var sk Key
	dkg.secretSharesMutex.RLock()
	defer dkg.secretSharesMutex.RUnlock()
	for _, Sij := range dkg.receivedSecretShares {
		sk.Add(&Sij)
	}
	dkg.Si = sk
	dkg.Pi = dkg.Si.GetPublicKey()
}

/*AggregateShares - Each party aggregates the received shares from other party which is calculated for that party */
func (dkg *DKG) GetSecretKeyShares() []string {
	var shares []string
	dkg.secretSharesMutex.RLock()
	defer dkg.secretSharesMutex.RUnlock()
	for _, Sij := range dkg.receivedSecretShares {
		shares = append(shares, Sij.GetHexString())
	}
	return shares
}

// AddSecretShare adds secret share for miner
//   - Force - replace share for miner
func (dkg *DKG) AddSecretShare(id PartyID, share string, force bool) error {
	dkg.secretSharesMutex.Lock()
	defer dkg.secretSharesMutex.Unlock()

	var secretShare Key
	if err := secretShare.SetHexString(share); err != nil {
		return err
	}

	if shareFound, ok := dkg.receivedSecretShares[id]; ok && !secretShare.IsEqual(&shareFound) {
		if !force {
			return common.NewError("failed to add secret share", "share already exists for miner")
		}
	}

	dkg.receivedSecretShares[id] = secretShare
	return nil
}

/*ComputeDKGKeyShare - Derive the share for each miner through polynomial substitution method */
func (dkg *DKG) GetSecretSharesSize() int {
	dkg.secretSharesMutex.RLock()
	defer dkg.secretSharesMutex.RUnlock()
	return len(dkg.receivedSecretShares)
}

/*ComputeDKGKeyShare - Derive the share for each miner through polynomial substitution method */
func (dkg *DKG) HasAllSecretShares() bool {
	dkg.secretSharesMutex.RLock()
	defer dkg.secretSharesMutex.RUnlock()
	return len(dkg.receivedSecretShares) >= dkg.T
}

func (dkg *DKG) HasSecretShare(key string) bool {
	dkg.secretSharesMutex.RLock()
	defer dkg.secretSharesMutex.RUnlock()
	_, ok := dkg.receivedSecretShares[ComputeIDdkg(key)]
	return ok
}

//Sign - sign using the group secret key share
func (dkg *DKG) Sign(msg string) *Sign {
	return dkg.Si.Sign(msg)
}

//VerifySignature - verify the signature using the group public key share
func (dkg *DKG) VerifySignature(sig *Sign, msg string, id PartyID) bool {
	dkg.gmpkMutex.RLock()
	defer dkg.gmpkMutex.RUnlock()
	key := dkg.gmpk[id]
	return sig.Verify(&key, msg)
}

/*RecoverGroupSig - To compute the Gp sign with any k number of BLS sig shares */
func (dkg *DKG) RecoverGroupSig(from []PartyID, shares []Sign) (Sign, error) {
	var sig Sign
	t := len(shares)
	if t > len(dkg.Msk) {
		t = len(dkg.Msk)
	}
	err := sig.Recover(shares, from)
	if err == nil {
		return sig, nil
	}
	return Sign{}, err
}

// CalBlsGpSign - The function calls the RecoverGroupSig function which calculates the Gp Sign
func (dkg *DKG) CalBlsGpSign(recSig []string, recIDs []string) (Sign, error) {
	signVec := make([]Sign, 0)
	var signShare Sign
	for i := 0; i < len(recSig); i++ {
		err := signShare.SetHexString(recSig[i])
		if err == nil {
			signVec = append(signVec, signShare)
		} else {
			return Sign{}, err
		}
	}
	idVec := make([]PartyID, 0)
	var forID PartyID
	for i := 0; i < len(recIDs); i++ {
		err := forID.SetHexString(recIDs[i])
		if err == nil {
			idVec = append(idVec, forID)
		}
	}
	return dkg.RecoverGroupSig(idVec, signVec)
}

//AggregatePublicKeyShares - compute Sigma(Aik, i in qual)
func (dkg *DKG) AggregatePublicKeyShares(mpks map[PartyID][]PublicKey) {
	dkg.gmpkMutex.Lock()
	defer dkg.gmpkMutex.Unlock()
	dkg.gmpk = make(map[PartyID]PublicKey)
	for k := range mpks {
		var pk PublicKey
		for _, mpk := range mpks {
			var pkj PublicKey
			pkj.Set(mpk, &k)
			pk.Add(&pkj)
		}
		dkg.gmpk[k] = pk
	}
}

// GetPublicKeyByID - returns public key by party id
func (dkg *DKG) GetPublicKeyByID(id PartyID) PublicKey {
	dkg.gmpkMutex.RLock()
	defer dkg.gmpkMutex.RUnlock()
	return dkg.gmpk[id]
}

/*CreateQualSet - Each party aggregates the received shares from other party which is calculated for that party */
func (dkg *DKG) DeleteFromSet(nodes []string) {
	dkg.secretSharesMutex.Lock()
	defer dkg.secretSharesMutex.Unlock()
	for _, id := range nodes {
		delete(dkg.receivedSecretShares, ComputeIDdkg(id))
	}
}

//ValidateShare - validate Sij using Pj coefficients
func (dkg *DKG) ValidateShare(jpk []PublicKey, sij bls.SecretKey) bool {
	return ValidateShare(jpk, sij, dkg.ID)
}

//ValidateShare - validate Sij using Pj coefficients
func ValidateShare(jpk []PublicKey, sij bls.SecretKey, id PartyID) bool {
	var mpk []string
	for _, pk := range jpk {
		mpk = append(mpk, pk.GetHexString())
	}
	var expectedSijPK PublicKey
	if err := expectedSijPK.Set(jpk, &id); err != nil {
		return false
	}
	sijPK := sij.GetPublicKey()
	return expectedSijPK.IsEqual(sijPK)
}

func ConvertStringToMpk(strMpk []string) []PublicKey {
	var mpk []PublicKey
	for _, str := range strMpk {
		var publickKey PublicKey
		publickKey.SetHexString(str)
		mpk = append(mpk, publickKey)
	}
	return mpk
}

func (dkgSummary *DKGSummary) GetEntityMetadata() datastore.EntityMetadata {
	return dkgSummaryMetadata
}

func DKGSummaryProvider() datastore.Entity {
	dkgSummary := &DKGSummary{}
	return dkgSummary
}

func SetupDKGSummary(store datastore.Store) {
	dkgSummaryMetadata = datastore.MetadataProvider()
	dkgSummaryMetadata.Name = "dkgsummary"
	dkgSummaryMetadata.DB = "dkgsummarydb"
	dkgSummaryMetadata.Store = store
	dkgSummaryMetadata.Provider = DKGSummaryProvider
	datastore.RegisterEntityMetadata("dkgsummary", dkgSummaryMetadata)
}

func SetupDKGDB() {
	db, err := ememorystore.CreateDB("data/rocksdb/dkg")
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("dkgsummarydb", db)
}

func (dkgSummary *DKGSummary) Read(ctx context.Context, key string) error {
	return dkgSummary.GetEntityMetadata().GetStore().Read(ctx, key, dkgSummary)
}

func (dkgSummary *DKGSummary) Write(ctx context.Context) error {
	return dkgSummary.GetEntityMetadata().GetStore().Write(ctx, dkgSummary)
}

func (dkgSummary *DKGSummary) Delete(ctx context.Context) error {
	return dkgSummary.GetEntityMetadata().GetStore().Delete(ctx, dkgSummary)
}

func (dkg *DKG) GetDKGSummary() *DKGSummary {
	dkgSummary := &DKGSummary{
		SecretShares:  make(map[string]string),
		StartingRound: dkg.StartingRound,
	}
	dkg.secretSharesMutex.RLock()
	defer dkg.secretSharesMutex.RUnlock()
	for k, v := range dkg.receivedSecretShares {
		dkgSummary.SecretShares[k.GetHexString()] = v.GetHexString()
	}
	dkgSummary.ID = strconv.FormatInt(dkg.MagicBlockNumber, 10)
	return dkgSummary
}
