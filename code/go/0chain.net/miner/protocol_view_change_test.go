package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/round"
	"0chain.net/smartcontract/minersc"
	"context"
	"github.com/herumi/bls/ffi/go/bls"
	"reflect"
	"sync"
	"testing"
)

func TestChain_DKGProcess(t *testing.T) {
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

func TestChain_DKGProcessStart(t *testing.T) {
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
		in0 context.Context
		in1 *block.Block
		in2 *block.MagicBlock
		in3 bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *httpclientutil.Transaction
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
			got, err := mc.DKGProcessStart(tt.args.in0, tt.args.in1, tt.args.in2, tt.args.in3)
			if (err != nil) != tt.wantErr {
				t.Errorf("DKGProcessStart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DKGProcessStart() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetMagicBlockFromSC(t *testing.T) {
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
		lfb    *block.Block
		mb     *block.MagicBlock
		active bool
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantMagicBlock *block.MagicBlock
		wantErr        bool
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
			gotMagicBlock, err := mc.GetMagicBlockFromSC(tt.args.lfb, tt.args.mb, tt.args.active)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMagicBlockFromSC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotMagicBlock, tt.wantMagicBlock) {
				t.Errorf("GetMagicBlockFromSC() gotMagicBlock = %v, want %v", gotMagicBlock, tt.wantMagicBlock)
			}
		})
	}
}

func TestChain_NextViewChangeOfBlock(t *testing.T) {
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
		lfb *block.Block
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantRound int64
		wantErr   bool
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
			gotRound, err := mc.NextViewChangeOfBlock(tt.args.lfb)
			if (err != nil) != tt.wantErr {
				t.Errorf("NextViewChangeOfBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRound != tt.wantRound {
				t.Errorf("NextViewChangeOfBlock() gotRound = %v, want %v", gotRound, tt.wantRound)
			}
		})
	}
}

func TestChain_SendSijs(t *testing.T) {
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
		ctx    context.Context
		lfb    *block.Block
		mb     *block.MagicBlock
		active bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantTx  *httpclientutil.Transaction
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
			gotTx, err := mc.SendSijs(tt.args.ctx, tt.args.lfb, tt.args.mb, tt.args.active)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendSijs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTx, tt.wantTx) {
				t.Errorf("SendSijs() gotTx = %v, want %v", gotTx, tt.wantTx)
			}
		})
	}
}

func TestChain_SetupLatestAndPreviousMagicBlocks(t *testing.T) {
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

func TestChain_Wait(t *testing.T) {
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
		ctx    context.Context
		lfb    *block.Block
		mb     *block.MagicBlock
		active bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantTx  *httpclientutil.Transaction
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
			gotTx, err := mc.Wait(tt.args.ctx, tt.args.lfb, tt.args.mb, tt.args.active)
			if (err != nil) != tt.wantErr {
				t.Errorf("Wait() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTx, tt.wantTx) {
				t.Errorf("Wait() gotTx = %v, want %v", gotTx, tt.wantTx)
			}
		})
	}
}

func TestChain_createSijs(t *testing.T) {
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
		lfb    *block.Block
		mb     *block.MagicBlock
		active bool
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
			if err := mc.createSijs(tt.args.lfb, tt.args.mb, tt.args.active); (err != nil) != tt.wantErr {
				t.Errorf("createSijs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_getDKGMiners(t *testing.T) {
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
		lfb    *block.Block
		mb     *block.MagicBlock
		active bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantDmn *minersc.DKGMinerNodes
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
			gotDmn, err := mc.getDKGMiners(tt.args.lfb, tt.args.mb, tt.args.active)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDKGMiners() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotDmn, tt.wantDmn) {
				t.Errorf("getDKGMiners() gotDmn = %v, want %v", gotDmn, tt.wantDmn)
			}
		})
	}
}

func TestChain_getMinersMpks(t *testing.T) {
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
		lfb    *block.Block
		mb     *block.MagicBlock
		active bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantMpks *block.Mpks
		wantErr  bool
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
			gotMpks, err := mc.getMinersMpks(tt.args.lfb, tt.args.mb, tt.args.active)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMinersMpks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotMpks, tt.wantMpks) {
				t.Errorf("getMinersMpks() gotMpks = %v, want %v", gotMpks, tt.wantMpks)
			}
		})
	}
}

func TestChain_getNodeSij(t *testing.T) {
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
		nodeID bls.ID
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantSecShare bls.SecretKey
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
			if gotSecShare := mc.getNodeSij(tt.args.nodeID); !reflect.DeepEqual(gotSecShare, tt.wantSecShare) {
				t.Errorf("getNodeSij() = %v, want %v", gotSecShare, tt.wantSecShare)
			}
		})
	}
}

func TestChain_sendSijsPrepare(t *testing.T) {
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
		ctx    context.Context
		lfb    *block.Block
		mb     *block.MagicBlock
		active bool
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantSendTo []string
		wantErr    bool
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
			gotSendTo, err := mc.sendSijsPrepare(tt.args.ctx, tt.args.lfb, tt.args.mb, tt.args.active)
			if (err != nil) != tt.wantErr {
				t.Errorf("sendSijsPrepare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotSendTo, tt.wantSendTo) {
				t.Errorf("sendSijsPrepare() gotSendTo = %v, want %v", gotSendTo, tt.wantSendTo)
			}
		})
	}
}

