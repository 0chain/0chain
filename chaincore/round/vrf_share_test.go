package round

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"0chain.net/core/mocks"
)

func init() {
	SetupVRFShareEntity(memorystore.GetStorageProvider())

	setupVrfShareDBMocks()
}

func setupVrfShareDBMocks() {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", new(VRFShare)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Write", context.Context(nil), new(VRFShare)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Delete", context.Context(nil), new(VRFShare)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	vrfsEntityMetadata.Store = &store
}

func TestVRFShare_Read(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField         datastore.NOIDField
		Round             int64
		Share             string
		RoundTimeoutCount int
		party             *node.Node
	}
	type args struct {
		ctx context.Context
		key datastore.Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vrfs := &VRFShare{
				NOIDField:         tt.fields.NOIDField,
				Round:             tt.fields.Round,
				Share:             tt.fields.Share,
				RoundTimeoutCount: tt.fields.RoundTimeoutCount,
				party:             tt.fields.party,
			}
			if err := vrfs.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVRFShare_Write(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField         datastore.NOIDField
		Round             int64
		Share             string
		RoundTimeoutCount int
		party             *node.Node
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vrfs := &VRFShare{
				NOIDField:         tt.fields.NOIDField,
				Round:             tt.fields.Round,
				Share:             tt.fields.Share,
				RoundTimeoutCount: tt.fields.RoundTimeoutCount,
				party:             tt.fields.party,
			}
			if err := vrfs.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVRFShare_Delete(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField         datastore.NOIDField
		Round             int64
		Share             string
		RoundTimeoutCount int
		party             *node.Node
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vrfs := &VRFShare{
				NOIDField:         tt.fields.NOIDField,
				Round:             tt.fields.Round,
				Share:             tt.fields.Share,
				RoundTimeoutCount: tt.fields.RoundTimeoutCount,
				party:             tt.fields.party,
			}
			if err := vrfs.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVRFShareProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want datastore.Entity
	}{
		{
			name: "OK",
			want: &VRFShare{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := VRFShareProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VRFShareProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVRFShare_GetRoundNumber(t *testing.T) {
	t.Parallel()

	num := int64(5)
	type fields struct {
		NOIDField         datastore.NOIDField
		Round             int64
		Share             string
		RoundTimeoutCount int
		party             *node.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name:   "OK",
			fields: fields{Round: num},
			want:   num,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vrfs := &VRFShare{
				NOIDField:         tt.fields.NOIDField,
				Round:             tt.fields.Round,
				Share:             tt.fields.Share,
				RoundTimeoutCount: tt.fields.RoundTimeoutCount,
				party:             tt.fields.party,
			}
			if got := vrfs.GetRoundNumber(); got != tt.want {
				t.Errorf("GetRoundNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVRFShare_GetRoundTimeoutCount(t *testing.T) {
	t.Parallel()

	c := 5

	type fields struct {
		NOIDField         datastore.NOIDField
		Round             int64
		Share             string
		RoundTimeoutCount int
		party             *node.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "OK",
			fields: fields{RoundTimeoutCount: c},
			want:   c,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vrfs := &VRFShare{
				NOIDField:         tt.fields.NOIDField,
				Round:             tt.fields.Round,
				Share:             tt.fields.Share,
				RoundTimeoutCount: tt.fields.RoundTimeoutCount,
				party:             tt.fields.party,
			}
			if got := vrfs.GetRoundTimeoutCount(); got != tt.want {
				t.Errorf("GetRoundTimeoutCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVRFShare_GetParty(t *testing.T) {
	t.Parallel()

	n, err := makeTestNode(blsPublicKeys[0])
	if err != nil {
		t.Error(err)
	}

	type fields struct {
		NOIDField         datastore.NOIDField
		Round             int64
		Share             string
		RoundTimeoutCount int
		party             *node.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   *node.Node
	}{
		{
			name: "OK",
			want: n,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vrfs := &VRFShare{
				NOIDField:         tt.fields.NOIDField,
				Round:             tt.fields.Round,
				Share:             tt.fields.Share,
				RoundTimeoutCount: tt.fields.RoundTimeoutCount,
				party:             tt.fields.party,
			}

			vrfs.SetParty(tt.want)
			if got := vrfs.GetParty(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetParty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVRFShare_GetKey(t *testing.T) {
	t.Parallel()

	num := int64(5)

	type fields struct {
		NOIDField         datastore.NOIDField
		Round             int64
		Share             string
		RoundTimeoutCount int
		party             *node.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name:   "OK",
			fields: fields{Round: num},
			want:   datastore.ToKey(fmt.Sprintf("%v", num)),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vrfs := &VRFShare{
				NOIDField:         tt.fields.NOIDField,
				Round:             tt.fields.Round,
				Share:             tt.fields.Share,
				RoundTimeoutCount: tt.fields.RoundTimeoutCount,
				party:             tt.fields.party,
			}
			if got := vrfs.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
