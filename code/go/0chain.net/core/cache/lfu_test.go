package cache

import (
	"reflect"
	"testing"

	"github.com/koding/cache"
)

func TestNewLFUCache(t *testing.T) {
	t.Parallel()

	size := 2000

	type args struct {
		size int
	}
	tests := []struct {
		name string
		args args
		want *LFU
	}{
		{
			name: "Test_NewLFUCache_OK",
			args: args{size: size},
			want: &LFU{
				Cache: cache.NewLFU(size),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewLFUCache(tt.args.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLFUCache() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLFU_Add(t *testing.T) {
	t.Parallel()

	c := NewLFUCache(2000)

	type fields struct {
		Cache cache.Cache
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
			name: "Test_LFU_Add_OK",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.Hit,
				Miss:  c.Miss,
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

			c := &LFU{
				Cache: tt.fields.Cache,
				Hit:   tt.fields.Hit,
				Miss:  tt.fields.Miss,
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

func TestLFU_Get(t *testing.T) {
	t.Parallel()

	c := NewLFUCache(2000)

	type fields struct {
		Cache cache.Cache
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
			name: "Test_LFU_Get_OK",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.Hit,
				Miss:  c.Miss,
			},
			args: args{
				key: "key",
			},
			want:       "value",
			wantErr:    false,
			saveBefore: true,
		},
		{
			name: "Test_LFU_Get_ERR",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.Hit,
				Miss:  c.Miss,
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

			c := &LFU{
				Cache: tt.fields.Cache,
				Hit:   tt.fields.Hit,
				Miss:  tt.fields.Miss,
			}
			if tt.saveBefore {
				if err := c.Add(tt.args.key, tt.want); err != nil {
					t.Fatal(err)
				}
			}

			var hit, miss = c.Hit, c.Miss

			got, err := c.Get(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}

			if !tt.wantErr {
				if hit != c.Hit-1 {
					t.Fatal("Expected incrementing hit, but it is not happened")
				}
			} else {
				if miss != c.Miss-1 {
					t.Fatal("Expected incrementing miss, but it is not happened")
				}
			}
		})
	}
}

func TestLFU_GetHit(t *testing.T) {
	t.Parallel()

	c := NewLFUCache(2000)

	type fields struct {
		Cache cache.Cache
		Hit   int64
		Miss  int64
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "Test_LFU_Get_ERR",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.Hit,
				Miss:  c.Miss,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &LFU{
				Cache: tt.fields.Cache,
				Hit:   tt.fields.Hit,
				Miss:  tt.fields.Miss,
			}
			if got := c.GetHit(); got != tt.want {
				t.Errorf("GetHit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLFU_GetMiss(t *testing.T) {
	t.Parallel()

	c := NewLFUCache(2000)

	type fields struct {
		Cache cache.Cache
		Hit   int64
		Miss  int64
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "Test_LFU_GetMiss_OK",
			fields: fields{
				Cache: c.Cache,
				Hit:   c.Hit,
				Miss:  c.Miss,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &LFU{
				Cache: tt.fields.Cache,
				Hit:   tt.fields.Hit,
				Miss:  tt.fields.Miss,
			}
			if got := c.GetMiss(); got != tt.want {
				t.Errorf("GetMiss() = %v, want %v", got, tt.want)
			}
		})
	}
}
