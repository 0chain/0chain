package common

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/0chain/gosdk/core/common/errors"
)

func TestError_Error(t *testing.T) {
	t.Parallel()

	type fields struct {
		Code string
		Msg  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test_Error_Error_OK",
			fields: fields{
				Code: "code",
				Msg:  "msg",
			},
			want: fmt.Sprintf("%s: %s", "code", "msg"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := errors.New(tt.fields.Code, tt.fields.Msg)

			if got := errors.PPrint(err); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidRequest(t *testing.T) {
	t.Parallel()

	type args struct {
		msg string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "",
			args: args{msg: "msg"},
			want: errors.New("invalid_request", fmt.Sprintf("Invalid request (%v)", "msg")),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := InvalidRequest(tt.args.msg); !reflect.DeepEqual(errors.PPrint(got), errors.PPrint(tt.want)) {
				t.Errorf("InvalidRequest() error = %#v, want = %#v", got, tt.want)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	t.Parallel()

	type args struct {
		code string
		msg  string
	}
	tests := []struct {
		name string
		args args
		want *errors.Error
	}{
		{
			name: "Test_NewError_OK",
			args: args{code: "code", msg: "msg"},
			want: &errors.Error{Code: "code", Msg: "msg"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := errors.New(tt.args.code, tt.args.msg); !reflect.DeepEqual(errors.PPrint(got), errors.PPrint(tt.want)) {
				t.Errorf("NewError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewErrorf(t *testing.T) {
	t.Parallel()

	type args struct {
		code   string
		format string
		args   []interface{}
	}
	tests := []struct {
		name string
		args args
		want *errors.Error
	}{
		{
			name: "Test_NewErrorf_OK",
			args: args{
				code:   "code",
				format: "format %v",
				args:   []interface{}{1},
			},
			want: &errors.Error{Code: "code", Msg: fmt.Sprintf("format %v", 1)},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := errors.Newf(tt.args.code, tt.args.format, tt.args.args...); !reflect.DeepEqual(errors.PPrint(got), errors.PPrint(tt.want)) {
				t.Errorf("NewErrorf() = %v, want %v", errors.PPrint(got), tt.want)
			}
		})
	}
}
