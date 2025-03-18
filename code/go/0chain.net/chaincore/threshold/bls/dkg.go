package bls

/* DKG implementation */

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"github.com/0chain/common/core/logging"
	"github.com/herumi/bls-go-binary/bls"
	"go.uber.org/zap"
)

/*DKG - to manage DKG process */
type DKG struct {
	T  int
	N  int
	ID PartyID

	msk []Key

	sij                  map[PartyID]Key
	sijMutex             *sync.Mutex
	receivedSecretShares map[PartyID]Key
	secretSharesMutex    *sync.RWMutex

	Si Key
	Pi *PublicKey

	mpksMutex  *sync.Mutex
	mpks       []PublicKey
	mpksMap    map[PartyID][]PublicKey
	mpksMapStr map[PartyID][]string

	gmpkMutex *sync.RWMutex
	gmpk      map[PartyID]PublicKey

	MagicBlockNumber int64
	StartingRound    int64
}

type DKGSummary struct {
	datastore.IDField
	StartingRound int64             `json:"starting_round"`
	SecretShares  map[string]string `json:"secret_shares"`
	IsFinalized   bool              `json:"is_finalized"`
}

// LatestMagicBlockID keeps ID of latest MB accepted and stored.
type LatestMagicBlockID struct {
	datastore.IDField
}

var dkgSummaryMetadata *datastore.EntityMetadataImpl

/* init -  To initialize a point on the curve */
func init() {
	err := bls.Init(int(bls.CurveFp254BNb))
	if err != nil {
		panic(fmt.Errorf("bls initialization error: %v", err))
	}
}

/*MakeDKG - to create a dkg object */
func MakeDKG(t, n int, id string) *DKG {
	dkg := &DKG{
		T:                    t,
		N:                    n,
		sij:                  make(map[PartyID]Key),
		receivedSecretShares: make(map[PartyID]Key),
		secretSharesMutex:    &sync.RWMutex{},
		sijMutex:             &sync.Mutex{},
		Si:                   Key{},
		ID:                   PartyID{},
		gmpkMutex:            &sync.RWMutex{},
		mpksMutex:            &sync.Mutex{},
	}
	var secKey Key
	secKey.SetByCSPRNG()

	dkg.ID = ComputeIDdkg(id)
	dkg.msk = secKey.GetMasterSecretKey(t)
	dkg.mpks = bls.GetMasterPublicKey(dkg.msk)
	dkg.mpksMapStr = make(map[PartyID][]string)
	dkg.gmpk = make(map[PartyID]PublicKey)
	return dkg
}

