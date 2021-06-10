package interestpoolsc

import (
	"reflect"
	"testing"
)

func Test_transferResponses_addResponse(t *testing.T) {
	type fields struct {
		Responses []string
	}
	type args struct {
		response string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "ok",
			fields: fields{Responses: []string{}},
			args:   args{response: "{\"test\":\"test response\"}"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &transferResponses{
				Responses: tt.fields.Responses,
			}
			tr.addResponse(tt.args.response)
			if tr.Responses[0] != tt.args.response {
				t.Errorf("wrong response added")
			}

		})
	}
}

func Test_transferResponses_encode(t *testing.T) {
	type fields struct {
		Responses []string
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "empty ok",
			fields: fields{Responses: []string{}},
			want: []byte{123, 34, 114, 101, 115, 112, 111,
				110, 115, 101, 115, 34, 58, 91, 93, 125},
		},
		{
			name:   "full ok",
			fields: fields{Responses: []string{"{t}"}},
			want: []byte{123, 34, 114, 101, 115, 112, 111, 110, 115,
				101, 115, 34, 58, 91, 34, 123, 116, 125, 34, 93, 125,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &transferResponses{
				Responses: tt.fields.Responses,
			}
			if got := tr.encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_transferResponses_decode(t *testing.T) {
	type fields struct {
		Responses []string
	}
	type args struct {
		input []byte
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		full    bool
	}{
		{
			name:   "empty ok",
			fields: fields{Responses: []string{}},
			args: args{
				input: []byte{123, 34, 114, 101, 115, 112, 111,
					110, 115, 101, 115, 34, 58, 91, 93, 125},
			},
			wantErr: false,
		},
		{
			name:   "full ok",
			fields: fields{Responses: []string{}},
			args: args{
				input: []byte{123, 34, 114, 101, 115, 112, 111, 110, 115,
					101, 115, 34, 58, 91, 34, 123, 116, 125, 34, 93, 125,
				},
			},
			wantErr: false,
			full:    true,
		},
		{
			name:   "full ok",
			fields: fields{Responses: []string{}},
			args: args{
				input: []byte{1},
			},

			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &transferResponses{
				Responses: tt.fields.Responses,
			}
			if err := tr.decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("decode() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.full && tr.Responses[0] != "{t}" {
				t.Errorf("wrong decoded data")
			}
		})
	}
}
