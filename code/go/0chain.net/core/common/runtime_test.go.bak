package common

import (
	"bytes"
	"log"
	"syscall"
	"testing"
)

func init() {
	log.SetOutput(bytes.NewBuffer(nil))
}

func TestWaitSigInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "Test_WaitSigInt_OK",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			go WaitSigInt()
			if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
				t.Error(err)
			}
		})
	}
}
