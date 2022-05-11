package datastore

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToKey(t *testing.T) {
	t.Parallel()

	type args struct {
		key interface{}
	}
	tests := []struct {
		name string
		args args
		want Key
	}{
		{
			name: "Test_ToKey_String_OK",
			args: args{key: "key"},
			want: "key",
		},
		{
			name: "Test_ToKey_Bytes_OK",
			args: args{key: []byte("key")},
			want: "key",
		},
		{
			name: "Test_ToKey_Default_OK",
			args: args{key: 123},
			want: Key(fmt.Sprintf("%v", 123)),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ToKey(tt.args.key); got != tt.want {
				t.Errorf("ToKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashIDField_ComputeProperties(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash Key
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Test_HashIDField_ComputeProperties_OK", // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &HashIDField{
				Hash: tt.fields.Hash,
			}

			err := h.ComputeProperties()
			assert.NoError(t, err)
		})
	}
}

func TestHashIDField_GetKey(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash Key
	}
	tests := []struct {
		name   string
		fields fields
		want   Key
	}{
		{
			name:   "Test_HashIDField_GetKey_OK",
			fields: fields{Hash: "hash"},
			want:   "hash",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &HashIDField{
				Hash: tt.fields.Hash,
			}
			if got := h.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashIDField_SetKey(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash Key
	}
	type args struct {
		key Key
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_HashIDField_SetKey_OK",
			args: args{key: "key"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &HashIDField{
				Hash: tt.fields.Hash,
			}

			h.SetKey(tt.args.key)
			if h.Hash != tt.args.key {
				t.Errorf("expected setted = %v, but got = %v", tt.args.key, h.Hash)
			}
		})
	}
}

func TestHashIDField_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash Key
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
		{
			name: "Test_HashIDField_Validate_OK", // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &HashIDField{
				Hash: tt.fields.Hash,
			}
			if err := h.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIDField_ComputeProperties(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID Key
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Test_IDField_ComputeProperties_OK", // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k := &IDField{
				ID: tt.fields.ID,
			}

			err := k.ComputeProperties()
			assert.NoError(t, err)
		})
	}
}

func TestIDField_Delete(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID Key
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
		{
			name:    "Test_IDField_Delete_ERR",
			wantErr: true, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k := &IDField{
				ID: tt.fields.ID,
			}
			if err := k.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIDField_GetKey(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID Key
	}
	tests := []struct {
		name   string
		fields fields
		want   Key
	}{
		{
			name:   "Test_IDField_GetKey_OK",
			fields: fields{ID: "id"},
			want:   "id",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k := &IDField{
				ID: tt.fields.ID,
			}
			if got := k.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIDField_GetScore(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID Key
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "Test_IDField_GetScore_OK", // not implemented
			want: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k := &IDField{
				ID: tt.fields.ID,
			}
			if got := k.GetScore(); got != tt.want {
				t.Errorf("GetScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIDField_Read(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID Key
	}
	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_IDField_Read_ERR",
			wantErr: true, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k := &IDField{
				ID: tt.fields.ID,
			}
			if err := k.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIDField_SetKey(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID Key
	}
	type args struct {
		key Key
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_IDField_SetKey_OK",
			args: args{key: "key"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k := &IDField{
				ID: tt.fields.ID,
			}

			k.SetKey(tt.args.key)
			if k.ID != tt.args.key {
				t.Errorf("expected setted = %v, but got = %v", tt.args.key, k.ID)
			}
		})
	}
}

func TestIDField_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID Key
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
		{
			name: "Test_IDField_Validate_OK", // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k := &IDField{
				ID: tt.fields.ID,
			}
			if err := k.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIDField_Write(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID Key
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
		{
			name:    "Test_IDField_Write_OK",
			wantErr: true, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k := &IDField{
				ID: tt.fields.ID,
			}
			if err := k.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	t.Parallel()

	type args struct {
		key Key
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_IsEmpty_TRUE",
			want: true,
		},
		{
			name: "Test_IsEmpty_FALSE",
			args: args{key: "123"},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsEmpty(tt.args.key); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEqual(t *testing.T) {
	t.Parallel()

	type args struct {
		key1 Key
		key2 Key
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_IsEqual_TRUE",
			args: args{key1: "2", key2: "2"},
			want: true,
		},
		{
			name: "Test_IsEqual_FALSE",
			args: args{key1: "1", key2: "2"},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsEqual(tt.args.key1, tt.args.key2); got != tt.want {
				t.Errorf("IsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNOIDField_ComputeProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "Test_NOIDField_ComputeProperties_OK", // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nif := &NOIDField{}

			err := nif.ComputeProperties()
			assert.NoError(t, err)
		})
	}
}

func TestNOIDField_Delete(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_NOIDField_Delete_OK",
			wantErr: true, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nif := &NOIDField{}
			if err := nif.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNOIDField_GetKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want Key
	}{
		{
			name: "Test_NOIDField_GetKey_OK",
			want: EmptyKey,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nif := &NOIDField{}
			if got := nif.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNOIDField_GetScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want int64
	}{
		{
			name: "Test_NOIDField_GetScore_OK", // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nif := &NOIDField{}
			if got := nif.GetScore(); got != tt.want {
				t.Errorf("GetScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNOIDField_Read(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		key string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_NOIDField_Read_OK",
			wantErr: true, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nif := &NOIDField{}
			if err := nif.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNOIDField_SetKey(t *testing.T) {
	t.Parallel()

	type args struct {
		key Key
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test_NOIDField_SetKey_OK", // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nif := &NOIDField{}

			nif.SetKey(tt.args.key)
		})
	}
}

func TestNOIDField_Validate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_NOIDField_Validate_OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nif := &NOIDField{}
			if err := nif.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNOIDField_Write(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_NOIDField_Write_OK",
			wantErr: true, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nif := &NOIDField{}
			if err := nif.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToString(t *testing.T) {
	t.Parallel()

	type args struct {
		key Key
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_ToString_OK",
			args: args{"key"},
			want: "key",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ToString(tt.args.key); got != tt.want {
				t.Errorf("ToString() = %v, want %v", got, tt.want)
			}
		})
	}
}
