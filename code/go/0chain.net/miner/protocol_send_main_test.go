package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"context"
	"sync"
	"testing"
)

func TestChain_SendVRFShare(t *testing.T) {
	type fields struct {
		Chain                                *chain.Chain
		blockMessageChannel                  chan *BlockMessage
		muDKG                                *sync.RWMutex
		roundDkg                             round.RoundStorage
		discoverClients                      bool
		started                              uint32
		viewChangeProcess                    viewChangeProcess
		pullingPin                           int64
		subRestartRoundEventChannel          chan chan struct{}
		unsubRestartRoundEventChannel        chan chan struct{}
		restartRoundEventChannel             chan struct{}
		restartRoundEventWorkerIsDoneChannel chan struct{}
	}
	type args struct {
		ctx  context.Context
		vrfs *round.VRFShare
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &Chain{
				Chain:                                tt.fields.Chain,
				blockMessageChannel:                  tt.fields.blockMessageChannel,
				muDKG:                                tt.fields.muDKG,
				roundDkg:                             tt.fields.roundDkg,
				discoverClients:                      tt.fields.discoverClients,
				started:                              tt.fields.started,
				viewChangeProcess:                    tt.fields.viewChangeProcess,
				pullingPin:                           tt.fields.pullingPin,
				subRestartRoundEventChannel:          tt.fields.subRestartRoundEventChannel,
				unsubRestartRoundEventChannel:        tt.fields.unsubRestartRoundEventChannel,
				restartRoundEventChannel:             tt.fields.restartRoundEventChannel,
				restartRoundEventWorkerIsDoneChannel: tt.fields.restartRoundEventWorkerIsDoneChannel,
			}
		})
	}
}

func TestChain_SendVerificationTicket(t *testing.T) {
	type fields struct {
		Chain                                *chain.Chain
		blockMessageChannel                  chan *BlockMessage
		muDKG                                *sync.RWMutex
		roundDkg                             round.RoundStorage
		discoverClients                      bool
		started                              uint32
		viewChangeProcess                    viewChangeProcess
		pullingPin                           int64
		subRestartRoundEventChannel          chan chan struct{}
		unsubRestartRoundEventChannel        chan chan struct{}
		restartRoundEventChannel             chan struct{}
		restartRoundEventWorkerIsDoneChannel chan struct{}
	}
	type args struct {
		ctx context.Context
		b   *block.Block
		bvt *block.BlockVerificationTicket
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &Chain{
				Chain:                                tt.fields.Chain,
				blockMessageChannel:                  tt.fields.blockMessageChannel,
				muDKG:                                tt.fields.muDKG,
				roundDkg:                             tt.fields.roundDkg,
				discoverClients:                      tt.fields.discoverClients,
				started:                              tt.fields.started,
				viewChangeProcess:                    tt.fields.viewChangeProcess,
				pullingPin:                           tt.fields.pullingPin,
				subRestartRoundEventChannel:          tt.fields.subRestartRoundEventChannel,
				unsubRestartRoundEventChannel:        tt.fields.unsubRestartRoundEventChannel,
				restartRoundEventChannel:             tt.fields.restartRoundEventChannel,
				restartRoundEventWorkerIsDoneChannel: tt.fields.restartRoundEventWorkerIsDoneChannel,
			}
		})
	}
}
