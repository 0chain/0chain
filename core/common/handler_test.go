package common

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRespond(t *testing.T) {
	t.Parallel()

	var (
		err  = error(NewError("code", "msg"))
		data = map[string]string{
			"key": "value",
		}
		r = httptest.NewRequest(http.MethodGet, "/", nil)
	)

	r.Header.Add("Accept-Encoding", "gzip")

	type args struct {
		w    http.ResponseWriter
		r    *http.Request
		data interface{}
		err  error
	}
	tests := []struct {
		name     string
		args     args
		wantResp http.ResponseWriter
	}{
		{
			name: "TestRespond_Bad_Request",
			args: args{w: httptest.NewRecorder(), err: err},
			wantResp: func() http.ResponseWriter {
				w := httptest.NewRecorder()

				w.Header().Set("Content-Type", "application/json")
				data := make(map[string]interface{}, 2)
				data["error"] = err.Error()
				if cerr, ok := err.(*Error); ok {
					data["code"] = cerr.Code
				}
				buf := bytes.NewBuffer(nil)
				if err := json.NewEncoder(buf).Encode(data); err != nil {
					t.Fatal(err)
				}
				w.WriteHeader(http.StatusBadRequest)
				if _, err := buf.WriteTo(w); err != nil {
					t.Fatal(err)
				}

				return w
			}(),
		},
		{
			name: "TestRespond_OK",
			args: args{
				w:    httptest.NewRecorder(),
				r:    httptest.NewRequest(http.MethodGet, "/", nil),
				data: data,
			},
			wantResp: func() http.ResponseWriter {
				w := httptest.NewRecorder()

				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(data); err != nil {
					t.Fatal(err)
				}

				return w
			}(),
		},
		{
			name: "TestRespond_OK2",
			args: args{w: httptest.NewRecorder(), r: r, data: data},
			wantResp: func() http.ResponseWriter {
				w := httptest.NewRecorder()

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Content-Encoding", "gzip")
				gzw := gzip.NewWriter(w)
				defer func() {
					err := gzw.Close()
					require.NoError(t, err)
				}()
				if err := json.NewEncoder(gzw).Encode(data); err != nil {
					t.Fatal(err)
				}

				return w
			}(),
		},
		{
			name: "TestRespond_No_Content",
			args: args{w: httptest.NewRecorder()},
			wantResp: func() http.ResponseWriter {
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

			Respond(tt.args.w, tt.args.r, tt.args.data, tt.args.err)

			if !reflect.DeepEqual(tt.args.w, tt.wantResp) {
				t.Errorf("Respond() got = %#v, want = %#v", tt.args.w, tt.wantResp)
			}
		})
	}
}

func TestSetupCORSResponse(t *testing.T) {
	t.Parallel()

	type args struct {
		w http.ResponseWriter
	}
	tests := []struct {
		name  string
		args  args
		wantW http.ResponseWriter
	}{
		{
			name: "Test_SetupCORSResponse_OK",
			args: args{w: httptest.NewRecorder()},
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				return w
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			SetupCORSResponse(tt.args.w)

			if !reflect.DeepEqual(tt.args.w, tt.wantW) {
				t.Errorf("Respond() got = %#v, want = %#v", tt.args.w, tt.wantW)
			}
		})
	}
}

func TestToJSONResponse(t *testing.T) {
	t.Parallel()

	handler := func(ctx context.Context, r *http.Request) (interface{}, error) {
		return nil, nil
	}

	type args struct {
		handler JSONResponderF
		r       *http.Request
	}
	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "Test_ToJSONResponse_OK",
			args: args{handler: handler, r: httptest.NewRequest(http.MethodGet, "/", nil)},
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				data, err := handler(nil, r)
				Respond(w, r, data, err)

				return w
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			handler := ToJSONResponse(tt.args.handler)
			handler(w, tt.args.r)

			if !reflect.DeepEqual(w, tt.want) {
				t.Errorf("ToJSONResponse() = %#v, want %#v", w, tt.want)
			}
		})
	}
}

func TestToJSONReqResponse(t *testing.T) {
	t.Parallel()

	handler := func(ctx context.Context, json map[string]interface{}) (interface{}, error) {
		return nil, nil
	}

	type args struct {
		handler JSONReqResponderF
		r       *http.Request
	}

	tests := []struct {
		name string
		args args
		want http.ResponseWriter
	}{
		{
			name: "Test_ToJSONResponse_Bad_Request",
			args: func() args {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				return args{handler: handler, r: r}
			}(),
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()

				http.Error(w, "Header Content-type=application/json not found", 400)

				return w
			}(),
		},
		{
			name: "Test_ToJSONResponse_Unmarshalling_Err_Internal_Server_Err",
			args: func() args {
				buf := bytes.NewBuffer([]byte("}{"))
				r := httptest.NewRequest(http.MethodGet, "/", buf)
				r.Header.Add("Content-type", "application/json")
				return args{handler: handler, r: r}
			}(),
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()

				http.Error(w, "Error decoding json", 500)

				return w
			}(),
		},
		{
			name: "Test_ToJSONReqResponse_OK",
			args: func() args {
				buf := bytes.NewBuffer([]byte("{}"))
				r := httptest.NewRequest(http.MethodGet, "/", buf)
				r.Header.Add("Content-type", "application/json")
				return args{handler: handler, r: r}
			}(),
			want: func() http.ResponseWriter {
				w := httptest.NewRecorder()

				Respond(w, nil, nil, nil)

				return w
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			handler := ToJSONReqResponse(tt.args.handler)
			handler(w, tt.args.r)

			if !reflect.DeepEqual(w, tt.want) {
				t.Errorf("ToJSONResponse() = %#v, want %#v", w, tt.want)
			}
		})
	}
}

func TestJSONString(t *testing.T) {
	t.Parallel()

	type args struct {
		json     map[string]interface{}
		field    string
		required bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test_JSONString_ERR",
			args: args{
				json:     map[string]interface{}{},
				field:    "key",
				required: true,
			},
			wantErr: true,
		},
		{
			name: "Test_JSONString_Empty_OK",
			args: args{
				json:     map[string]interface{}{},
				field:    "key",
				required: false,
			},
			want: "",
		},
		{
			name: "Test_JSONString_String_Value_OK",
			args: args{
				json: map[string]interface{}{
					"key": "value",
				},
				field:    "key",
				required: true,
			},
			want: "value",
		},
		{
			name: "Test_JSONString_No_String_Value_OK",
			args: args{
				json: map[string]interface{}{
					"key": bytes.NewBuffer([]byte("value")),
				},
				field:    "key",
				required: true,
			},
			want: "value",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := JSONString(tt.args.json, tt.args.field, tt.args.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("JSONString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getContext(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.TODO()
	r = r.WithContext(ctx)

	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    context.Context
		wantErr bool
	}{
		{
			name:    "Test_getContext_OK",
			args:    args{r: r},
			want:    ctx,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := getContext(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("getContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getContext() got = %v, want %v", got, tt.want)
			}
		})
	}
}
