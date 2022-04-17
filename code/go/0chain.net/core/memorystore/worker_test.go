package memorystore_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
)

func TestMemoryDBChunk_Add(t *testing.T) {
	t.Parallel()

	r := round.NewRound(2)

	type fields struct {
		Buffer []datastore.Entity
		Length int
	}
	type args struct {
		entity datastore.Entity
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *memorystore.MemoryDBChunk
	}{
		{
			name: "Test_MemoryDBChunk_Add_OK",
			fields: fields{
				Buffer: make([]datastore.Entity, 1),
				Length: 0,
			},
			args: args{entity: r},
			want: &memorystore.MemoryDBChunk{
				Buffer: []datastore.Entity{
					r,
				},
				Length: 1,
			},
		},
		{
			name: "Test_MemoryDBChunk_Add_OK",
			fields: fields{
				Buffer: make([]datastore.Entity, 1),
				Length: 0,
			},
			args: args{entity: r},
			want: &memorystore.MemoryDBChunk{
				Buffer: []datastore.Entity{
					r,
				},
				Length: 1,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &memorystore.MemoryDBChunk{
				Buffer: tt.fields.Buffer,
				Length: tt.fields.Length,
			}

			mc.Add(tt.args.entity)

			if !reflect.DeepEqual(mc, tt.want) {
				t.Errorf("Add() got = %v, want = %v", mc, tt.want)
			}
		})
	}
}

func TestMemoryDBChunk_Get(t *testing.T) {
	t.Parallel()

	r := round.NewRound(1)

	type fields struct {
		Buffer []datastore.Entity
		Length int
	}
	type args struct {
		index int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   datastore.Entity
	}{
		{
			name: "Test_MemoryDBChunk_Get_OK",
			fields: fields{
				Buffer: []datastore.Entity{
					r,
				},
				Length: 1,
			},
			args: args{index: 0},
			want: r,
		},
		{
			name: "Test_MemoryDBChunk_Get_OK",
			fields: fields{
				Buffer: []datastore.Entity{
					r,
				},
				Length: 1,
			},
			args: args{index: 0},
			want: r,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &memorystore.MemoryDBChunk{
				Buffer: tt.fields.Buffer,
				Length: tt.fields.Length,
			}
			if got := mc.Get(tt.args.index); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryDBChunk_Trim(t *testing.T) {
	t.Parallel()

	r := round.NewRound(1)
	buff := make([]datastore.Entity, 3)
	buff[0] = r

	type fields struct {
		Buffer []datastore.Entity
		Length int
	}
	tests := []struct {
		name   string
		fields fields
		want   *memorystore.MemoryDBChunk
	}{
		{
			name: "Test_MemoryDBChunk_Trim_OK",
			fields: fields{
				Buffer: buff,
				Length: 1,
			},
			want: &memorystore.MemoryDBChunk{
				Buffer: []datastore.Entity{
					r,
				},
				Length: 1,
			},
		},
		{
			name: "Test_MemoryDBChunk_Trim_OK",
			fields: fields{
				Buffer: buff,
				Length: 1,
			},
			want: &memorystore.MemoryDBChunk{
				Buffer: []datastore.Entity{
					r,
				},
				Length: 1,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := &memorystore.MemoryDBChunk{
				Buffer: tt.fields.Buffer,
				Length: tt.fields.Length,
			}

			mc.Trim()

			if !reflect.DeepEqual(mc, tt.want) {
				t.Errorf("Trim() got = %v, want = %v", mc, tt.want)
			}
		})
	}
}

func TestMemoryDBChunkProcessor_Process(t *testing.T) {
	initDefaultTxnPool(t)

	txn := transaction.Transaction{}
	txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn.SetKey("key")

	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		EntityMetadata datastore.EntityMetadata
		ChunkChannel   <-chan datastore.Chunk
	}
	type args struct {
		ctx   context.Context
		chunk datastore.Chunk
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantPanic bool
	}{
		{
			name: "Test_MemoryDBChunkProcessor_Process_OK",
			fields: fields{
				EntityMetadata: txn.GetEntityMetadata(),
				ChunkChannel:   nil,
			},
			args: args{
				ctx: context.TODO(),
				chunk: &memorystore.MemoryDBChunk{
					Buffer: []datastore.Entity{
						&txn,
					},
				}},
			wantPanic: false,
		},
		{
			name:      "Test_MemoryDBChunkProcessor_Process_OK",
			wantPanic: true,
		},
		{
			name: "Test_MemoryDBChunkProcessor_Process_OK",
			fields: fields{
				EntityMetadata: txn.GetEntityMetadata(),
				ChunkChannel:   nil,
			},
			args: args{
				ctx: context.TODO(),
				chunk: &memorystore.MemoryDBChunk{
					Buffer: []datastore.Entity{
						&txn,
					},
				}},
			wantPanic: false,
		},
		{
			name:      "Test_MemoryDBChunkProcessor_Process_OK",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		for i := 0; i < 2; i++ {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				defer func() {
					got := recover()
					if (got != nil) != tt.wantPanic {
						t.Errorf("Process() want panic  = %v, but got = %v", tt.wantPanic, got)
					}
				}()

				mcp := &memorystore.MemoryDBChunkProcessor{
					EntityMetadata: tt.fields.EntityMetadata,
					ChunkChannel:   tt.fields.ChunkChannel,
				}

				mcp.Process(tt.args.ctx, tt.args.chunk)
			})
		}
	}
}

func TestMemoryDBChunkProcessor_Run(t *testing.T) {
	initDefaultTxnPool(t)

	txn := transaction.Transaction{}
	txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn.SetKey("key")

	type fields struct {
		EntityMetadata datastore.EntityMetadata
		ChunkChannel   <-chan datastore.Chunk
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_MemoryDBChunkProcessor_Run_OK",
			fields: fields{
				EntityMetadata: transaction.Provider().GetEntityMetadata(),
			},
			args: args{ctx: context.TODO()},
		},
		{
			name: "Test_MemoryDBChunkProcessor_Run_OK",
			fields: fields{
				EntityMetadata: transaction.Provider().GetEntityMetadata(),
			},
			args: args{ctx: context.TODO()},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ch := make(chan datastore.Chunk)

			mcp := &memorystore.MemoryDBChunkProcessor{
				EntityMetadata: tt.fields.EntityMetadata,
				ChunkChannel:   ch,
			}

			go mcp.Run(tt.args.ctx)
			chunk := &memorystore.MemoryDBChunk{
				Buffer: []datastore.Entity{
					&txn,
				},
			}
			ch <- chunk
		})
	}
}