func TestChain_setSecretShares(t *testing.T) {
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
		shareOrSignSuccess map[string]*bls.DKGKeyShare
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

func TestChain_updateMagicBlocks(t *testing.T) {
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
		mbs []*block.Block
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

func TestChain_waitTransaction(t *testing.T) {
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
		wantTx  *httpclientutil.Transaction
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
			gotTx, err := mc.waitTransaction(tt.args.mb)
			if (err != nil) != tt.wantErr {
				t.Errorf("waitTransaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTx, tt.wantTx) {
				t.Errorf("waitTransaction() gotTx = %v, want %v", gotTx, tt.wantTx)
			}
		})
	}
}

func TestLoadDKGSummary(t *testing.T) {
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name     string
		args     args
		wantDkgs *bls.DKGSummary
		wantErr  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDkgs, err := LoadDKGSummary(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadDKGSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotDkgs, tt.wantDkgs) {
				t.Errorf("LoadDKGSummary() gotDkgs = %v, want %v", gotDkgs, tt.wantDkgs)
			}
		})
	}
}

func TestLoadLatestMB(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantMb  *block.MagicBlock
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMb, err := LoadLatestMB(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadLatestMB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotMb, tt.wantMb) {
				t.Errorf("LoadLatestMB() gotMb = %v, want %v", gotMb, tt.wantMb)
			}
		})
	}
}

func TestLoadMagicBlock(t *testing.T) {
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		args    args
		wantMb  *block.MagicBlock
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMb, err := LoadMagicBlock(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadMagicBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotMb, tt.wantMb) {
				t.Errorf("LoadMagicBlock() gotMb = %v, want %v", gotMb, tt.wantMb)
			}
		})
	}
}

