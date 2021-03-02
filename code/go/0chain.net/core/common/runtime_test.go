package common

import (
	"syscall"
	"testing"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func TestWaitSigInt(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Test_WaitSigInt_OK",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go WaitSigInt()
			if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestLogRuntime(t *testing.T) {
	viper.Set("logging.goroutines", true)

	type args struct {
		logger *zap.Logger
		ref    zap.Field
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestLogRuntime_OK",
			args: args{
				logger: zap.NewNop(),
				ref:    zap.Field{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			LogRuntime(tt.args.logger, tt.args.ref)
		})
	}
}
