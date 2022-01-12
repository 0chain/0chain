package smartcontractinterface

import (
	"reflect"
	"testing"
)

func TestNewSC(t *testing.T) {
	t.Parallel()

	id := "id"
	rh := make([]map[string]SmartContractRestHandler, 1)
	rh[0] = make(map[string]SmartContractRestHandler)
	type args struct {
		id string
	}
	tests := []struct {
		name string
		args args
		want *SmartContract
	}{
		{
			name: "OK",
			args: args{id: id},
			want: &SmartContract{
				ID:                          id,
				RestHandlers:                rh,
				SmartContractExecutionStats: make(map[string]interface{}),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewSC(tt.args.id); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSC() = %v, want %v", got, tt.want)
			}
		})
	}
}