func TestStoreDKG(t *testing.T) {
	type args struct {
		ctx context.Context
		dkg *bls.DKG
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
			if err := StoreDKG(tt.args.ctx, tt.args.dkg); (err != nil) != tt.wantErr {
				t.Errorf("StoreDKG() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStoreDKGSummary(t *testing.T) {
	type args struct {
		ctx     context.Context
		summary *bls.DKGSummary
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
			if err := StoreDKGSummary(tt.args.ctx, tt.args.summary); (err != nil) != tt.wantErr {
				t.Errorf("StoreDKGSummary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStoreMagicBlock(t *testing.T) {
	type args struct {
		ctx        context.Context
		magicBlock *block.MagicBlock
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
			if err := StoreMagicBlock(tt.args.ctx, tt.args.magicBlock); (err != nil) != tt.wantErr {
				t.Errorf("StoreMagicBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_viewChangeProcess_CurrentPhase(t *testing.T) {
	type fields struct {
		Mutex          sync.Mutex
		scFunctions    map[minersc.Phase]SmartContractFunctions
		currentPhase   minersc.Phase
		shareOrSigns   *block.ShareOrSigns
		mpks           *block.Mpks
		viewChangeDKG  *bls.DKG
		nvcmx          sync.RWMutex
		nextViewChange int64
	}
	tests := []struct {
		name   string
		fields fields
		want   minersc.Phase
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vcp := &viewChangeProcess{
				Mutex:          tt.fields.Mutex,
				scFunctions:    tt.fields.scFunctions,
				currentPhase:   tt.fields.currentPhase,
				shareOrSigns:   tt.fields.shareOrSigns,
				mpks:           tt.fields.mpks,
				viewChangeDKG:  tt.fields.viewChangeDKG,
				nvcmx:          tt.fields.nvcmx,
				nextViewChange: tt.fields.nextViewChange,
			}
			if got := vcp.CurrentPhase(); got != tt.want {
				t.Errorf("CurrentPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_viewChangeProcess_NextViewChange(t *testing.T) {
	type fields struct {
		Mutex          sync.Mutex
		scFunctions    map[minersc.Phase]SmartContractFunctions
		currentPhase   minersc.Phase
		shareOrSigns   *block.ShareOrSigns
		mpks           *block.Mpks
		viewChangeDKG  *bls.DKG
		nvcmx          sync.RWMutex
		nextViewChange int64
	}
	tests := []struct {
		name      string
		fields    fields
		wantRound int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vcp := &viewChangeProcess{
				Mutex:          tt.fields.Mutex,
				scFunctions:    tt.fields.scFunctions,
				currentPhase:   tt.fields.currentPhase,
				shareOrSigns:   tt.fields.shareOrSigns,
				mpks:           tt.fields.mpks,
				viewChangeDKG:  tt.fields.viewChangeDKG,
				nvcmx:          tt.fields.nvcmx,
				nextViewChange: tt.fields.nextViewChange,
			}
			if gotRound := vcp.NextViewChange(); gotRound != tt.wantRound {
				t.Errorf("NextViewChange() = %v, want %v", gotRound, tt.wantRound)
			}
		})
	}
}

func Test_viewChangeProcess_SetCurrentPhase(t *testing.T) {
	type fields struct {
		Mutex          sync.Mutex
		scFunctions    map[minersc.Phase]SmartContractFunctions
		currentPhase   minersc.Phase
		shareOrSigns   *block.ShareOrSigns
		mpks           *block.Mpks
		viewChangeDKG  *bls.DKG
		nvcmx          sync.RWMutex
		nextViewChange int64
	}
	type args struct {
		ph minersc.Phase
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
			vcp := &viewChangeProcess{
				Mutex:          tt.fields.Mutex,
				scFunctions:    tt.fields.scFunctions,
				currentPhase:   tt.fields.currentPhase,
				shareOrSigns:   tt.fields.shareOrSigns,
				mpks:           tt.fields.mpks,
				viewChangeDKG:  tt.fields.viewChangeDKG,
				nvcmx:          tt.fields.nvcmx,
				nextViewChange: tt.fields.nextViewChange,
			}
		})
	}
}

func Test_viewChangeProcess_SetNextViewChange(t *testing.T) {
	type fields struct {
		Mutex          sync.Mutex
		scFunctions    map[minersc.Phase]SmartContractFunctions
		currentPhase   minersc.Phase
		shareOrSigns   *block.ShareOrSigns
		mpks           *block.Mpks
		viewChangeDKG  *bls.DKG
		nvcmx          sync.RWMutex
		nextViewChange int64
	}
	type args struct {
		round int64
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
			vcp := &viewChangeProcess{
				Mutex:          tt.fields.Mutex,
				scFunctions:    tt.fields.scFunctions,
				currentPhase:   tt.fields.currentPhase,
				shareOrSigns:   tt.fields.shareOrSigns,
				mpks:           tt.fields.mpks,
				viewChangeDKG:  tt.fields.viewChangeDKG,
				nvcmx:          tt.fields.nvcmx,
				nextViewChange: tt.fields.nextViewChange,
			}
		})
	}
}

func Test_viewChangeProcess_clearViewChange(t *testing.T) {
	type fields struct {
		Mutex          sync.Mutex
		scFunctions    map[minersc.Phase]SmartContractFunctions
		currentPhase   minersc.Phase
		shareOrSigns   *block.ShareOrSigns
		mpks           *block.Mpks
		viewChangeDKG  *bls.DKG
		nvcmx          sync.RWMutex
		nextViewChange int64
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vcp := &viewChangeProcess{
				Mutex:          tt.fields.Mutex,
				scFunctions:    tt.fields.scFunctions,
				currentPhase:   tt.fields.currentPhase,
				shareOrSigns:   tt.fields.shareOrSigns,
				mpks:           tt.fields.mpks,
				viewChangeDKG:  tt.fields.viewChangeDKG,
				nvcmx:          tt.fields.nvcmx,
				nextViewChange: tt.fields.nextViewChange,
			}
		})
	}
}

func Test_viewChangeProcess_isDKGSet(t *testing.T) {
	type fields struct {
		Mutex          sync.Mutex
		scFunctions    map[minersc.Phase]SmartContractFunctions
		currentPhase   minersc.Phase
		shareOrSigns   *block.ShareOrSigns
		mpks           *block.Mpks
		viewChangeDKG  *bls.DKG
		nvcmx          sync.RWMutex
		nextViewChange int64
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vcp := &viewChangeProcess{
				Mutex:          tt.fields.Mutex,
				scFunctions:    tt.fields.scFunctions,
				currentPhase:   tt.fields.currentPhase,
				shareOrSigns:   tt.fields.shareOrSigns,
				mpks:           tt.fields.mpks,
				viewChangeDKG:  tt.fields.viewChangeDKG,
				nvcmx:          tt.fields.nvcmx,
				nextViewChange: tt.fields.nextViewChange,
			}
			if got := vcp.isDKGSet(); got != tt.want {
				t.Errorf("isDKGSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_viewChangeProcess_isNeedCreateSijs(t *testing.T) {
	type fields struct {
		Mutex          sync.Mutex
		scFunctions    map[minersc.Phase]SmartContractFunctions
		currentPhase   minersc.Phase
		shareOrSigns   *block.ShareOrSigns
		mpks           *block.Mpks
		viewChangeDKG  *bls.DKG
		nvcmx          sync.RWMutex
		nextViewChange int64
	}
	tests := []struct {
		name   string
		fields fields
		wantOk bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vcp := &viewChangeProcess{
				Mutex:          tt.fields.Mutex,
				scFunctions:    tt.fields.scFunctions,
				currentPhase:   tt.fields.currentPhase,
				shareOrSigns:   tt.fields.shareOrSigns,
				mpks:           tt.fields.mpks,
				viewChangeDKG:  tt.fields.viewChangeDKG,
				nvcmx:          tt.fields.nvcmx,
				nextViewChange: tt.fields.nextViewChange,
			}
			if gotOk := vcp.isNeedCreateSijs(); gotOk != tt.wantOk {
				t.Errorf("isNeedCreateSijs() = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}
