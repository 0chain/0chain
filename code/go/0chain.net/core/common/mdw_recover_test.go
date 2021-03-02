package common

import (
	"0chain.net/core/logging"
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestRecover(t *testing.T) {
	t.Parallel()

	UseRecoverHandler = false

	handler := func(w http.ResponseWriter, r *http.Request) {
	}

	type args struct {
		handler ReqRespHandlerf
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "Test_Recover_OK",
			args: args{handler: handler},
			want: httptest.NewRecorder(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler := Recover(tt.args.handler)
			handler(w, r)

			if !reflect.DeepEqual(w, tt.want) {
				t.Errorf("ToJSONResponse() = %#v, want %#v", w, tt.want)
			}
		})
	}
}

func TestRecover_Use_Recover_Handler(t *testing.T) {
	t.Parallel()

	var err error = NewError("code", "msg")
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic(err)
	}

	type args struct {
		handler ReqRespHandlerf
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "Test_Recover_Use_Recover_Handler_OK",
			args: args{handler: handler},
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()

				w.Header().Set("Content-Type", "application/json")
				data := make(map[string]interface{}, 2)
				data["error"] = fmt.Sprintf("%v", err)
				if are, ok := err.(*Error); ok {
					data["code"] = are.Code
				}
				buf := bytes.NewBuffer(nil)
				json.NewEncoder(buf).Encode(data)
				w.WriteHeader(http.StatusInternalServerError)
				if _, err := buf.WriteTo(w); err != nil {
					t.Error(err)
				}

				return w
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler := Recover(tt.args.handler)
			handler(w, r)

			if !reflect.DeepEqual(w, tt.want) {
				t.Errorf("ToJSONResponse() = %#v, want %#v", w, tt.want)
			}
		})
	}
}
