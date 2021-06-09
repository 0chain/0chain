package interestpoolsc

import (
	"reflect"
	"testing"
	"time"
)

var (
	encodedPoolRequest = []byte{123, 34, 100, 117, 114, 97, 116, 105, 111,
		110, 34, 58, 34, 49, 48, 115, 34, 125}
)

func Test_newPoolRequest_encode(t *testing.T) {
	type fields struct {
		Duration time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "encoed pool request",
			fields: fields{Duration: 10 * time.Second},
			want:   encodedPoolRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			npr := &newPoolRequest{
				Duration: tt.fields.Duration,
			}
			if got := npr.encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newPoolRequest_decode(t *testing.T) {
	type fields struct {
		Duration time.Duration
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "error",
			fields:  fields{Duration: 10 * time.Second},
			args:    args{input: encodedPoolRequest},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			npr := &newPoolRequest{
				Duration: tt.fields.Duration,
			}
			if err := npr.decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