// SetDKG - to create a dkg object
func SetDKG(t, n int, shares map[string]string, msk []string, mpks map[PartyID][]PublicKey, id string) *DKG {
	dkg := &DKG{
		T:                    t,
		N:                    n,
		sij:                  make(map[PartyID]Key),
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
		dkg.msk = append(dkg.msk, secretKey)
	}
	dkg.mpks = bls.GetMasterPublicKey(dkg.msk)
	if err := dkg.AggregatePublicKeyShares(mpks); err != nil {
		panic(err)
	}

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

// ComputeIDdkg - to create an ID of party of type PartyID
func ComputeIDdkg(minerID string) PartyID {
	var forID PartyID
	if err := forID.SetHexString("1" + minerID[:31]); err != nil {
		fmt.Printf("Error while computing ID %s\n", forID.GetHexString())
		panic(fmt.Sprintf("Error while computing ID %s, err: %v", forID.GetHexString(), err))
	}
	return forID
}

// GetMPKs returns the mpks
func (dkg *DKG) GetMPKs() []PublicKey {
	dkg.mpksMutex.Lock()
	defer dkg.mpksMutex.Unlock()
	mpks := make([]PublicKey, len(dkg.mpks))
	copy(mpks, dkg.mpks)
	return mpks
}

// ComputeDKGKeyShare - Derive the share for each miner through polynomial substitution method
func (dkg *DKG) ComputeDKGKeyShare(forID PartyID) (Key, error) {
	var secVec Key
	err := secVec.Set(dkg.msk, &forID)
	if err != nil {
		return Key{}, err
	}

	dkg.sijMutex.Lock()
	dkg.sij[forID] = secVec
	dkg.sijMutex.Unlock()
	return secVec, nil
}

// GetDKGKeyShare gets the DKGKeyShare of given PartyID
func (dkg *DKG) GetDKGKeyShare(to PartyID) *DKGKeyShare {
	dkg.sijMutex.Lock()
	defer dkg.sijMutex.Unlock()
	share, ok := dkg.sij[to]
	if !ok {
		return nil
	}
	dShare := &DKGKeyShare{Share: share.GetHexString()}
	dShare.SetKey(to.GetHexString())
	return dShare
}

// GetKeyShare gets the Key of given PartyID
func (dkg *DKG) GetKeyShare(id PartyID) (Key, bool) {
	dkg.sijMutex.Lock()
	defer dkg.sijMutex.Unlock()
	share, ok := dkg.sij[id]
	return share, ok
}

// GetSijLen returns the length of Sij
func (dkg *DKG) GetSijLen() int {
	dkg.sijMutex.Lock()
	defer dkg.sijMutex.Unlock()
	return len(dkg.sij)
}

// AggregateSecretKeyShares - Each party aggregates the received shares from other party which is calculated for that party
func (dkg *DKG) AggregateSecretKeyShares() {
	dkg.secretSharesMutex.RLock()
	defer dkg.secretSharesMutex.RUnlock()

	aggStart := time.Now()
	defer func() {
		logging.Logger.Debug("[dkg_timing] Aggregate secret key shares",
			zap.Duration("duration", time.Since(aggStart)))
	}()

	if len(dkg.receivedSecretShares) == 0 {
		dkg.Si = Key{}
		dkg.Pi = nil
		return
	}

	// For small numbers of shares, use sequential processing
	if len(dkg.receivedSecretShares) < 8 {
		var sk Key
		for _, Sij := range dkg.receivedSecretShares {
			sk.Add(&Sij)
		}
		dkg.Si = sk
		dkg.Pi = dkg.Si.GetPublicKey()
		return
	}

	// For larger numbers, use parallel processing
	numWorkers := runtime.NumCPU()
	if numWorkers > len(dkg.receivedSecretShares) {
		numWorkers = len(dkg.receivedSecretShares)
	}

	// Split work into chunks
	shares := make([]Key, 0, len(dkg.receivedSecretShares))
	for _, share := range dkg.receivedSecretShares {
		shares = append(shares, share)
	}

	// Calculate optimal chunk size to avoid empty slices
	chunkSize := (len(shares) + numWorkers - 1) / numWorkers
	results := make(chan Key, numWorkers)
	var wg sync.WaitGroup

	// Process chunks in parallel
	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		// Skip if we're past the end of the slice
		if start >= len(shares) {
			continue
		}

		end := start + chunkSize
		if end > len(shares) {
			end = len(shares)
		}

		// Don't create empty chunks
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(chunk []Key) {
			defer wg.Done()
			var partialSum Key
			for _, share := range chunk {
				partialSum.Add(&share)
			}
			results <- partialSum
		}(shares[start:end])
	}

	// Close results channel after all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Combine partial results
	var finalSum Key
	for partialSum := range results {
		finalSum.Add(&partialSum)
	}

	dkg.Si = finalSum
	dkg.Pi = dkg.Si.GetPublicKey()
}

// GetSecretKeyShares - Each party aggregates the received shares from other party which is calculated for that party
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

func (dkg *DKG) GetSecretShare(key string) (Key, bool) {
	dkg.secretSharesMutex.RLock()
	defer dkg.secretSharesMutex.RUnlock()
	share, ok := dkg.receivedSecretShares[ComputeIDdkg(key)]
	return share, ok
}

