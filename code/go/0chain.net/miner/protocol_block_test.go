package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"context"
	"reflect"
	"sync"
	"testing"
)

func TestChain_FinalizeBlock(t *testing.T) {
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
			if err := mc.FinalizeBlock(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("FinalizeBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_NotarizedBlockFetched(t *testing.T) {
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

func TestChain_UpdateFinalizedBlock(t *testing.T) {
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

func TestChain_UpdatePendingBlock(t *testing.T) {
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
		b    *block.Block
		txns []datastore.Entity
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

func TestChain_ValidateTransactions(t *testing.T) {
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
			if err := mc.ValidateTransactions(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransactions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_VerifyBlock(t *testing.T) {
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
		name    string
		fields  fields
		args    args
		wantBvt *block.BlockVerificationTicket
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
			gotBvt, err := mc.VerifyBlock(tt.args.ctx, tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotBvt, tt.wantBvt) {
				t.Errorf("VerifyBlock() gotBvt = %v, want %v", gotBvt, tt.wantBvt)
			}
		})
	}
}

func TestChain_VerifyBlockMagicBlock(t *testing.T) {
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
			if err := mc.VerifyBlockMagicBlock(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("VerifyBlockMagicBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_VerifyBlockMagicBlockReference(t *testing.T) {
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
		b *block.Block
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
			if err := mc.VerifyBlockMagicBlockReference(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("VerifyBlockMagicBlockReference() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_createFeeTxn(t *testing.T) {
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
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *transaction.Transaction
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
			if got := mc.createFeeTxn(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createFeeTxn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_processFeeTxn(t *testing.T) {
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
		ctx     context.Context
		b       *block.Block
		clients map[string]*client.Client
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
			if err := mc.processFeeTxn(tt.args.ctx, tt.args.b, tt.args.clients); (err != nil) != tt.wantErr {
				t.Errorf("processFeeTxn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_signBlock(t *testing.T) {
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
			got, err := mc.signBlock(tt.args.ctx, tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("signBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("signBlock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_txnToReuse(t *testing.T) {
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
		txn *transaction.Transaction
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *transaction.Transaction
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
			if got := mc.txnToReuse(tt.args.txn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("txnToReuse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_validateTransaction(t *testing.T) {
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
		b   *block.Block
		txn *transaction.Transaction
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
			if got := mc.validateTransaction(tt.args.b, tt.args.txn); got != tt.want {
				t.Errorf("validateTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_verifySmartContracts(t *testing.T) {
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
			if err := mc.verifySmartContracts(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("verifySmartContracts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getLatestBlockFromSharders(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *block.Block
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getLatestBlockFromSharders(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getLatestBlockFromSharders() = %v, want %v", got, tt.want)
			}
		})
	}
}
