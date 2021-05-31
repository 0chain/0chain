package persistencestore_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/mocks"
	"0chain.net/core/persistencestore"
)

func TestWithConnectionHandler(t *testing.T) {
	handler := func(ctx context.Context, r *http.Request) (interface{}, error) {
		return nil, nil
	}

	persistencestore.Session = &mocks.SessionI{}

	type args struct {
		handler common.JSONResponderF
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "Test_WithConnectionHandler_OK",
			args:    args{handler: handler},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := persistencestore.WithConnectionHandler(tt.args.handler)
			got, err := handler(context.TODO(), httptest.NewRequest(http.MethodGet, "/", nil))

			if (err != nil) != tt.wantErr {
				t.Errorf("WithConnectionHandler() got err = %v, want err = %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithConnectionHandler() got = %v, want = %v", got, tt.want)
			}
		})
	}
}

func TestWithConnectionJSONHandler(t *testing.T) {
	handler := func(ctx context.Context, json map[string]interface{}) (interface{}, error) {
		return nil, nil
	}

	persistencestore.Session = &mocks.SessionI{}

	type args struct {
		handler common.JSONReqResponderF
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "Test_WithConnectionJSONHandler_OK",
			args:    args{handler: handler},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := persistencestore.WithConnectionJSONHandler(tt.args.handler)
			got, err := handler(context.TODO(), map[string]interface{}{})

			if (err != nil) != tt.wantErr {
				t.Errorf("WithConnectionJSONHandler() got err = %v, want err = %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithConnectionJSONHandler() got = %v, want = %v", got, tt.want)
			}
		})
	}
}

func TestWithConnectionEntityJSONHandler(t *testing.T) {
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		return nil, nil
	}

	persistencestore.Session = &mocks.SessionI{}

	type args struct {
		handler        datastore.JSONEntityReqResponderF
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "Test_WithConnectionEntityJSONHandler_OK",
			args:    args{handler: handler},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := persistencestore.WithConnectionEntityJSONHandler(tt.args.handler, nil)
			got, err := handler(context.TODO(), block.Provider())

			if (err != nil) != tt.wantErr {
				t.Errorf("WithConnectionEntityJSONHandler() got err = %v, want err = %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithConnectionEntityJSONHandler() got = %v, want = %v", got, tt.want)
			}
		})
	}
}