// Sign - sign using the group secret key share
func (dkg *DKG) Sign(msg string) *Sign {
	logging.Logger.Debug("dkg sign",
		zap.String("key", dkg.Si.GetHexString()),
		zap.String("pi", dkg.Pi.GetHexString()))
	return dkg.Si.Sign(msg)
}

// VerifySignature - verify the signature using the group public key share
func (dkg *DKG) VerifySignature(sig *Sign, msg string, id PartyID) bool {
	dkg.gmpkMutex.Lock()
	defer dkg.gmpkMutex.Unlock()
	verifyStart := time.Now()
	defer func() {
		if time.Since(verifyStart) > 200*time.Millisecond {
			logging.Logger.Debug("[dkg_timing] Verify signature slow",
				zap.Duration("duration", time.Since(verifyStart)))
		}
	}()

	key, ok := dkg.gmpk[id]
	if !ok {
		if dkg.mpksMap == nil {
			mpks, err := dkg.getMpkMap()
			if err != nil {
				logging.Logger.Error("dkg verify signature, failed to get mpk map",
					zap.Error(err))
				return false
			}
			dkg.mpksMap = mpks
		}

		var err error
		key, err = aggregatePublicKeysForID(dkg.mpksMap, id)
		if err != nil {
			logging.Logger.Error("dkg verify signature, failed to aggregate public key shares",
				zap.Error(err))
			return false
		}

		dkg.gmpk[id] = key
	}
	logging.Logger.Debug("dkg verify",
		zap.String("id", id.GetHexString()),
		zap.String("key", key.GetHexString()),
		zap.String("msg", msg),
		zap.String("sig", sig.GetHexString()))
	return sig.Verify(&key, msg)
}

func (dkg *DKG) getMpkMap() (map[PartyID][]PublicKey, error) {
	startTime := time.Now()
	defer func() {
		logging.Logger.Debug("[dkg_timing] Parallel MPK map conversion",
			zap.Duration("duration", time.Since(startTime)))
	}()

	result := make(map[PartyID][]PublicKey)
	var (
		numWorkers = runtime.NumCPU()
		wg         sync.WaitGroup
		mu         sync.Mutex
		errChan    = make(chan error, 1)
	)

	// Create work channel
	workChan := make(chan struct {
		key PartyID
		mpk []string
	}, len(dkg.mpksMapStr))

	// Feed work channel
	for k, v := range dkg.mpksMapStr {
		workChan <- struct {
			key PartyID
			mpk []string
		}{k, v}
	}
	close(workChan)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				// Convert string array to PublicKey array
				pks, err := ConvertStringToMpk(work.mpk)
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}

				// Convert key to PartyID
				// Safely store result
				mu.Lock()
				result[work.key] = pks
				mu.Unlock()
			}
		}()
	}

	// Wait in a separate goroutine
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for errors
	if err := <-errChan; err != nil {
		return nil, err
	}

	return result, nil
}

/*RecoverGroupSig - To compute the Gp sign with any k number of BLS sig shares */
func (dkg *DKG) RecoverGroupSig(from []PartyID, shares []Sign) (Sign, error) {
	var sig Sign
	if err := sig.Recover(shares, from); err != nil {
		return Sign{}, err
	}

	return sig, nil
}

// CalBlsGpSign - The function calls the RecoverGroupSig function which calculates the Gp Sign
func (dkg *DKG) CalBlsGpSign(recSig []string, recIDs []string) (Sign, error) {
	// logging.Logger.Debug("dkg recover",
	// 	zap.Strings("recSig", recSig),
	// 	zap.Strings("recIDs", recIDs))

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

	if len(idVec) == 0 || len(signVec) == 0 {
		return Sign{}, errors.New("empty id or share")
	}
	return dkg.RecoverGroupSig(idVec, signVec)
}

