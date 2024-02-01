package state

import (
	"math"
)

//go:generate msgp -io=false -tests=false -unexported=true -v
type HardFork struct {
	name  string
	round int64
}

func NewHardFork(name string, round int64) *HardFork {
	return &HardFork{name: name, round: round}
}

func (h *HardFork) GetKey() string {
	return "hardfork:" + h.name

}

func GetRoundByName(c StateContextI, name string) (int64, error) {
	fork := NewHardFork(name, 0)
	err := c.GetTrieNode(fork.GetKey(), fork)
	if err != nil {
		return math.MaxInt64, err
	}

	return fork.round, nil
}

func WithActivation(ctx StateContextI, name string, before func() error, after func() error) error {
	round, err := GetRoundByName(ctx, name)
	if err != nil {
		return err
	}

	if ctx.GetBlock().Round < round {
		err = before()
	} else {
		err = after()
	}

	return err
}
