package datastore_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
)

func makeTestChunkProcessors() []datastore.ChunkProcessor {
	bchannel := make(chan datastore.Chunk, 5)

	bworkers := make([]datastore.ChunkProcessor, 5)
	for i := 0; i < len(bworkers); i++ {
		bworker := &memorystore.MemoryDBChunkProcessor{}
		bworker.EntityMetadata = datastore.GetEntityMetadata("block")
		bworker.ChunkChannel = bchannel
		bworkers[i] = bworker
	}

	return bworkers
}

func makeTestEntityChunkBuilder() datastore.EntityChunkBuilder {
	bchannel := make(chan datastore.Chunk, 5)

	var ecb datastore.EntityChunkBuilder
	ecb.Chunk = &memorystore.MemoryDBChunk{Length: 1}
	ecb.ChunkProvider = &memorystore.MemoryDBChunkProvider{}
	ecb.ChunkSize = 200
	ecb.MaxHoldupTime = time.Millisecond
	ecb.EntityChannel = make(chan datastore.QueuedEntity, 5)
	ecb.ChunkChannel = bchannel
	ecb.TimeoutChannel = time.NewTimer(-100 * time.Second)

	return ecb
}

func TestSetupChunkWorkers(t *testing.T) {
	ecb, bworkers := makeTestEntityChunkBuilder(), makeTestChunkProcessors()
	datastore.SetupChunkWorkers(context.TODO(), ecb, bworkers)
}

func TestWithAsyncChannel(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	ch := make(chan<- datastore.QueuedEntity)

	type args struct {
		ctx     context.Context
		channel chan<- datastore.QueuedEntity
	}
	tests := []struct {
		name string
		args args
		want context.Context
	}{
		{
			name: "Test_WithAsyncChannel_OK",
			args: args{
				ctx:     ctx,
				channel: ch,
			},
			want: context.WithValue(ctx, datastore.ASYNC_CHANNEL, ch),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := datastore.WithAsyncChannel(tt.args.ctx, tt.args.channel); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithAsyncChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsyncChannel(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	ch := make(chan<- datastore.QueuedEntity)
	ctx = datastore.WithAsyncChannel(ctx, ch)

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want chan<- datastore.QueuedEntity
	}{
		{
			name: "Test_AsyncChannel_OK",
			args: args{ctx: ctx},
			want: ch,
		},
		{
			name: "Test_AsyncChannel_No_Value_In_Context_OK",
			args: args{ctx: context.TODO()},
			want: nil,
		},
		{
			name: "Test_AsyncChannel_Not_A_Channel_OK",
			args: args{ctx: context.WithValue(context.TODO(), datastore.ASYNC_CHANNEL, 123)},
			want: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := datastore.AsyncChannel(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AsyncChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDoAsync(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	ch := make(chan datastore.QueuedEntity)
	ctx = datastore.WithAsyncChannel(ctx, ch)

	b := block.NewBlock("", 1)

	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name  string
		args  args
		wantE datastore.Entity
		want  bool
	}{
		{
			name:  "Test_DoAsync_TRUE",
			args:  args{ctx: ctx, entity: b},
			wantE: b,
			want:  true,
		},
		{
			name: "Test_DoAsync_FALSE",
			args: args{ctx: context.TODO()},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.want {
				go func() {
					qe := <-ch

					if !reflect.DeepEqual(qe.Entity, tt.wantE) {
						t.Errorf("DoAsync() got = %v, want = %v", qe.Entity, tt.wantE)
					}
				}()
			}

			if got := datastore.DoAsync(tt.args.ctx, tt.args.entity); got != tt.want {
				t.Errorf("DoAsync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDoAsyncEntityJSONHandler(t *testing.T) {
	t.Parallel()

	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		if entity.GetEntityMetadata().GetName() == "block" && len(entity.GetKey()) != 64 {
			return nil, errors.New("hash must be 64 size")
		}

		return nil, nil
	}

	type args struct {
		handler datastore.JSONEntityReqResponderF
		channel chan<- datastore.QueuedEntity
		entity  datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "Test_DoAsyncEntityJSONHandler_ERR",
			args:    args{handler: handler, entity: block.NewBlock("", 1)},
			wantErr: true,
		},
		{
			name: "Test_DoAsyncEntityJSONHandler_OK",
			args: func() args {
				b := block.NewBlock("", 1)
				b.Hash = encryption.Hash("data")

				return args{handler: handler, entity: b}
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := datastore.DoAsyncEntityJSONHandler(tt.args.handler, tt.args.channel)
			if _, err := handler(context.TODO(), tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("DoAsyncEntityJSONHandler() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEntityChunkBuilder_Run(t *testing.T) {
	t.Parallel()

	ecb := makeTestEntityChunkBuilder()
	entCh := make(chan datastore.QueuedEntity)

	type fields struct {
		ChunkSize      int
		MaxHoldupTime  time.Duration
		EntityChannel  <-chan datastore.QueuedEntity
		ChunkChannel   chan<- datastore.Chunk
		TimeoutChannel *time.Timer
		Chunk          datastore.Chunk
		ChunkProvider  datastore.ChunkProvider
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		entCh  chan datastore.QueuedEntity
		args   args
	}{
		{
			name: "Test_EntityChunkBuilder_Run_Max_Holdup_Time_>0_OK",
			fields: fields{
				ChunkSize:      ecb.Chunk.Size() + 1,
				MaxHoldupTime:  ecb.MaxHoldupTime,
				EntityChannel:  entCh,
				ChunkChannel:   ecb.ChunkChannel,
				TimeoutChannel: ecb.TimeoutChannel,
				Chunk:          ecb.Chunk,
				ChunkProvider:  ecb.ChunkProvider,
			},
			entCh: entCh,
			args:  args{ctx: context.TODO()},
		},
		{
			name: "Test_EntityChunkBuilder_Run_OK",
			fields: fields{
				ChunkSize:      ecb.Chunk.Size(),
				MaxHoldupTime:  0,
				EntityChannel:  entCh,
				ChunkChannel:   ecb.ChunkChannel,
				TimeoutChannel: ecb.TimeoutChannel,
				Chunk:          ecb.Chunk,
				ChunkProvider:  ecb.ChunkProvider,
			},
			entCh: entCh,
			args:  args{ctx: context.TODO()},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ecb := &datastore.EntityChunkBuilder{
				ChunkSize:      tt.fields.ChunkSize,
				MaxHoldupTime:  tt.fields.MaxHoldupTime,
				EntityChannel:  tt.fields.EntityChannel,
				ChunkChannel:   tt.fields.ChunkChannel,
				TimeoutChannel: tt.fields.TimeoutChannel,
				Chunk:          tt.fields.Chunk,
				ChunkProvider:  tt.fields.ChunkProvider,
			}

			if ecb.MaxHoldupTime > 0 {
				ctx, cancel := context.WithCancel(tt.args.ctx)

				ts := time.NewTimer(time.Millisecond * 100)
				ecb.TimeoutChannel = ts

				go ecb.Run(ctx)
				tt.entCh <- datastore.QueuedEntity{
					Entity:     block.NewBlock("", 1),
					QueuedTime: time.Now().Add(time.Hour),
				}
				cancel()
			} else {
				ctx, cancel := context.WithCancel(tt.args.ctx)

				go ecb.Run(ctx)
				tt.entCh <- datastore.QueuedEntity{Entity: block.NewBlock("", 1)}
				cancel()
			}
		})
	}
}
