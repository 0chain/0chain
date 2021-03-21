package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"context"
	"reflect"
	"sync"
	"testing"
)

func TestChain_AddVRFShare(t *testing.T) {
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
		mr   *Round
		vrfs *round.VRFShare
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
			if got := mc.AddVRFShare(tt.args.ctx, tt.args.mr, tt.args.vrfs); got != tt.want {
				t.Errorf("AddVRFShare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlsMessageForRound(t *testing.T) {
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
		r *round.Round
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
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
			got, err := mc.GetBlsMessageForRound(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlsMessageForRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBlsMessageForRound() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlsShare(t *testing.T) {
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
		r   *round.Round
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
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
			got, err := mc.GetBlsShare(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlsShare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBlsShare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlsThreshold(t *testing.T) {
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
		round int64
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
			if got := mc.GetBlsThreshold(tt.args.round); got != tt.want {
				t.Errorf("GetBlsThreshold() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_SetDKGSFromStore(t *testing.T) {
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
		mb  *block.MagicBlock
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
			if err := mc.SetDKGSFromStore(tt.args.ctx, tt.args.mb); (err != nil) != tt.wantErr {
				t.Errorf("SetDKGSFromStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_ThresholdNumBLSSigReceived(t *testing.T) {
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
		ctx          context.Context
		mr           *Round
		blsThreshold int
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

func TestChain_computeRBO(t *testing.T) {
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
		rbo string
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

func TestChain_computeRoundRandomSeed(t *testing.T) {
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
		pr  round.RoundI
		r   *Round
		rbo string
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

func TestComputeBlsID(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeBlsID(tt.args.key); got != tt.want {
				t.Errorf("ComputeBlsID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetDKG(t *testing.T) {
	type args struct {
		ctx context.Context
		mb  *block.MagicBlock
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetDKG(tt.args.ctx, tt.args.mb); (err != nil) != tt.wantErr {
				t.Errorf("SetDKG() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetDKGFromMagicBlocksChainPrev(t *testing.T) {
	type args struct {
		ctx context.Context
		mb  *block.MagicBlock
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetDKGFromMagicBlocksChainPrev(tt.args.ctx, tt.args.mb); (err != nil) != tt.wantErr {
				t.Errorf("SetDKGFromMagicBlocksChainPrev() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifySigShares(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifySigShares(); got != tt.want {
				t.Errorf("VerifySigShares() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getVRFShareInfo(t *testing.T) {
	type args struct {
		mr *Round
	}
	tests := []struct {
		name  string
		args  args
		want  []string
		want1 []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getVRFShareInfo(tt.args.mr)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getVRFShareInfo() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("getVRFShareInfo() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
