package httpclientutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"0chain.net/chaincore/currency"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/mocks"
	"0chain.net/core/util"
)

func init() {
	block.SetupEntity(&mocks.Store{})
	logging.InitLogging("development", "")

	startTestServer()
}

var (
	serverURL   string
	serverURLMu sync.Mutex
)

func startTestServer() {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
			},
		),
	)
	httpClient = server.Client()
	serverURL = server.URL
}

func getTestServerURL() string {
	serverURLMu.Lock()
	defer serverURLMu.Unlock()

	return serverURL
}

func TestTransaction_ComputeHashAndSign(t *testing.T) {
	t.Parallel()

	txn := NewTransactionEntity("id", "chainID", "public key")
	txn.CreationDate = 0

	_, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	handler := func(h string) (string, error) {
		return encryption.Sign(prK, h)
	}

	want := *txn
	hashdata := fmt.Sprintf("%v:%v:%v:%v:%v:%v", want.CreationDate, want.Nonce, want.ClientID,
		want.ToClientID, want.Value, encryption.Hash(want.TransactionData))
	want.Hash = encryption.Hash(hashdata)
	want.Signature, err = handler(want.Hash)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Hash              string
		Version           string
		ClientID          string
		PublicKey         string
		ToClientID        string
		ChainID           string
		TransactionData   string
		Value             int64
		Signature         string
		CreationDate      common.Timestamp
		Fee               int64
		Nonce             int64
		TransactionType   int
		TransactionOutput string
		OutputHash        string
	}
	type args struct {
		handler Signer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *Transaction
	}{
		{
			name:    "OK",
			fields:  fields(*txn),
			args:    args{handler: handler},
			wantErr: false,
			want:    &want,
		},
		{
			name:   "ERR",
			fields: fields(*txn),
			args: args{
				handler: func(h string) (string, error) {
					return "", errors.New("")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			txn := &Transaction{
				Hash:              tt.fields.Hash,
				Version:           tt.fields.Version,
				ClientID:          tt.fields.ClientID,
				PublicKey:         tt.fields.PublicKey,
				ToClientID:        tt.fields.ToClientID,
				ChainID:           tt.fields.ChainID,
				TransactionData:   tt.fields.TransactionData,
				Value:             tt.fields.Value,
				Signature:         tt.fields.Signature,
				CreationDate:      tt.fields.CreationDate,
				Fee:               tt.fields.Fee,
				TransactionType:   tt.fields.TransactionType,
				TransactionOutput: tt.fields.TransactionOutput,
				OutputHash:        tt.fields.OutputHash,
			}
			if err := txn.ComputeHashAndSign(tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("ComputeHashAndSign() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, txn)
			}
		})
	}
}

func TestNewHTTPRequest(t *testing.T) {
	t.Parallel()

	const (
		data = "data"
	)
	var (
		url  = "/"
		id   = "id"
		pKey = "pkey"
	)

	makeTestReq := func() *http.Request {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(data)))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Access-Control-Allow-Origin", "*")
		req.Header.Set("X-App-Client-ID", id)
		req.Header.Set("X-App-Client-Key", pKey)

		return req
	}

	type args struct {
		method string
		url    string
		data   []byte
		ID     string
		pkey   string
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Request
		wantErr bool
	}{
		{
			name: "OK",
			args: args{
				method: "",
				url:    url,
				data:   []byte(data),
				ID:     id,
				pkey:   pKey,
			},
			want:    makeTestReq(),
			wantErr: false,
		},
		{
			name: "ERR",
			args: args{
				url: string(rune(0x7f)),
			},
			want:    makeTestReq(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewHTTPRequest(tt.args.method, tt.args.url, tt.args.data, tt.args.ID, tt.args.pkey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && !tt.wantErr {
				assert.Equal(t, tt.want.URL, got.URL)
				tt.want.URL = nil
				got.URL = nil

				assert.Equal(t, tt.want.Body, got.Body)
			}
		})
	}
}

func TestSendPostRequest(t *testing.T) {
	t.Parallel()

	type args struct {
		url  string
		data []byte
		ID   string
		pkey string
		wg   *sync.WaitGroup
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Request_Creating_ERR",
			args: args{
				url: string(rune(0x7f)),
			},
			wantErr: true,
		},
		{
			name: "Client_ERR",
			args: args{
				url: "/",
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				url: getTestServerURL(),
			},
			want:    make([]byte, 0, 512),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := SendPostRequest(tt.args.url, tt.args.data, tt.args.ID, tt.args.pkey, tt.args.wg)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendPostRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SendPostRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendMultiPostRequest(t *testing.T) {
	t.Parallel()

	type args struct {
		urls []string
		data []byte
		ID   string
		pkey string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "OK",
			args: args{
				urls: []string{
					getTestServerURL(),
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			SendMultiPostRequest(tt.args.urls, tt.args.data, tt.args.ID, tt.args.pkey)
		})
	}
}

func TestSendTransaction(t *testing.T) {
	t.Parallel()

	type args struct {
		txn  *Transaction
		urls []string
		ID   string
		pkey string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "OK",
			args: args{
				urls: []string{
					getTestServerURL(),
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			SendTransaction(tt.args.txn, tt.args.urls, tt.args.ID, tt.args.pkey)
		})
	}
}

func makeErrServer() string {
	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	return errServer.URL
}

func makeValidServer(v interface{}) func() string {
	return func() string {
		validServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					if v == nil {
						return
					}

					b, _ := json.Marshal(v)
					io.Copy(rw, bytes.NewReader(b))
				},
			),
		)
		return validServer.URL
	}
}

