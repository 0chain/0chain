package common

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

func TestUserRateLimit(t *testing.T) {
	t.Parallel()

	handler := func(w http.ResponseWriter, r *http.Request) {
	}

	w := httptest.NewRecorder()
	w.Body = nil

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
			name:   "Test_UserRateLimit_1.0_OK",
			args:   args{handler: handler},
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
		{
			name:   "Test_UserRateLimit_0_OK",
			args:   args{handler: handler},
			userRL: 0,
			want:   w,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			viper.Set("network.user_handlers.rate_limit", tt.userRL)
			ConfigRateLimits()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler := UserRateLimit(tt.args.handler)
			handler(w, r)
			w.Body = nil

			if !reflect.DeepEqual(w, tt.want) {
				t.Errorf("UserRateLimit() = %#v, want %#v", w, tt.want)
			}
		})
	}
}

func TestN2NRateLimit(t *testing.T) {
	t.Parallel()

	handler := func(w http.ResponseWriter, r *http.Request) {
	}

	w := httptest.NewRecorder()
	w.Body = nil

	type args struct {
		handler ReqRespHandlerf
	}
	tests := []struct {
		name  string
		args  args
		n2nRL float64
		want  http.ResponseWriter
	}{
		{
			name:  "Test_N2NRateLimit_1.0_OK",
			args:  args{handler: handler},
			n2nRL: 1.0,
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
		{
			name:  "Test_N2NRateLimit_0_OK",
			args:  args{handler: handler},
			n2nRL: 0,
			want:  w,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			viper.Set("network.n2n_handlers.rate_limit", tt.n2nRL)
			ConfigRateLimits()

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler := N2NRateLimit(tt.args.handler)
			handler(w, r)
			w.Body = nil

			if !reflect.DeepEqual(w, tt.want) {
				t.Errorf("N2NRateLimit() = %#v, want %#v", w, tt.want)
			}
		})
	}
}
