package common

import (
	"0chain.net/core/logging"
	"context"
	"go.uber.org/zap"
	"net/http"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestSetupRootContext(t *testing.T) {
	logging.Logger = zap.NewNop()

	ctx := context.Background()
	wantRootCtx, _ := context.WithCancel(ctx)
	SetupRootContext(context.Background())

	if !reflect.DeepEqual(wantRootCtx, rootContext) {
		t.Errorf("expected setted = %v, but got = %v", wantRootCtx, rootContext)
	}

	HandleShutdown(&http.Server{})
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Error(err)
	}

	HandleShutdown(&http.Server{})
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGQUIT); err != nil {
		t.Error(err)
	}

	time.Sleep(200 * time.Millisecond)

	if GetRootContext() != rootContext {
		t.Errorf("expected root context not same with provided")
	}
}
