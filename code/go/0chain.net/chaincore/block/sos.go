package block

import (
	"encoding/json"
	"runtime"
	"sort"
	"sync"

	"github.com/0chain/common/core/logging"

	"go.uber.org/zap"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/encryption"
)

//go:generate msgp -io=false -tests=false -v

type ShareOrSigns struct {
	ID           string                      `json:"id"`
	ShareOrSigns map[string]*bls.DKGKeyShare `json:"share_or_sign"`
}

func NewShareOrSigns() *ShareOrSigns {
	return &ShareOrSigns{ShareOrSigns: make(map[string]*bls.DKGKeyShare)}
}

func (sos *ShareOrSigns) Hash() string {
	data := sos.ID
	var keys []string
	for k := range sos.ShareOrSigns {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		data += string(sos.ShareOrSigns[k].Encode())
	}
	return encryption.Hash(data)
}

func (sos *ShareOrSigns) Validate(mpks *Mpks, publicKeys map[string]string, scheme encryption.SignatureScheme) ([]string, bool) {
	if len(sos.ShareOrSigns) == 0 {
		return nil, true
	}

	type validationResult struct {
		key     string
		valid   bool
		isShare bool
	}

	type job struct {
		key   string
		share *bls.DKGKeyShare
	}

	// Number of concurrent workers
	numWorkers := runtime.NumCPU()
	if numWorkers > len(sos.ShareOrSigns) {
		numWorkers = len(sos.ShareOrSigns)
	}

	// Create work and result channels
	jobs := make(chan job, len(sos.ShareOrSigns))
	results := make(chan validationResult, len(sos.ShareOrSigns))

	// Start worker goroutines
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				key := job.key
				share := job.share

				if share == nil {
					continue
				}

				result := validationResult{key: key, valid: false, isShare: false}

				if share.Sign != "" {
					// Create a new signature scheme instance to avoid concurrent access issues
					// We need to use the same type as the input scheme
					signatureScheme := scheme
					pk, ok := publicKeys[key]
					if !ok {
						results <- result
						continue
					}
					if err := signatureScheme.SetPublicKey(pk); err != nil {
						logging.Logger.Error("failed to validate share or signs",
							zap.Any("share", share),
							zap.String("message", share.Message),
							zap.String("sign", share.Sign))
						results <- result
						continue
					}
					sigOK, err := signatureScheme.Verify(share.Sign, share.Message)
					if !sigOK || err != nil {
						logging.Logger.Error("failed to validate share or signs",
							zap.Any("share", share),
							zap.String("message", share.Message),
							zap.String("sign", share.Sign))
						results <- result
						continue
					}
					result.valid = true
				} else {
					var sij bls.Key
					if err := sij.SetHexString(share.Share); err != nil {
						results <- result
						continue
					}

					// Slightly inefficient to convert MPK for each worker, but safer than concurrent access
					pks, err := bls.ConvertStringToMpk(mpks.Mpks[sos.ID].Mpk)
					if err != nil {
						logging.Logger.Error("failed to convert mpks", zap.Error(err))
						results <- result
						continue
					}

					if !bls.ValidateShare(pks, sij, bls.ComputeIDdkg(key)) {
						logging.Logger.Error("failed to validate share or signs",
							zap.Any("share", share),
							zap.String("sij.pi", sij.GetPublicKey().GetHexString()))
						results <- result
						continue
					}
					result.valid = true
					result.isShare = true
				}
				results <- result
			}
		}()
	}

	// Send jobs to workers
	for key, share := range sos.ShareOrSigns {
		jobs <- job{key, share}
	}
	close(jobs)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	var keys []string
	validShareCount := 0
	for result := range results {
		if !result.valid {
			// If any validation fails, abort and return false
			return nil, false
		}
		if result.isShare {
			keys = append(keys, result.key)
		}
		validShareCount++
	}

	// Make sure all validations were successful
	if validShareCount != len(sos.ShareOrSigns) {
		return nil, false
	}

	return keys, true
}

func (sos *ShareOrSigns) Encode() []byte {
	buff, _ := json.Marshal(sos)
	return buff
}

func (sos *ShareOrSigns) Decode(input []byte) error {
	return json.Unmarshal(input, sos)
}

func (sos *ShareOrSigns) Clone() *ShareOrSigns {
	clone := &ShareOrSigns{
		ID:           sos.ID,
		ShareOrSigns: make(map[string]*bls.DKGKeyShare, len(sos.ShareOrSigns)),
	}
	for key, dkg := range sos.ShareOrSigns {
		clone.ShareOrSigns[key] = &bls.DKGKeyShare{
			IDField: dkg.IDField,
			Message: dkg.Message,
			Share:   dkg.Share,
			Sign:    dkg.Sign,
		}
	}
	return clone
}

func (sos *ShareOrSigns) ValidateV2(publicKeys map[string]string) ([]string, bool) {
	if len(sos.ShareOrSigns) == 0 {
		return nil, true
	}

	type validationResult struct {
		key     string
		valid   bool
		isShare bool
	}

	type job struct {
		key   string
		share *bls.DKGKeyShare
	}

	// Number of concurrent workers
	numWorkers := runtime.NumCPU()
	if numWorkers > len(sos.ShareOrSigns) {
		numWorkers = len(sos.ShareOrSigns)
	}

	// Create work and result channels
	jobs := make(chan job, len(sos.ShareOrSigns))
	results := make(chan validationResult, len(sos.ShareOrSigns))

	// Start worker goroutines
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				key := job.key
				share := job.share

				if share == nil {
					continue
				}

				result := validationResult{key: key, valid: false, isShare: false}

				if share.Sign != "" {
					// Create a new signature scheme instance to avoid concurrent access issues
					// We need to use the same type as the input scheme
					signatureScheme := encryption.GetSignatureScheme(encryption.SignatureSchemeBls0chain)
					pk, ok := publicKeys[key]
					if !ok {
						logging.Logger.Error("could not find public key in public keys map", zap.String("key", key))
						results <- result
						continue
					}
					if err := signatureScheme.SetPublicKey(pk); err != nil {
						logging.Logger.Error("failed to set public key",
							zap.Any("share", share),
							zap.String("message", share.Message),
							zap.String("sign", share.Sign))
						results <- result
						continue
					}
					sigOK, err := signatureScheme.Verify(share.Sign, share.Message)
					if !sigOK || err != nil {
						logging.Logger.Error("failed to validate share or signs",
							zap.Any("share", share),
							zap.String("message", share.Message),
							zap.String("sign", share.Sign))
						results <- result
						continue
					}
					result.valid = true
				}
				results <- result
			}
		}()
	}

	// Send jobs to workers
	for key, share := range sos.ShareOrSigns {
		jobs <- job{key, share}
	}
	close(jobs)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	var keys []string
	validShareCount := 0
	for result := range results {
		if !result.valid {
			// If any validation fails, abort and return false
			return nil, false
		}
		if result.isShare {
			keys = append(keys, result.key)
		}
		validShareCount++
	}

	// Make sure all validations were successful
	if validShareCount != len(sos.ShareOrSigns) {
		return nil, false
	}

	return keys, true
}
