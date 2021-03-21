package miner

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestRound_AddBlockToVerify(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
	}
	type args struct {
		b *block.Block
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
			//r := &Round{
			//	Round:                 tt.fields.Round,
			//	muVerification:        tt.fields.muVerification,
			//	blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
			//	verificationCancelf:   tt.fields.verificationCancelf,
			//	delta:                 tt.fields.delta,
			//	verificationTickets:   tt.fields.verificationTickets,
			//	vrfShare:              tt.fields.vrfShare,
			//}
		})
	}
}

func TestRound_AddVerificationTicket(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
	}
	type args struct {
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
			//r := &Round{
			//	Round:                 tt.fields.Round,
			//	muVerification:        tt.fields.muVerification,
			//	blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
			//	verificationCancelf:   tt.fields.verificationCancelf,
			//	delta:                 tt.fields.delta,
			//	verificationTickets:   tt.fields.verificationTickets,
			//	vrfShare:              tt.fields.vrfShare,
			//}
		})
	}
}

func TestRound_CancelVerification(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//r := &Round{
			//	Round:                 tt.fields.Round,
			//	muVerification:        tt.fields.muVerification,
			//	blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
			//	verificationCancelf:   tt.fields.verificationCancelf,
			//	delta:                 tt.fields.delta,
			//	verificationTickets:   tt.fields.verificationTickets,
			//	vrfShare:              tt.fields.vrfShare,
			//}
		})
	}
}

func TestRound_Clear(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//r := &Round{
			//	Round:                 tt.fields.Round,
			//	muVerification:        tt.fields.muVerification,
			//	blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
			//	verificationCancelf:   tt.fields.verificationCancelf,
			//	delta:                 tt.fields.delta,
			//	verificationTickets:   tt.fields.verificationTickets,
			//	vrfShare:              tt.fields.vrfShare,
			//}
		})
	}
}

func TestRound_GetBlocksToVerifyChannel(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
	}
	tests := []struct {
		name   string
		fields fields
		want   chan *block.Block
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Round{
				Round:                 tt.fields.Round,
				muVerification:        tt.fields.muVerification,
				blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
				verificationCancelf:   tt.fields.verificationCancelf,
				delta:                 tt.fields.delta,
				verificationTickets:   tt.fields.verificationTickets,
				vrfShare:              tt.fields.vrfShare,
			}
			if got := r.GetBlocksToVerifyChannel(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlocksToVerifyChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_GetVerificationTickets(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
	}
	type args struct {
		blockID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*block.VerificationTicket
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Round{
				Round:                 tt.fields.Round,
				muVerification:        tt.fields.muVerification,
				blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
				verificationCancelf:   tt.fields.verificationCancelf,
				delta:                 tt.fields.delta,
				verificationTickets:   tt.fields.verificationTickets,
				vrfShare:              tt.fields.vrfShare,
			}
			if got := r.GetVerificationTickets(tt.args.blockID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVerificationTickets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_IsVRFComplete(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
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
			r := &Round{
				Round:                 tt.fields.Round,
				muVerification:        tt.fields.muVerification,
				blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
				verificationCancelf:   tt.fields.verificationCancelf,
				delta:                 tt.fields.delta,
				verificationTickets:   tt.fields.verificationTickets,
				vrfShare:              tt.fields.vrfShare,
			}
			if got := r.IsVRFComplete(); got != tt.want {
				t.Errorf("IsVRFComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_IsVerificationComplete(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
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
			r := &Round{
				Round:                 tt.fields.Round,
				muVerification:        tt.fields.muVerification,
				blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
				verificationCancelf:   tt.fields.verificationCancelf,
				delta:                 tt.fields.delta,
				verificationTickets:   tt.fields.verificationTickets,
				vrfShare:              tt.fields.vrfShare,
			}
			if got := r.IsVerificationComplete(); got != tt.want {
				t.Errorf("IsVerificationComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_StartVerificationBlockCollection(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   context.Context
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Round{
				Round:                 tt.fields.Round,
				muVerification:        tt.fields.muVerification,
				blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
				verificationCancelf:   tt.fields.verificationCancelf,
				delta:                 tt.fields.delta,
				verificationTickets:   tt.fields.verificationTickets,
				vrfShare:              tt.fields.vrfShare,
			}
			if got := r.StartVerificationBlockCollection(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StartVerificationBlockCollection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_isVerificationComplete(t *testing.T) {
	type fields struct {
		Round                 *round.Round
		muVerification        sync.RWMutex
		blocksToVerifyChannel chan *block.Block
		verificationCancelf   context.CancelFunc
		delta                 time.Duration
		verificationTickets   map[string]*block.BlockVerificationTicket
		vrfShare              *round.VRFShare
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
			r := &Round{
				Round:                 tt.fields.Round,
				muVerification:        tt.fields.muVerification,
				blocksToVerifyChannel: tt.fields.blocksToVerifyChannel,
				verificationCancelf:   tt.fields.verificationCancelf,
				delta:                 tt.fields.delta,
				verificationTickets:   tt.fields.verificationTickets,
				vrfShare:              tt.fields.vrfShare,
			}
			if got := r.isVerificationComplete(); got != tt.want {
				t.Errorf("isVerificationComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}
