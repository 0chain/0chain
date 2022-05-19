package cases

import (
	"context"
	"errors"
	"strconv"
)

type (
	// RoundHasFinalized represents implementation of the TestCase interface.
	RoundHasFinalized struct {
		res *RoundInfo

		roundID int

		prepared chan struct{}
	}
)

var (
	// Ensure RoundHasFinalized implements TestCase interface.
	_ TestCase = (*RoundHasFinalized)(nil)
)

// NewRoundHasFinalized creates initialised RoundHasFinalized.
func NewRoundHasFinalized() *RoundHasFinalized {
	return &RoundHasFinalized{
		prepared: make(chan struct{}),
	}
}

// Check implements TestCase interface.
func (n *RoundHasFinalized) Check(ctx context.Context) (success bool, err error) {
	select {
	case <-ctx.Done():
		return false, errors.New("cases state is not prepared, context is done")

	case <-n.prepared:
		return n.check()
	}
}

func (n *RoundHasFinalized) check() (success bool, err error) {
	if !n.res.IsFinalised {
		return false, errors.New("expected the round to be finalised")
	}

	return true, nil
}

// Configure implements TestCase interface.
func (n *RoundHasFinalized) Configure(blob []byte) error {
	roundIDStr := string(blob)

	roundID, err := strconv.Atoi(roundIDStr)
	if err != nil {
		return err
	}

	n.roundID = roundID
	return nil
}

// AddResult implements TestCase interface.
func (n *RoundHasFinalized) AddResult(blob []byte) error {
	roundInfo := &RoundInfo{}

	if err := roundInfo.Decode(blob); err != nil {
		return err
	}

	if roundInfo.Num == int64(n.roundID) && (n.res == nil || n.res.Num == 0) {
		defer func() {
			n.prepared <- struct{}{}
		}()
		n.res = new(RoundInfo)
		return n.res.Decode(blob)
	}

	return nil
}
