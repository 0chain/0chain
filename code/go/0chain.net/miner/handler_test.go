package miner

import (
	"context"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type HandlerSuite struct {
	suite.Suite
}

func TestHandlerSuiteSuite(t *testing.T) {
	suite.Run(t, &HandlerSuite{})
}

func TestChainStatsHandler(t *testing.T) {
	ChainStatsHandler(context.Background(), &http.Request{})
}

func TestChainStatsWriter(t *testing.T) {

	httptest.NewRecorder()

	ChainStatsWriter(http.Wr)
}

func TestGetWalletStats(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func TestGetWalletTable(t *testing.T) {
	type args struct {
		latest bool
	}
	tests := []struct {
		name  string
		args  args
		want  int64
		want1 int64
		want2 int64
		want3 int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3 := GetWalletTable(tt.args.latest)
			if got != tt.want {
				t.Errorf("GetWalletTable() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetWalletTable() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("GetWalletTable() got2 = %v, want %v", got2, tt.want2)
			}
			if got3 != tt.want3 {
				t.Errorf("GetWalletTable() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}

func TestMinerStatsHandler(t *testing.T) {
	type args struct {
		ctx context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MinerStatsHandler(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinerStatsHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MinerStatsHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupHandlers(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
