package persistencestore_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"0chain.net/core/datastore"
	"0chain.net/core/mocks"
	"0chain.net/core/persistencestore"
)

func init() {
	viper.Set("mode", "testing")
}

func TestGetConnection(t *testing.T) {
	persistencestore.Session = &mocks.SessionI{}

	tests := []struct {
		name string
	}{
		{
			name: "Test_GetConnection_OK",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := persistencestore.GetConnection()
			want := persistencestore.Session
			if !assert.Equal(t, got, want) {
				t.Errorf("GetConnection() = %v, want %v", got, want)
			}
		})
	}
}

func TestGetCon(t *testing.T) {
	s := &mocks.SessionI{}

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want persistencestore.SessionI
	}{
		{
			name: "Test_GetCon_Nil_Ctx_OK",
			want: persistencestore.GetConnection(),
		},
		{
			name: "Test_GetCon__OK",
			args: args{ctx: context.WithValue(context.TODO(), persistencestore.CONNECTION, s)},
			want: s,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := persistencestore.GetCon(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCon() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithEntityConnection(t *testing.T) {
	ctx := context.TODO()

	type args struct {
		ctx context.Context
		in1 datastore.EntityMetadata
	}
	tests := []struct {
		name string
		args args
		want context.Context
	}{
		{
			name: "Test_WithEntityConnection_OK",
			args: args{ctx: ctx},
			want: context.WithValue(ctx, persistencestore.CONNECTION, persistencestore.GetConnection()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := persistencestore.WithEntityConnection(tt.args.ctx, tt.args.in1); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithEntityConnection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClose(t *testing.T) {
	m := &mocks.SessionI{}
	m.On("Close").Return(nil)

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test_Close_OK",
			args: args{ctx: context.WithValue(context.TODO(), persistencestore.CONNECTION, m)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persistencestore.Close(tt.args.ctx)
		})
	}
}
