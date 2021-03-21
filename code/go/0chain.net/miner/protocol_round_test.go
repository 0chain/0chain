package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestChain_AddNotarizedBlock(t *testing.T) {
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
		r   *Round
		b   *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
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
			if got := mc.AddNotarizedBlock(tt.args.ctx, tt.args.r, tt.args.b); got != tt.want {
				t.Errorf("AddNotarizedBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_AddToRoundVerification(t *testing.T) {
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
		mr  *Round
		b   *block.Block
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

func TestChain_CancelRoundVerification(t *testing.T) {
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
		r   *Round
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

func TestChain_CollectBlocksForVerification(t *testing.T) {
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
		r   *Round
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

func TestChain_GenerateRoundBlock(t *testing.T) {
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
		r   *Round
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *block.Block
		wantErr bool
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
			got, err := mc.GenerateRoundBlock(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRoundBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateRoundBlock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockProposalWaitTime(t *testing.T) {
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
		r round.RoundI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   time.Duration
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
			if got := mc.GetBlockProposalWaitTime(tt.args.r); got != tt.want {
				t.Errorf("GetBlockProposalWaitTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockToExtend(t *testing.T) {
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
		r   round.RoundI
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantBnb *block.Block
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
			if gotBnb := mc.GetBlockToExtend(tt.args.ctx, tt.args.r); !reflect.DeepEqual(gotBnb, tt.wantBnb) {
				t.Errorf("GetBlockToExtend() = %v, want %v", gotBnb, tt.wantBnb)
			}
		})
	}
}

func TestChain_GetLatestFinalizedBlockFromSharder(t *testing.T) {
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
		name    string
		fields  fields
		args    args
		wantFbs []*BlockConsensus
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
			if gotFbs := mc.GetLatestFinalizedBlockFromSharder(tt.args.ctx); !reflect.DeepEqual(gotFbs, tt.wantFbs) {
				t.Errorf("GetLatestFinalizedBlockFromSharder() = %v, want %v", gotFbs, tt.wantFbs)
			}
		})
	}
}

func TestChain_GetNextRoundTimeoutTime(t *testing.T) {
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
		want   int
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
			if got := mc.GetNextRoundTimeoutTime(tt.args.ctx); got != tt.want {
				t.Errorf("GetNextRoundTimeoutTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_HandleRoundTimeout(t *testing.T) {
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

func TestChain_LoadMagicBlocksAndDKG(t *testing.T) {
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

func TestChain_MergeNotarization(t *testing.T) {
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
		r   *Round
		b   *block.Block
		vts []*block.VerificationTicket
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

func TestChain_ProcessVerifiedTicket(t *testing.T) {
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
		r   *Round
		b   *block.Block
		vt  *block.VerificationTicket
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

func TestChain_RedoVrfShare(t *testing.T) {
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
		r   *Round
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
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
			if got := mc.RedoVrfShare(tt.args.ctx, tt.args.r); got != tt.want {
				t.Errorf("RedoVrfShare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_StartNextRound(t *testing.T) {
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
		r   *Round
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Round
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
			if got := mc.StartNextRound(tt.args.ctx, tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StartNextRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_SyncFetchFinalizedBlockFromSharders(t *testing.T) {
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
		hash string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantFb *block.Block
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
			if gotFb := mc.SyncFetchFinalizedBlockFromSharders(tt.args.ctx, tt.args.hash); !reflect.DeepEqual(gotFb, tt.wantFb) {
				t.Errorf("SyncFetchFinalizedBlockFromSharders() = %v, want %v", gotFb, tt.wantFb)
			}
		})
	}
}

func TestChain_VerifyRoundBlock(t *testing.T) {
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
		r   round.RoundI
		b   *block.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *block.BlockVerificationTicket
		wantErr bool
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
			got, err := mc.VerifyRoundBlock(tt.args.ctx, tt.args.r, tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyRoundBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VerifyRoundBlock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_WaitForActiveSharders(t *testing.T) {
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
		name    string
		fields  fields
		args    args
		wantErr bool
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
			if err := mc.WaitForActiveSharders(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("WaitForActiveSharders() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_addMyVRFShare(t *testing.T) {
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
		pr  *Round
		r   *Round
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

func TestChain_addToRoundVerification(t *testing.T) {
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
		mr  *Round
		b   *block.Block
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

func TestChain_bumpLFBTicket(t *testing.T) {
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
		lfbs *block.Block
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

func TestChain_checkBlockNotarization(t *testing.T) {
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
		r   *Round
		b   *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
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
			if got := mc.checkBlockNotarization(tt.args.ctx, tt.args.r, tt.args.b); got != tt.want {
				t.Errorf("checkBlockNotarization() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_computeBlockProposalDynamicWaitTime(t *testing.T) {
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
		r round.RoundI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   time.Duration
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
			if got := mc.computeBlockProposalDynamicWaitTime(tt.args.r); got != tt.want {
				t.Errorf("computeBlockProposalDynamicWaitTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_ensureDKG(t *testing.T) {
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
		mb  *block.Block
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

func TestChain_ensureLatestFinalizedBlock(t *testing.T) {
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
		name        string
		fields      fields
		args        args
		wantUpdated bool
		wantErr     bool
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
			gotUpdated, err := mc.ensureLatestFinalizedBlock(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureLatestFinalizedBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUpdated != tt.wantUpdated {
				t.Errorf("ensureLatestFinalizedBlock() gotUpdated = %v, want %v", gotUpdated, tt.wantUpdated)
			}
		})
	}
}

func TestChain_ensureLatestFinalizedBlocks(t *testing.T) {
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
		name        string
		fields      fields
		args        args
		wantUpdated bool
		wantErr     bool
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
			gotUpdated, err := mc.ensureLatestFinalizedBlocks(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureLatestFinalizedBlocks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUpdated != tt.wantUpdated {
				t.Errorf("ensureLatestFinalizedBlocks() gotUpdated = %v, want %v", gotUpdated, tt.wantUpdated)
			}
		})
	}
}

func TestChain_ensureState(t *testing.T) {
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
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantOk bool
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
			if gotOk := mc.ensureState(tt.args.ctx, tt.args.b); gotOk != tt.wantOk {
				t.Errorf("ensureState() = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestChain_finalizeRound(t *testing.T) {
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
		r   *Round
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

func TestChain_getOrCreateRound(t *testing.T) {
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
		rn  int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantMr *Round
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
			if gotMr := mc.getOrCreateRound(tt.args.ctx, tt.args.rn); !reflect.DeepEqual(gotMr, tt.wantMr) {
				t.Errorf("getOrCreateRound() = %v, want %v", gotMr, tt.wantMr)
			}
		})
	}
}

func TestChain_getOrStartRound(t *testing.T) {
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
		rn  int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantMr *Round
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
			if gotMr := mc.getOrStartRound(tt.args.ctx, tt.args.rn); !reflect.DeepEqual(gotMr, tt.wantMr) {
				t.Errorf("getOrStartRound() = %v, want %v", gotMr, tt.wantMr)
			}
		})
	}
}

func TestChain_getOrStartRoundNotAhead(t *testing.T) {
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
		rn  int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantMr *Round
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
			if gotMr := mc.getOrStartRoundNotAhead(tt.args.ctx, tt.args.rn); !reflect.DeepEqual(gotMr, tt.wantMr) {
				t.Errorf("getOrStartRoundNotAhead() = %v, want %v", gotMr, tt.wantMr)
			}
		})
	}
}

func TestChain_getRoundRandomSeed(t *testing.T) {
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
		rn int64
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantSeed int64
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
			if gotSeed := mc.getRoundRandomSeed(tt.args.rn); gotSeed != tt.wantSeed {
				t.Errorf("getRoundRandomSeed() = %v, want %v", gotSeed, tt.wantSeed)
			}
		})
	}
}

func TestChain_handleNoProgress(t *testing.T) {
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

func TestChain_isAheadOfSharders(t *testing.T) {
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
		ctx   context.Context
		round int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
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
			if got := mc.isAheadOfSharders(tt.args.ctx, tt.args.round); got != tt.want {
				t.Errorf("isAheadOfSharders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_kickFinalization(t *testing.T) {
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

func TestChain_kickRoundByLFB(t *testing.T) {
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
		lfb *block.Block
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

func TestChain_kickSharders(t *testing.T) {
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

func TestChain_pullNotarizedBlocks(t *testing.T) {
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
		r   *Round
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

func TestChain_restartRound(t *testing.T) {
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

func TestChain_setupLoadedMagicBlock(t *testing.T) {
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
		mb *block.MagicBlock
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
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
			if err := mc.setupLoadedMagicBlock(tt.args.mb); (err != nil) != tt.wantErr {
				t.Errorf("setupLoadedMagicBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_startNewRound(t *testing.T) {
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
		mr  *Round
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

func TestChain_startNextRoundInRestartRound(t *testing.T) {
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
		i   int64
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

func TestChain_startNextRoundNotAhead(t *testing.T) {
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
		r   *Round
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

func TestChain_startProtocolOnLFB(t *testing.T) {
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
		lfb *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantMr *Round
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
			if gotMr := mc.startProtocolOnLFB(tt.args.ctx, tt.args.lfb); !reflect.DeepEqual(gotMr, tt.wantMr) {
				t.Errorf("startProtocolOnLFB() = %v, want %v", gotMr, tt.wantMr)
			}
		})
	}
}

func TestChain_startRound(t *testing.T) {
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
		r    *Round
		seed int64
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

func TestChain_updatePriorBlock(t *testing.T) {
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
		r   round.RoundI
		b   *block.Block
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

func TestChain_waitNotAhead(t *testing.T) {
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
		ctx   context.Context
		round int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantOk bool
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
			if gotOk := mc.waitNotAhead(tt.args.ctx, tt.args.round); gotOk != tt.wantOk {
				t.Errorf("waitNotAhead() = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestSetNetworkRelayTime(t *testing.T) {
	type args struct {
		delta time.Duration
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

func TestStartProtocol(t *testing.T) {
	type args struct {
		ctx context.Context
		gb  *block.Block
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

func Test_isNilRound(t *testing.T) {
	type args struct {
		r round.RoundI
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNilRound(tt.args.r); got != tt.want {
				t.Errorf("isNilRound() = %v, want %v", got, tt.want)
			}
		})
	}
}
