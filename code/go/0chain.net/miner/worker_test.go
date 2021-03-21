package miner

import (
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"context"
	"sync"
	"testing"
)

func TestChain_BlockWorker(t *testing.T) {
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
			//mc := &Chain{
			//	Chain:                                tt.fields.Chain,
			//	blockMessageChannel:                  tt.fields.blockMessageChannel,
			//	muDKG:                                tt.fields.muDKG,
			//	roundDkg:                             tt.fields.roundDkg,
			//	discoverClients:                      tt.fields.discoverClients,
			//	started:                              tt.fields.started,
			//	viewChangeProcess:                    tt.fields.viewChangeProcess,
			//	pullingPin:                           tt.fields.pullingPin,
			//	subRestartRoundEventChannel:          tt.fields.subRestartRoundEventChannel,
			//	unsubRestartRoundEventChannel:        tt.fields.unsubRestartRoundEventChannel,
			//	restartRoundEventChannel:             tt.fields.restartRoundEventChannel,
			//	restartRoundEventWorkerIsDoneChannel: tt.fields.restartRoundEventWorkerIsDoneChannel,
			//}
		})
	}
}

func TestChain_MinerHealthCheck(t *testing.T) {
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
			//mc := &Chain{
			//	Chain:                                tt.fields.Chain,
			//	blockMessageChannel:                  tt.fields.blockMessageChannel,
			//	muDKG:                                tt.fields.muDKG,
			//	roundDkg:                             tt.fields.roundDkg,
			//	discoverClients:                      tt.fields.discoverClients,
			//	started:                              tt.fields.started,
			//	viewChangeProcess:                    tt.fields.viewChangeProcess,
			//	pullingPin:                           tt.fields.pullingPin,
			//	subRestartRoundEventChannel:          tt.fields.subRestartRoundEventChannel,
			//	unsubRestartRoundEventChannel:        tt.fields.unsubRestartRoundEventChannel,
			//	restartRoundEventChannel:             tt.fields.restartRoundEventChannel,
			//	restartRoundEventWorkerIsDoneChannel: tt.fields.restartRoundEventWorkerIsDoneChannel,
			//}
		})
	}
}

func TestChain_RestartRoundEventWorker(t *testing.T) {
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
			//mc := &Chain{
			//	Chain:                                tt.fields.Chain,
			//	blockMessageChannel:                  tt.fields.blockMessageChannel,
			//	muDKG:                                tt.fields.muDKG,
			//	roundDkg:                             tt.fields.roundDkg,
			//	discoverClients:                      tt.fields.discoverClients,
			//	started:                              tt.fields.started,
			//	viewChangeProcess:                    tt.fields.viewChangeProcess,
			//	pullingPin:                           tt.fields.pullingPin,
			//	subRestartRoundEventChannel:          tt.fields.subRestartRoundEventChannel,
			//	unsubRestartRoundEventChannel:        tt.fields.unsubRestartRoundEventChannel,
			//	restartRoundEventChannel:             tt.fields.restartRoundEventChannel,
			//	restartRoundEventWorkerIsDoneChannel: tt.fields.restartRoundEventWorkerIsDoneChannel,
			//}
		})
	}
}

func TestChain_RoundWorker(t *testing.T) {
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
			//mc := &Chain{
			//	Chain:                                tt.fields.Chain,
			//	blockMessageChannel:                  tt.fields.blockMessageChannel,
			//	muDKG:                                tt.fields.muDKG,
			//	roundDkg:                             tt.fields.roundDkg,
			//	discoverClients:                      tt.fields.discoverClients,
			//	started:                              tt.fields.started,
			//	viewChangeProcess:                    tt.fields.viewChangeProcess,
			//	pullingPin:                           tt.fields.pullingPin,
			//	subRestartRoundEventChannel:          tt.fields.subRestartRoundEventChannel,
			//	unsubRestartRoundEventChannel:        tt.fields.unsubRestartRoundEventChannel,
			//	restartRoundEventChannel:             tt.fields.restartRoundEventChannel,
			//	restartRoundEventWorkerIsDoneChannel: tt.fields.restartRoundEventWorkerIsDoneChannel,
			//}
		})
	}
}

func TestChain_getPruneCountRoundStorage(t *testing.T) {
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
	tests := []struct {
		name   string
		fields fields
		want   func(storage round.RoundStorage) int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func TestSetupWorkers(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
