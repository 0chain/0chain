package memorystore

import (
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestWithConnectionHandler(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Error(err)
	}

	result := "result"

	handler := func(ctx context.Context, r *http.Request) (interface{}, error) {
		return result, nil
	}

	type args struct {
		handler common.JSONResponderF
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		{
			name: "Test_WithConnectionHandler_OK",
			args: args{handler: handler},
			want: result,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := WithConnectionHandler(tt.args.handler)
			got, err := handler(context.TODO(), nil)

			if (err != nil) && !tt.wantErr {
				t.Errorf("WithConnectionHandler() err = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithConnectionHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithConnectionJSONHandler(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Error(err)
	}

	result := "result"
	handler := func(ctx context.Context, json map[string]interface{}) (interface{}, error) {
		return result, nil
	}

	type args struct {
		handler common.JSONReqResponderF
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		{
			name: "Test_WithConnectionJSONHandler_OK",
			args: args{handler: handler},
			want: result,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := WithConnectionJSONHandler(tt.args.handler)
			got, err := handler(context.TODO(), nil)

			if (err != nil) && !tt.wantErr {
				t.Errorf("WithConnectionJSONHandler() err = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithConnectionJSONHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithConnectionEntityJSONHandler(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Error(err)
	}

	dbid := "dbid"
	AddPool(dbid, DefaultPool)

	result := "result"
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		return result, nil
	}

	type args struct {
		handler        datastore.JSONEntityReqResponderF
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		{
			name: "Test_WithConnectionEntityJSONHandler_OK",
			args: args{handler: handler, entityMetadata: &datastore.EntityMetadataImpl{DB: dbid}},
			want: result,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := WithConnectionEntityJSONHandler(tt.args.handler, tt.args.entityMetadata)
			got, err := handler(context.TODO(), nil)

			if (err != nil) && !tt.wantErr {
				t.Errorf("WithConnectionEntityJSONHandler() err = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithConnectionEntityJSONHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}
