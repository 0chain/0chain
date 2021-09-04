package common

import (
	"reflect"
	"testing"

	"github.com/0chain/errors"
)

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
			want: errors.Newf("invalid_request", "Invalid request (%v)", "msg"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := InvalidRequest(tt.args.msg); !reflect.DeepEqual(got.Error(), tt.want.Error()) {
				t.Errorf("InvalidRequest() error = %#v, want = %#v", got, tt.want)
			}
		})
	}
}