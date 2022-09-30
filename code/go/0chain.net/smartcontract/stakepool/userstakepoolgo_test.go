package stakepool

import (
	"testing"

	"0chain.net/core/datastore"
	"github.com/stretchr/testify/require"
)

func TestUserStakePools_Del(t *testing.T) {
	type fields struct {
		Providers []string
	}
	type args struct {
		providerID datastore.Key
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		expect    []string
		wantEmpty bool
	}{
		// TODO: Add test cases.
		{
			name:      "ok - remove first",
			fields:    fields{[]string{"p1", "p2", "p3"}},
			args:      args{providerID: "p1"},
			expect:    []string{"p3", "p2"},
			wantEmpty: false,
		},
		{
			name:      "ok - empty",
			fields:    fields{[]string{"p1"}},
			args:      args{providerID: "p1"},
			expect:    []string{},
			wantEmpty: true,
		},
		{
			name:      "remove last",
			fields:    fields{[]string{"p1", "p2", "p3"}},
			args:      args{providerID: "p3"},
			expect:    []string{"p1", "p2"},
			wantEmpty: false,
		},
		{
			name:      "remove middle",
			fields:    fields{[]string{"p1", "p2", "p3"}},
			args:      args{providerID: "p2"},
			expect:    []string{"p1", "p3"},
			wantEmpty: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usp := &UserStakePools{
				Providers: tt.fields.Providers,
			}
			gotEmpty := usp.Del(tt.args.providerID)
			require.Equal(t, tt.wantEmpty, gotEmpty)

			require.Equal(t, tt.expect, usp.Providers)
		})
	}
}
