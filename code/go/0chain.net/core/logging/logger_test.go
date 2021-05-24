package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitLogging(t *testing.T) {
	type args struct {
		mode string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test_InitLogging_testing_mode_OK",
			args: args{mode: "testing"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitLogging(tt.args.mode)
		})
	}
}

func Test_getEncoder(t *testing.T) {
	t.Parallel()

	cfgUnknown := zap.NewProductionConfig()
	cfgUnknown.Encoding = ""

	type args struct {
		conf zap.Config
	}
	tests := []struct {
		name      string
		args      args
		want      zapcore.Encoder
		wantPanic bool
	}{

		{
			name:      "Test_getEncoder_Panic",
			args:      args{conf: cfgUnknown},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("getEncoder() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			if got := getEncoder(tt.args.conf); !assert.Equal(t, got, tt.want) {
				t.Errorf("getEncoder() = %v, want %v", got, tt.want)
			}
		})
	}
}
