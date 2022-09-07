package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go.uber.org/zap"

	"github.com/0chain/common/core/logging"
)

func init() {
	logging.Logger = zap.NewNop()
}

func TestRecover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want http.ResponseWriter
	}{
		{
			name: "Test_Recover_Use_Recover_Handler_OK",
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()

				w.Header().Set("Content-Type", "application/json")
				data := make(map[string]interface{}, 2)
				err := NewError("code", "msg")
				data["error"] = fmt.Sprintf("%v", err)
				data["code"] = err.Code
				buf := bytes.NewBuffer(nil)
				if err := json.NewEncoder(buf).Encode(data); err != nil {
					t.Fatal(err)
				}
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

			panHandler := func(w http.ResponseWriter, r *http.Request) {
				panic(NewError("code", "msg"))
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler := Recover(panHandler)
			handler(w, r)

			if !reflect.DeepEqual(w, tt.want) {
				t.Errorf("ToJSONResponse() = %#v, want %#v", w, tt.want)
			}
		})
	}
}
