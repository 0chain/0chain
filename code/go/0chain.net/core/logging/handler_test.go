package logging

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestLogWriter(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	mLogger.WriteLogs(w, 1)

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "Test_LogWriter_OK",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				return args{
					w: httptest.NewRecorder(),
					r: r,
				}
			}(),
			want: w,
		},
		// duplicating tests to expose race issues
		{
			name: "Test_LogWriter_OK",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				return args{
					w: httptest.NewRecorder(),
					r: r,
				}
			}(),
			want: w,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			LogWriter(tt.args.w, tt.args.r)

			if !reflect.DeepEqual(tt.args.w, tt.want) {
				t.Errorf("LogWriter() got = %v, want = %v", tt.args.w, tt.want)
			}
		})
	}
}

func TestN2NLogWriter(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	mLogger.WriteLogs(w, 1)

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "Test_N2NLogWriter_OK",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				return args{
					w: httptest.NewRecorder(),
					r: r,
				}
			}(),
			want: w,
		},
		// duplicating tests to expose race issues
		{
			name: "Test_N2NLogWriter_OK",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				return args{
					w: httptest.NewRecorder(),
					r: r,
				}
			}(),
			want: w,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			N2NLogWriter(tt.args.w, tt.args.r)

			if !reflect.DeepEqual(tt.args.w, tt.want) {
				t.Errorf("N2NLogWriter() got = %v, want = %v", tt.args.w, tt.want)
			}
		})
	}
}

func TestMemLogWriter(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	mLogger.WriteLogs(w, 1)

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "Test_MemLogWriter_OK",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				return args{
					w: httptest.NewRecorder(),
					r: r,
				}
			}(),
			want: w,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			MemLogWriter(tt.args.w, tt.args.r)

			if !reflect.DeepEqual(tt.args.w, tt.want) {
				t.Errorf("MemLogWriter() got = %v, want = %v", tt.args.w, tt.want)
			}
		})
	}
}
