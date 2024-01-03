package common

import "0chain.net/chaincore/chain/state"

func WithActivation(ctx state.StateContextI, round int64, before func(), after func()) {
	if ctx.GetBlock().Round < round {
		before()
	} else {
		after()
	}
}
