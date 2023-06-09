package ememorystore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCounterMergeOperator_FullMerge(t *testing.T) {
	type fields struct {
		KeyFieldName     string
		CounterFieldName string
	}
	type args struct {
		key           []byte
		existingValue []byte
		operands      [][]byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
		want1  bool
	}{
		{
			name: "multiple operands without existing value",
			fields: fields{
				KeyFieldName:     "key",
				CounterFieldName: "counter",
			},
			args: args{
				key:           []byte("A12FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
				existingValue: []byte(""),
				operands: [][]byte{
					[]byte(`{"key":"A12FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF","counter":1}`),
					[]byte(`{"key":"A12FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF","counter":1}`),
				},
			},
			want:  []byte(`{"counter":2,"key":"A12FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"}`),
			want1: true,
		},
		{
			name: "multiple operands with existing value",
			fields: fields{
				KeyFieldName:     "key",
				CounterFieldName: "counter",
			},
			args: args{
				key:           []byte("A12FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
				existingValue: []byte(`{"key":"A12FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF","counter":1}`),
				operands: [][]byte{
					[]byte(`{"key":"A12FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF","counter":1}`),
					[]byte(`{"key":"A12FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF","counter":1}`),
				},
			},
			want:  []byte(`{"counter":3,"key":"A12FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"}`),
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &CounterMergeOperator{
				KeyFieldName:     tt.fields.KeyFieldName,
				CounterFieldName: tt.fields.CounterFieldName,
			}
			got, got1 := m.FullMerge(tt.args.key, tt.args.existingValue, tt.args.operands)
			t.Logf("got: %v", string(got))
			require.JSONEq(t, string(tt.want), string(got))
			require.Equal(t, got1, tt.want1)
		})
	}
}
