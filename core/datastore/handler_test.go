package datastore_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"0chain.net/core/mocks"
)

func init() {
	logging.Logger = zap.NewNop()
	logging.N2n = zap.NewNop()
	block.SetupEntity(memorystore.GetStorageProvider())
	client.SetClientSignatureScheme("ed25519")
	log.SetOutput(bytes.NewBuffer(nil))
}

func TestToJSONEntityReqResponse(t *testing.T) {
	t.Parallel()

	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		return nil, nil
	}
	em := makeTestEntityMetadataImpl()

	type args struct {
		handler        datastore.JSONEntityReqResponderF
		entityMetadata datastore.EntityMetadata
		r              *http.Request
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "Test_ToJSONEntityReqResponse_Options_OK",
			args: func() args {
				r := httptest.NewRequest(http.MethodOptions, "/", nil)
				r.Header.Add("Origin", "file://localhost:8080/")

				return args{
					handler: handler,
					r:       r,
				}
			}(),
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				return w
			}(),
		},
		{
			name: "Test_ToJSONEntityReqResponse_Not_Application_JSON_ERR",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Origin", "file://localhost:8080/")

				return args{
					handler: handler,
					r:       r,
				}
			}(),
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				http.Error(w, "Header Content-type=application/json not found", 400)
				return w
			}(),
		},
		{
			name: "Test_ToJSONEntityReqResponse_Decoding_ERR",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", bytes.NewBuffer(nil))
				r.Header.Add("Origin", "file://localhost:8080/")
				r.Header.Add("Content-type", "application/json")

				return args{
					handler:        handler,
					entityMetadata: &em,
					r:              r,
				}
			}(),
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				http.Error(w, "Error decoding json", 500)
				return w
			}(),
		},
		{
			name: "Test_ToJSONEntityReqResponse_OK",
			args: func() args {
				data := map[string]interface{}{
					"key": "value",
				}
				buf := bytes.NewBuffer(nil)
				if err := json.NewEncoder(buf).Encode(data); err != nil {
					t.Fatal(err)
				}

				r := httptest.NewRequest(http.MethodGet, "/", buf)
				r.Header.Add("Origin", "file://localhost:8080/")
				r.Header.Add("Content-type", "application/json")

				return args{
					handler:        handler,
					entityMetadata: &em,
					r:              r,
				}
			}(),
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				w.WriteHeader(http.StatusNoContent)
				return w
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			handler := datastore.ToJSONEntityReqResponse(tt.args.handler, tt.args.entityMetadata)
			handler(w, tt.args.r)

			require.Equal(t, tt.want, w)
		})
	}
}

func TestGetEntityHandler(t *testing.T) {
	t.Parallel()

	e := mocks.Entity{}
	e.On("Read", context.TODO(), mock.AnythingOfType("string")).Return(
		func(ctx context.Context, _ datastore.Key) error {
			return nil
		},
	)
	e.On("Read", context.Context(nil), mock.AnythingOfType("string")).Return(
		func(ctx context.Context, _ datastore.Key) error {
			return errors.New("")
		},
	)

	em := mocks.EntityMetadata{}
	em.On("Instance").Return(
		func() datastore.Entity {
			return &e
		},
	)

	type args struct {
		ctx            context.Context
		r              *http.Request
		entityMetadata datastore.EntityMetadata
		idparam        string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "Test_GetEntityHandler_Empty_ID_Param_ERR",
			args: args{
				r:       httptest.NewRequest(http.MethodGet, "/", nil),
				idparam: "param",
			},
			wantErr: true,
		},
		{
			name: "Test_GetEntityHandler_Read_ERR",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				id := "id"
				q := r.URL.Query()
				q.Add(id, "param")
				r.URL.RawQuery = q.Encode()

				return args{
					r:              r,
					entityMetadata: &em,
					idparam:        id,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Test_GetEntityHandler_OK",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				id := "id"
				q := r.URL.Query()
				q.Add(id, "param")
				r.URL.RawQuery = q.Encode()

				return args{
					ctx:            context.TODO(),
					r:              r,
					entityMetadata: &em,
					idparam:        id,
				}
			}(),
			want:    &e,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := datastore.GetEntityHandler(tt.args.ctx, tt.args.r, tt.args.entityMetadata, tt.args.idparam)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEntityHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPutEntityHandler(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	ch := make(chan datastore.QueuedEntity)
	ctx = datastore.WithAsyncChannel(ctx, ch)

	n := node.Node{}
	n.ID = "123"
	node.RegisterNode(&n)

	b := block.NewBlock("", 1)
	b.MinerID = n.ID
	b.Hash = b.ComputeHash()
	pbKey, prKey, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	b.Signature, err = encryption.Sign(prKey, b.Hash)
	if err != nil {
		t.Fatal(err)
	}
	n.PublicKey = pbKey

	type args struct {
		ctx    context.Context
		object interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		qeCh    chan datastore.QueuedEntity
		wantErr bool
	}{
		{
			name:    "Test_PutEntityHandler_Object_Is_Not_Entity_ERR",
			args:    args{ctx: context.TODO(), object: "not an entity"},
			wantErr: true,
		},
		{
			name:    "Test_PutEntityHandler_Invalid_Entity_ERR",
			args:    args{ctx: context.TODO(), object: block.NewBlock("", 1)},
			wantErr: true,
		},
		{
			name:    "Test_PutEntityHandler_Sync_Ctx_OK",
			args:    args{ctx: ctx, object: b},
			qeCh:    ch,
			want:    b,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.qeCh != nil {
				go func() {
					<-tt.qeCh
				}()
			}

			got, err := datastore.PutEntityHandler(tt.args.ctx, tt.args.object)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutEntityHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PutEntityHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}