// Helper function to parallelize the inner loop of public key aggregation for a given PartyID
func aggregatePublicKeysForID(mpks map[PartyID][]PublicKey, k PartyID) (PublicKey, error) {
	// For small numbers of mpks, use sequential processing
	aggStart := time.Now()
	defer func() {
		logging.Logger.Debug("[dkg_timing] Aggregate public key shares for id",
			zap.Duration("duration", time.Since(aggStart)))
	}()
	if len(mpks) < 8 {
		var pk PublicKey
		for _, mpk := range mpks {
			var pkj PublicKey
			if err := pkj.Set(mpk, &k); err != nil {
				return PublicKey{}, err
			}
			pk.Add(&pkj)
		}
		return pk, nil
	}

	// For larger numbers, use parallel processing
	numWorkers := runtime.NumCPU()
	if numWorkers > len(mpks) {
		numWorkers = len(mpks)
	}

	type result struct {
		pk  PublicKey
		err error
	}

	// Prepare the mpk keys for chunking
	mpkKeys := make([]PartyID, 0, len(mpks))
	for mpkKey := range mpks {
		mpkKeys = append(mpkKeys, mpkKey)
	}

	chunkSize := (len(mpkKeys) + numWorkers - 1) / numWorkers
	results := make(chan result, numWorkers)
	var wg sync.WaitGroup

	// Process chunks in parallel
	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		// Skip if we're past the end of the slice
		if start >= len(mpkKeys) {
			continue
		}

		end := start + chunkSize
		if end > len(mpkKeys) {
			end = len(mpkKeys)
		}

		// Don't create empty chunks
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(chunk []PartyID) {
			defer wg.Done()
			var partialSum PublicKey
			for _, mpkKey := range chunk {
				mpk := mpks[mpkKey]
				var pkj PublicKey
				if err := pkj.Set(mpk, &k); err != nil {
					results <- result{err: err}
					return
				}
				partialSum.Add(&pkj)
			}
			results <- result{pk: partialSum}
		}(mpkKeys[start:end])
	}

	// Close results channel after all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Combine partial results
	var finalSum PublicKey
	for r := range results {
		if r.err != nil {
			return PublicKey{}, r.err
		}
		finalSum.Add(&r.pk)
	}

	return finalSum, nil
}

// Helper function for parallel processing
func (dkg *DKG) aggregatePublicKeySharesParallel(mpks map[PartyID][]PublicKey) (map[PartyID]PublicKey, error) {
	var (
		numWorkers = runtime.NumCPU()
		wg         sync.WaitGroup
		mu         sync.Mutex
		result     = make(map[PartyID]PublicKey)
		errChan    = make(chan error, 1)
	)

	// Create work channel
	workChan := make(chan PartyID, len(mpks))
	for k := range mpks {
		workChan <- k
	}
	close(workChan)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range workChan {
				// Use the helper function for parallelized inner processing
				pk, err := aggregatePublicKeysForID(mpks, k)
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}

				mu.Lock()
				result[k] = pk
				mu.Unlock()
			}
		}()
	}

	// Wait for completion
	wg.Wait()
	close(errChan)

	// Check for errors
	if err := <-errChan; err != nil {
		return nil, err
	}

	return result, nil
}

func (dkg *DKG) SetMpksMap(mpks map[string][]string) {
	dkg.gmpkMutex.Lock()
	dkg.mpksMapStr = mpks
	dkg.gmpkMutex.Unlock()
}

// Parallel version
func (dkg *DKG) AggregatePublicKeyShares(mpks map[PartyID][]PublicKey) error {
	startTime := time.Now()
	defer func() {
		logging.Logger.Debug("[dkg_timing] Parallel public key shares aggregation",
			zap.Duration("duration", time.Since(startTime)))
	}()

	dkg.gmpkMutex.Lock()
	defer dkg.gmpkMutex.Unlock()

	result, err := dkg.aggregatePublicKeySharesParallel(mpks)
	if err != nil {
		return err
	}

	dkg.gmpk = result
	return nil
}

