package encryption

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestHashHandler(t *testing.T) {
	var (
		data = "data"
		hash = Hash(data)
		w    = httptest.NewRecorder()
	)

	fmt.Fprint(w, hash)

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name  string
		args  args
		wantW http.ResponseWriter
	}{
		{
			name: "Test_HashHandler_OK",
			args: func() args {
				buf := bytes.NewBuffer(nil)
				buf.WriteString(data)
				r := httptest.NewRequest(http.MethodGet, "/", buf)

				return args{w: httptest.NewRecorder(), r: r}
			}(),
			wantW: w,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HashHandler(tt.args.w, tt.args.r)

			if !assert.Equal(t, tt.wantW, tt.args.w) {
				t.Errorf("HashHandler() got = %v, want = %v", tt.args.w, tt.wantW)
			}
		})
	}
}

func TestSignHandler(t *testing.T) {
	pbKey, prKey, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	var clientID string
	if pbKeyBytes, err := hex.DecodeString(pbKey); err != nil {
		t.Fatal(err)
	} else {
		clientID = Hash(pbKeyBytes)
	}

	var (
		ts   = time.Now().String()
		data = "data"
		hash = Hash(fmt.Sprintf("%v:%v:%v", clientID, ts, data))
		sign string
	)
	if sign, err = Sign(prKey, hash); err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "Test_SignHandler_Invalid_Public_Key_ERR",
			args: func() args {
				u := url.URL{}
				q := u.Query()
				q.Set("public_key", "!")
				u.RawQuery = q.Encode()

				return args{r: httptest.NewRequest(http.MethodPost, "/"+u.String(), nil)}
			}(),
			wantErr: true,
		},
		{
			name: "Test_SignHandler_Invalid_Sign_ERR_ERR",
			args: func() args {
				u := url.URL{}
				q := u.Query()
				q.Set("public_key", pbKey)
				q.Set("private_key", "123")
				q.Set("data", "!")
				u.RawQuery = q.Encode()

				return args{r: httptest.NewRequest(http.MethodPost, "/"+u.String(), nil)}
			}(),
			wantErr: true,
		},
		{
			name: "Test_SignHandler_OK",
			args: func() args {
				u := url.URL{}
				q := u.Query()
				q.Set("public_key", pbKey)
				q.Set("private_key", prKey)
				q.Set("data", data)
				q.Set("timestamp", ts)
				u.RawQuery = q.Encode()

				return args{r: httptest.NewRequest(http.MethodPost, "/"+u.String(), nil)}
			}(),
			want: map[string]interface{}{
				"client_id": clientID,
				"hash":      hash,
				"signature": sign,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SignHandler(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("SignHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.Equal(t, got, tt.want) {
				t.Errorf("SignHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}
