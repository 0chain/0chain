package cases

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"0chain.net/core/common"
)

type ChallengeResponse struct {
	ID                string              `json:"challenge_id"`
	ValidationTickets []*ValidationTicket `json:"validation_tickets"`
}

type ValidationTicket struct {
	ChallengeID  string           `json:"challenge_id"`
	BlobberID    string           `json:"blobber_id"`
	ValidatorID  string           `json:"validator_id"`
	ValidatorKey string           `json:"validator_key"`
	Result       bool             `json:"success"`
	Message      string           `json:"message"`
	MessageCode  string           `json:"message_code"`
	Timestamp    common.Timestamp `json:"timestamp"`
	Signature    string           `json:"signature"`
}

type (
	// CheckChallengeIsValid represents implementation of the TestCase interface.
	//
	//	Flow of this test case:
	//		1. Miner generates a challenge that requires more than 2 validation tickets
	//    2. Validators validate challenge. One of the validators is adversarial and will set the challenge as invalid
	//		2. Check: Challenge must pass because most of the validators validated the challenge
	CheckChallengeIsValid struct {
		challengeResult *ChallengeResponse

		wg *sync.WaitGroup

		resultLocker sync.RWMutex
	}
)

var (
	// Ensure CheckChallengeIsValid implements TestCase interface.
	_ TestCase = (*CheckChallengeIsValid)(nil)
)

// NewCheckChallengeIsValid creates initialised CheckChallengeIsValid.
func NewCheckChallengeIsValid() *CheckChallengeIsValid {
	wg := new(sync.WaitGroup)

	wg.Add(1) // miners num = number of results
	return &CheckChallengeIsValid{
		wg: wg,
	}
}

// Check implements TestCase interface.
func (n *CheckChallengeIsValid) Check(ctx context.Context) (success bool, err error) {
	prepared := make(chan struct{})
	go func() {
		n.wg.Wait()
		prepared <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return false, errors.New("cases state is not prepared, context is done")

	case <-prepared:
		return true, nil // when received challenge result
	}
}

// Configure implements TestCase interface.
func (n *CheckChallengeIsValid) Configure(blob []byte) error {
	return nil
}

// AddResult implements TestCase interface.
// When miners nodes got challenge validated, they report the challenge result.
func (n *CheckChallengeIsValid) AddResult(blob []byte) error {
	n.resultLocker.Lock()
	defer n.resultLocker.Unlock()

	res := new(ChallengeResponse)
	if err := json.Unmarshal(blob, res); err != nil {
		return err
	}

	if n.challengeResult == nil {
		defer n.wg.Done()
		n.challengeResult = res
	}

	return nil
}
