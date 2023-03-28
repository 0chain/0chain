package cache

import (
	"reflect"
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
)

func TestNewLRUCache(t *testing.T) {
	t.Parallel()

	size := 2000
	c, _ := lru.New[string, string](2000)

	type args struct {
		size int
	}
	tests := []struct {
		name string
		args args
		want *LRU[string, string]
	}{
		{
			name: "Test_NewLRUCache_OK",
			args: args{size: size},
			want: &LRU[string, string]{
				Cache: c,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewLRUCache[string, string](tt.args.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLRUCache() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLRU_Add(t *testing.T) {
	t.Parallel()

	c := NewLRUCache[string, interface{}](2000)

	type fields struct {
		Cache *lru.Cache[string, interface{}]
		Hit   int64
		Miss  int64
	}
	type args struct {
		key   string
		value interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_LRU_Add_OK",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.hit,
				Miss:  c.miss,
			},
			args: args{
				key:   "key",
				value: "value",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &LRU[string, interface{}]{
				Cache: tt.fields.Cache,
				hit:   tt.fields.Hit,
				miss:  tt.fields.Miss,
			}
			if err := c.Add(tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}

			got, err := c.Get(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() from cache error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.args.value) {
				t.Errorf("expected in cache =%v, got =%v", tt.args.value, got)
			}
		})
	}
}

func TestLRU_Get(t *testing.T) {
	t.Parallel()

	c := NewLRUCache[string, interface{}](2000)

	type fields struct {
		Cache *lru.Cache[string, interface{}]
		Hit   int64
		Miss  int64
	}
	type args struct {
		key string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       interface{}
		wantErr    bool
		saveBefore bool
	}{
		{
			name: "Test_LRU_Get_OK",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.hit,
				Miss:  c.miss,
			},
			args: args{
				key: "key",
			},
			want:       "value",
			wantErr:    false,
			saveBefore: true,
		},
		{
			name: "Test_LRU_Get_ERR",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.hit,
				Miss:  c.miss,
			},
			args: args{
				key: "key_with_err",
			},
			want:       "value",
			wantErr:    true,
			saveBefore: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &LRU[string, interface{}]{
				Cache: tt.fields.Cache,
				hit:   tt.fields.Hit,
				miss:  tt.fields.Miss,
			}
			if tt.saveBefore {
				if err := c.Add(tt.args.key, tt.want); err != nil {
					t.Fatal(err)
				}
			}

			var hit, miss int64 = c.hit, c.miss

			got, err := c.Get(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}

			if !tt.wantErr {
				if hit != c.hit-1 {
					t.Fatal("Expected incrementing hit, but it not happened")
				}
			} else {
				if miss != c.miss-1 {
					t.Fatal("Expected incrementing miss, but it not happened")
				}
			}
		})
	}
}

func TestLRU_GetHit(t *testing.T) {
	t.Parallel()

	c := NewLRUCache[string, interface{}](2000)

	type fields struct {
		Cache *lru.Cache[string, interface{}]
		Hit   int64
		Miss  int64
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "Test_LRU_GetHit_OK",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.hit,
				Miss:  c.miss,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &LRU[string, interface{}]{
				Cache: tt.fields.Cache,
				hit:   tt.fields.Hit,
				miss:  tt.fields.Miss,
			}
			if got := c.GetHit(); got != tt.want {
				t.Errorf("GetHit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLRU_GetMiss(t *testing.T) {
	t.Parallel()

	c := NewLRUCache[string, interface{}](2000)

	type fields struct {
		Cache *lru.Cache[string, interface{}]
		Hit   int64
		Miss  int64
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "Test_LRU_GetMiss_OK",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.hit,
				Miss:  c.miss,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &LRU[string, interface{}]{
				Cache: tt.fields.Cache,
				hit:   tt.fields.Hit,
				miss:  tt.fields.Miss,
			}
			if got := c.GetMiss(); got != tt.want {
				t.Errorf("GetMiss() = %v, want %v", got, tt.want)
			}
		})
	}
}
