package block

import (
	"encoding/json"
	"runtime"
	"sync"
	"time"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

//go:generate msgp -io=false -tests=false -v

type MPK struct {
	ID  string
	Mpk []string
}

// swagger:model Mpks
type Mpks struct {
	Mpks map[string]*MPK
}

func NewMpks() *Mpks {
	return &Mpks{Mpks: make(map[string]*MPK)}
}

func (mpks *Mpks) Delete(id string) {
	delete(mpks.Mpks, id)
}

func (mpks *Mpks) Encode() []byte {
	buff, _ := json.Marshal(mpks)
	return buff
}

func (mpks *Mpks) Decode(input []byte) error {
	err := json.Unmarshal(input, mpks)
	if err != nil {
		return err
	}
	return nil
}

func (mpks *Mpks) GetHash() string {
	return util.ToHex(mpks.GetHashBytes())
}

func (mpks *Mpks) GetHashBytes() []byte {
	return encryption.RawHash(mpks.Encode())
}

func (mpks *Mpks) GetMpkMap() (map[bls.PartyID][]bls.PublicKey, error) {
	mpkMap := make(map[bls.PartyID][]bls.PublicKey)
	for k, v := range mpks.Mpks {
		mpk, err := bls.ConvertStringToMpk(v.Mpk)
		if err != nil {
			return nil, err
		}

		mpkMap[bls.ComputeIDdkg(k)] = mpk
	}
	return mpkMap, nil
}

func (mpks *Mpks) GetMpkMapStrings() map[bls.PartyID][]string {
	result := make(map[bls.PartyID][]string, len(mpks.Mpks))
	for k, v := range mpks.Mpks {
		id := bls.ComputeIDdkg(k)
		result[id] = v.Mpk
	}
	return result
}

func (mpks *Mpks) GetMpkMapParallel() (map[bls.PartyID][]bls.PublicKey, error) {
	startTime := time.Now()
	defer func() {
		logging.Logger.Debug("[mpk_timing] Parallel MPK map conversion",
			zap.Duration("duration", time.Since(startTime)))
	}()

	result := make(map[bls.PartyID][]bls.PublicKey)
	var (
		numWorkers = runtime.NumCPU()
		wg         sync.WaitGroup
		mu         sync.Mutex
		errChan    = make(chan error, 1)
	)

	// Create work channel
	workChan := make(chan struct {
		key string
		mpk *MPK
	}, len(mpks.Mpks))

	// Feed work channel
	for k, v := range mpks.Mpks {
		workChan <- struct {
			key string
			mpk *MPK
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
				pks, err := bls.ConvertStringToMpk(work.mpk.Mpk)
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}

				// Convert key to PartyID
				id := bls.ComputeIDdkg(work.key)
				// Safely store result
				mu.Lock()
				result[id] = pks
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

func (mpks *Mpks) GetMpks() map[string]*MPK {
	result := make(map[string]*MPK, len(mpks.Mpks))
	for k, v := range mpks.Mpks {
		result[k] = v
	}
	return result
}

// Clone returns a clone of Mpks instance
func (mpks *Mpks) Clone() *Mpks {
	clone := &Mpks{Mpks: make(map[string]*MPK, len(mpks.Mpks))}
	for k, v := range mpks.Mpks {
		nv := *v
		nv.Mpk = make([]string, len(v.Mpk))
		copy(nv.Mpk, v.Mpk)
		clone.Mpks[k] = &nv
	}

	return clone
}

func (mpk *MPK) Encode() []byte {
	buff, _ := json.Marshal(mpk)
	return buff
}

func (mpk *MPK) Decode(input []byte) error {
	err := json.Unmarshal(input, mpk)
	if err != nil {
		return err
	}
	return nil
}