// GetPublicKeyByID - returns public key by party id
func (dkg *DKG) GetPublicKeyByID(id PartyID) PublicKey {
	dkg.gmpkMutex.RLock()
	defer dkg.gmpkMutex.RUnlock()
	return dkg.gmpk[id]
}

// DeleteFromSet - Each party aggregates the received shares from other party which is calculated for that party */
func (dkg *DKG) DeleteFromSet(nodes []string) {
	dkg.secretSharesMutex.Lock()
	defer dkg.secretSharesMutex.Unlock()
	for _, id := range nodes {
		delete(dkg.receivedSecretShares, ComputeIDdkg(id))
	}
	logging.Logger.Debug("[mvc] dkg_ss, delete from dkg set",
		zap.Int("deleted", len(nodes)),
		zap.Int("dkg received ss", len(dkg.receivedSecretShares)))
}

// ValidateShare - validate Sij using Pj coefficients
func (dkg *DKG) ValidateShare(jpk []PublicKey, sij bls.SecretKey) bool {
	return ValidateShare(jpk, sij, dkg.ID)
}

// ValidateShare - validate Sij using Pj coefficients
func ValidateShare(jpk []PublicKey, sij bls.SecretKey, id PartyID) bool {
	var expectedSijPK PublicKey
	if err := expectedSijPK.Set(jpk, &id); err != nil {
		return false
	}
	sijPK := sij.GetPublicKey()
	return expectedSijPK.IsEqual(sijPK)
}

func ConvertStringToMpk(strMpk []string) ([]PublicKey, error) {
	if len(strMpk) == 0 {
		return nil, nil
	}

	// Create channels for work distribution and result collection
	type result struct {
		index int
		pk    PublicKey
		err   error
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > len(strMpk) {
		numWorkers = len(strMpk)
	}

	jobs := make(chan int, len(strMpk))
	results := make(chan result, len(strMpk))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				var pk PublicKey
				err := pk.SetHexString(strMpk[idx])
				results <- result{index: idx, pk: pk, err: err}
			}
		}()
	}

	// Send jobs
	for i := range strMpk {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	mpk := make([]PublicKey, len(strMpk))
	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		mpk[r.index] = r.pk
	}

	return mpk, nil
}

//
// DKG summary storage.
//

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

func SetupDKGDB(workdir string) {
	db, err := ememorystore.CreateDB(filepath.Join(workdir, "data/rocksdb/dkg"))
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("dkgsummarydb", db)
}

func (dkgSummary *DKGSummary) Decode(input []byte) error {
	return json.Unmarshal(input, dkgSummary)
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

// Verify is used to verify a dkg summary with the mpks
func (dkgSummary *DKGSummary) Verify(id PartyID, mpks map[PartyID][]PublicKey) error {
	for k, v := range mpks {
		var sij Key
		share := dkgSummary.SecretShares[k.GetHexString()]
		if share == "" {
			return common.NewError("failed to verify dkg summary", "share is nil")
		}
		if err := sij.SetHexString(share); err != nil {
			return err
		}
		if !ValidateShare(v, sij, id) {
			return common.NewError("failed to verify dkg summary", fmt.Sprintf("share unable to verify: %v", share))
		}
	}
	return nil
}

func (dkg *DKG) GetDKGSummary() *DKGSummary {
	dkgSummary := &DKGSummary{
		SecretShares:  make(map[string]string),
		StartingRound: dkg.StartingRound,
	}
	dkg.secretSharesMutex.RLock()
	defer dkg.secretSharesMutex.RUnlock()
	ids := make([]string, 0, len(dkg.receivedSecretShares))
	for k, v := range dkg.receivedSecretShares {
		dkgSummary.SecretShares[k.GetHexString()] = v.GetHexString()
		ids = append(ids, k.GetHexString())
	}
	dkgSummary.ID = strconv.FormatInt(dkg.MagicBlockNumber, 10)
	logging.Logger.Debug("[dkg] dkg_ss, get dkg summary", zap.Int("size", len(dkg.receivedSecretShares)))
	return dkgSummary
}
