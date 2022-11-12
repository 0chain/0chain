package common

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"0chain.net/core/viper"
)

func init() {
	viper.Set("network.user_handlers.rate_limit", 1.0)
	viper.Set("network.n2n_handlers.rate_limit", 1.0)
	ConfigRateLimits()
}

func makeTestHandler() ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func TestUserRateLimit(t *testing.T) {
	t.Parallel()

	type args struct {
		handler ReqRespHandlerf
	}
	tests := []struct {
		name   string
		args   args
		userRL float64
		want   http.ResponseWriter
	}{
		{
			name:   "Test_UserRateLimit_OK",
			args:   args{handler: makeTestHandler()},
			userRL: 1.0,
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				w.Header().Set("X-Rate-Limit-Limit", "1.00")
				w.Header().Set("X-Rate-Limit-Duration", "1")
				w.Header().Set("X-Rate-Limit-Request-Forwarded-For", "")
				w.Header().Set("X-Rate-Limit-Request-Remote-Addr", "192.0.2.1:1234")
				w.Body = nil

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
			handler := UserRateLimit(tt.args.handler)
			handler(w, r)
			w.Body = nil

			assert.Equal(t, tt.want, w)
		})
	}
}

func TestN2NRateLimit(t *testing.T) {
	t.Parallel()

	type args struct {
		handler ReqRespHandlerf
	}
	tests := []struct {
		name string
		args args
		want *httptest.ResponseRecorder
	}{
		{
			name: "Test_N2NRateLimit_OK",
			args: args{handler: makeTestHandler()},
			want: func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				w.Header().Set("X-Rate-Limit-Limit", "1.00")
				w.Header().Set("X-Rate-Limit-Duration", "1")
				w.Header().Set("X-Rate-Limit-Request-Forwarded-For", "")
				w.Header().Set("X-Rate-Limit-Request-Remote-Addr", "192.0.2.1:1234")
				w.Body = nil

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
			handler := N2NRateLimit(tt.args.handler)
			handler(w, r)
			w.Body = nil

			if !reflect.DeepEqual(w.Header(), tt.want.Header()) && !reflect.DeepEqual(w, tt.want) {
				t.Errorf("N2NRateLimit() = %#v, want %#v", w, tt.want)
			}
		})
	}
}
