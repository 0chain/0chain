package common

import (
	"context"
	"net/http"
	"reflect"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"

	"0chain.net/core/logging"
)

func TestSetupRootContext(t *testing.T) {
	logging.Logger = zap.NewNop()

	ctx := context.Background()
	//nolint:govet
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