func TestMakeGetRequest(t *testing.T) {
	t.Parallel()

	makeValidServer := func() string {
		validServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					data := map[string]interface{}{
						"key": "value",
					}
					blob, err := json.Marshal(data)
					if err != nil {
						t.Fatal(err)
					}

					if _, err := rw.Write(blob); err != nil {
						t.Fatal(err)
					}
				},
			),
		)
		return validServer.URL
	}

	type (
		args struct {
			remoteUrl string
			result    interface{}
		}
		makeServer func() (URL string)
	)
	tests := []struct {
		name       string
		args       args
		makeServer makeServer
		wantErr    bool
	}{
		{
			name: "Request_Creating_ERR",
			args: args{
				remoteUrl: string(rune(0x7f)),
			},
			makeServer: makeValidServer,
			wantErr:    true,
		},
		{
			name: "Client_ERR",
			args: args{
				remoteUrl: "/",
			},
			makeServer: makeValidServer,
			wantErr:    true,
		},
		{
			name:       "Resp_Status_Not_Ok_ERR",
			args:       args{},
			makeServer: makeErrServer,
			wantErr:    true,
		},
		{
			name: "JSON_Decoding_ERR",
			args: args{
				result: "}{",
			},
			makeServer: makeValidServer,
			wantErr:    true,
		},
		{
			name: "OK",
			args: args{
				result: &map[string]interface{}{},
			},
			makeServer: makeValidServer,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			remoteURL := tt.makeServer()
			if tt.args.remoteUrl != "" {
				remoteURL = tt.args.remoteUrl
			}

			if err := MakeGetRequest(remoteURL, tt.args.result); (err != nil) != tt.wantErr {
				t.Errorf("MakeGetRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMakeClientBalanceRequest(t *testing.T) {
	t.Parallel()

	balance := currency.Coin(5)
	makeValidServer := func() string {
		validServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					st := &state.State{
						Balance: balance,
					}
					blob, err := json.Marshal(st)
					if err != nil {
						t.Fatal(err)
					}

					if _, err := rw.Write(blob); err != nil {
						t.Fatal(err)
					}
				},
			),
		)
		return validServer.URL
	}

	makeInvServer := func() string {
		invServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					if _, err := rw.Write([]byte("}{")); err != nil {
						t.Fatal(err)
					}
				},
			),
		)
		return invServer.URL
	}

	type (
		args struct {
			clientID  string
			urls      []string
			consensus int
		}
		makeServer func() (URL string)
	)
	tests := []struct {
		name        string
		args        args
		want        currency.Coin
		makeServers []makeServer
		wantErr     bool
	}{
		{
			name: "ERR",
			args: args{
				urls: []string{
					"worng url",
				},
			},
			makeServers: []makeServer{
				makeErrServer,
				makeInvServer,
			},
			wantErr: true,
		},
		{
			name: "Empty_ERR",
			args: args{
				urls: []string{},
			},
			wantErr: true,
		},
		{
			name: "Consensus_ERR",
			args: args{
				urls:      []string{},
				consensus: 200,
			},
			makeServers: []makeServer{
				makeValidServer,
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				urls:      []string{},
				consensus: 0,
			},
			makeServers: []makeServer{
				makeValidServer,
			},
			want: 5,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, f := range tt.makeServers {
				URL := f()
				tt.args.urls = append(tt.args.urls, URL)
			}

			got, err := MakeClientBalanceRequest(context.TODO(), tt.args.clientID, tt.args.urls, tt.args.consensus)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeClientBalanceRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MakeClientBalanceRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTransactionStatus(t *testing.T) {
	t.Parallel()

	txn := Transaction{
		Hash:      encryption.Hash("data"),
		Signature: "signature",
	}

	makeValidServer := func() string {
		validServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					blob, err := json.Marshal(&txn)
					if err != nil {
						t.Fatal(err)
					}

					data := map[string]interface{}{
						"txn": json.RawMessage(blob),
					}
					blob, err = json.Marshal(data)
					if err != nil {
						t.Fatal(err)
					}

					if _, err := rw.Write(blob); err != nil {
						t.Fatal(err)
					}
				},
			),
		)
		return validServer.URL
	}

	makeInvServer := func() string {
		invServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					if _, err := rw.Write([]byte("}{")); err != nil {
						t.Fatal(err)
					}
				},
			),
		)
		return invServer.URL
	}

	makeNilTxnServer := func() string {
		nilTxnServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					txn := Transaction{
						Hash: encryption.Hash("data"),
					}
					blob, err := json.Marshal(&txn)
					if err != nil {
						t.Fatal(err)
					}

					data := map[string]interface{}{
						"txn": blob,
					}
					blob, err = json.Marshal(data)
					if err != nil {
						t.Fatal(err)
					}

					if _, err := rw.Write(blob); err != nil {
						t.Fatal(err)
					}
				},
			),
		)
		return nilTxnServer.URL
	}

	type (
		args struct {
			txnHash string
			urls    []string
			sf      int
		}
		makeServer func() (URL string)
	)
	tests := []struct {
		name        string
		args        args
		want        *Transaction
		makeServers []makeServer
		wantErr     bool
	}{
		{
			name: "ERR",
			args: args{
				urls: []string{
					"worng url",
				},
			},
			makeServers: []makeServer{
				makeErrServer,
				makeInvServer,
				makeNilTxnServer,
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				urls: []string{},
			},
			want: &txn,
			makeServers: []makeServer{
				makeValidServer,
			},
			wantErr: false,
		},
		{
			name:    "Txn_Not_Found_ERR",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, f := range tt.makeServers {
				URL := f()
				tt.args.urls = append(tt.args.urls, URL)
			}

			got, err := GetTransactionStatus(tt.args.txnHash, tt.args.urls, tt.args.sf)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTransactionStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTransactionStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

type mokeErrEntity struct {
	mocks.Serializable
}

func (ee *mokeErrEntity) Decode([]byte) error {
	return errors.New("")
}

type mokeEntity struct {
	mocks.Serializable
}

func (me *mokeEntity) Decode([]byte) error {
	return nil
}

func TestMakeSCRestAPICall(t *testing.T) {
	t.Parallel()

	errEntity := mokeErrEntity{}
	entity := mokeEntity{}

	type (
		args struct {
			scAddress    string
			relativePath string
			params       map[string]string
			urls         []string
			entity       util.Serializable
			consensus    int
		}
		makeServer func() (URL string)
	)
	tests := []struct {
		name        string
		args        args
		makeServers []makeServer
		wantErr     bool
	}{
		{
			name: "Client_ERR",
			args: args{
				entity: &entity,
				params: map[string]string{
					"key": "value",
				},
				urls: []string{
					"wrong url",
				},
			},
			wantErr: true,
		},
		{
			name:    "Empty_URLs_ERR",
			wantErr: true,
		},
		{
			name: "Response_Status_Not_Ok_ERR",
			args: args{
				urls: []string{},
			},
			makeServers: []makeServer{
				makeErrServer,
			},
			wantErr: true,
		},
		{
			name: "Response_Body_Decode_ERR",
			args: args{
				urls:   []string{},
				entity: &errEntity,
			},
			makeServers: []makeServer{
				makeValidServer(nil),
			},
			wantErr: true,
		},
		{
			name: "Consensus_Success_OK",
			args: args{
				urls:   []string{},
				entity: &entity,
			},
			makeServers: []makeServer{
				makeValidServer(struct{}{}),
			},
			wantErr: false,
		},
		{
			name: "Consensus_ERR",
			args: args{
				urls:      []string{},
				entity:    &entity,
				consensus: 200,
			},
			makeServers: []makeServer{
				makeValidServer(nil),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, f := range tt.makeServers {
				URL := f()
				tt.args.urls = append(tt.args.urls, URL)
			}

			if err := MakeSCRestAPICall(context.TODO(), tt.args.scAddress, tt.args.relativePath, tt.args.params, tt.args.urls, tt.args.entity, tt.args.consensus); (err != nil) != tt.wantErr {
				t.Errorf("MakeSCRestAPICall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetBlockSummaryCall(t *testing.T) {
	t.Parallel()

	type (
		args struct {
			urls       []string
			consensus  int
			magicBlock bool
		}
		makeServer func() (URL string)
	)
	tests := []struct {
		name        string
		args        args
		want        *block.BlockSummary
		makeServers []makeServer
		wantErr     bool
	}{
		{
			name: "Client_ERR",
			args: args{
				urls: []string{
					"wrong url",
				},
			},
			wantErr: true,
		},
		{
			name:    "Empty_URLs_ERR",
			wantErr: true,
		},
		{
			name: "Response_Status_Not_Ok_ERR",
			args: args{
				urls: []string{},
			},
			makeServers: []makeServer{
				makeErrServer,
			},
			wantErr: true,
		},
		{
			name: "Consensus_Success_OK",
			args: args{
				urls: []string{},
			},
			want: &block.BlockSummary{},
			makeServers: []makeServer{
				makeValidServer(struct{}{}),
			},
			wantErr: false,
		},
		{
			name: "Consensus_ERR",
			args: args{
				magicBlock: true,
				urls:       []string{},
				consensus:  200,
			},
			makeServers: []makeServer{
				makeValidServer(nil),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			//t.Parallel()

			for _, f := range tt.makeServers {
				URL := f()
				tt.args.urls = append(tt.args.urls, URL)
			}

			got, err := GetBlockSummaryCall(tt.args.urls, tt.args.consensus, tt.args.magicBlock)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGetMagicBlockCall(t *testing.T) {
	//this test is skipped since the only place tested method is used is test itself
	t.Skip()
	t.Parallel()

	b := block.NewBlock("", 1)
	b.HashBlock()

	makeValidServer := func() string {
		validServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					blob, err := json.Marshal(b)
					if err != nil {
						t.Fatal(err)
					}

					if _, err := rw.Write(blob); err != nil {
						t.Fatal(err)
					}
				},
			),
		)
		return validServer.URL
	}

	makeErrServer := func() string {
		errServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					rw.WriteHeader(http.StatusTooManyRequests)
				},
			),
		)
		return errServer.URL
	}

	makeErrBodyEncodedServer := func() string {
		errBodyEncroded := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					if _, err := rw.Write([]byte("}{")); err != nil {
						t.Fatal(err)
					}
				},
			),
		)
		return errBodyEncroded.URL
	}

	type (
		args struct {
			urls             []string
			magicBlockNumber int64
			consensus        int
		}
		makeServer func() (URL string)
	)
	tests := []struct {
		name        string
		args        args
		want        *block.Block
		makeServers []makeServer
		wantErr     bool
	}{
		{
			name: "Client_ERR",
			args: args{
				urls: []string{
					"wrong url",
				},
			},
			wantErr: true,
		},
		{
			name:    "Empty_URLs_ERR",
			wantErr: true,
		},
		{
			name: "Response_Status_Not_Ok_ERR",
			args: args{
				urls: []string{},
			},
			makeServers: []makeServer{
				makeErrServer,
			},
			wantErr: true,
		},
		{
			name: "Decode_Block_ERR",
			args: args{
				urls: []string{},
			},
			makeServers: []makeServer{
				makeErrBodyEncodedServer,
			},
			wantErr: true,
		},
		{
			name: "Consensus_Success_OK",
			args: args{
				urls: []string{},
			},
			want: func() *block.Block {
				b := block.NewBlock("", 1)
				b.HashBlock()
				b.MagicBlock = block.NewMagicBlock()

				return b
			}(),
			makeServers: []makeServer{
				makeValidServer,
			},
			wantErr: false,
		},
		{
			name: "Consensus_ERR",
			args: args{
				urls:      []string{},
				consensus: 200,
			},
			makeServers: []makeServer{
				makeValidServer,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, f := range tt.makeServers {
				URL := f()
				tt.args.urls = append(tt.args.urls, URL)
			}

			got, err := GetMagicBlockCall(tt.args.urls, tt.args.magicBlockNumber, tt.args.consensus)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMagicBlockCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSendSmartContractTxn(t *testing.T) {
	t.Parallel()

	scheme := encryption.NewED25519Scheme()
	if err := scheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}
	nonce := int64(5)
	makeValidServer := func() string {
		validServer := httptest.NewServer(
			http.HandlerFunc(
				func(rw http.ResponseWriter, r *http.Request) {
					s := state.State{
						TxnHash:      "",
						TxnHashBytes: nil,
						Round:        1,
						Balance:      10,
						Nonce:        nonce,
					}
					blob, err := json.Marshal(s)
					if err != nil {
						t.Fatal(err)
					}

					if _, err := rw.Write(blob); err != nil {
						t.Fatal(err)
					}
				},
			),
		)
		return validServer.URL
	}

	node.Self.SetSignatureScheme(scheme)
	type args struct {
		txn       *Transaction
		address   string
		value     int64
		fee       int64
		scData    *SmartContractTxnData
		minerUrls []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "OK",
			args: args{
				txn:       &Transaction{},
				minerUrls: []string{makeValidServer()},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := SendSmartContractTxn(tt.args.txn, tt.args.address, tt.args.value, tt.args.fee, tt.args.scData, tt.args.minerUrls, tt.args.minerUrls); (err != nil) != tt.wantErr {
				t.Errorf("SendSmartContractTxn() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, nonce+2, node.Self.GetNextNonce())
		})
	}
}
