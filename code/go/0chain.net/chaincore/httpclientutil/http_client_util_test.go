package httpclientutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/mocks"
)

func init() {
	block.SetupEntity(&mocks.Store{})
	logging.InitLogging("development")

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
	hashdata := fmt.Sprintf("%v:%v:%v:%v:%v", want.CreationDate, want.ClientID,
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

func TestMakeGetRequest(t *testing.T) {
	server := httptest.NewServer(
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
	defer server.Close()

	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	defer errServer.Close()

	type args struct {
		remoteUrl string
		result    interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Request_Creating_ERR",
			args: args{
				remoteUrl: string(rune(0x7f)),
			},
			wantErr: true,
		},
		{
			name: "Client_ERR",
			args: args{
				remoteUrl: "/",
			},
			wantErr: true,
		},
		{
			name: "Resp_Status_Not_Ok_ERR",
			args: args{
				remoteUrl: errServer.URL,
			},
			wantErr: true,
		},
		{
			name: "JSON_Decoding_ERR",
			args: args{
				remoteUrl: server.URL,
				result:    "}{",
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				remoteUrl: server.URL,
				result:    &map[string]interface{}{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MakeGetRequest(tt.args.remoteUrl, tt.args.result); (err != nil) != tt.wantErr {
				t.Errorf("MakeGetRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMakeClientBalanceRequest(t *testing.T) {
	balance := state.Balance(5)
	server := httptest.NewServer(
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
	defer server.Close()

	invServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				if _, err := rw.Write([]byte("}{")); err != nil {
					t.Fatal(err)
				}
			},
		),
	)
	defer invServer.Close()

	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	defer invServer.Close()

	type args struct {
		clientID  string
		urls      []string
		consensus int
	}
	tests := []struct {
		name    string
		args    args
		want    state.Balance
		wantErr bool
	}{
		{
			name: "ERR",
			args: args{
				urls: []string{
					"worng url",
					errServer.URL,
					invServer.URL,
				},
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
				urls: []string{
					server.URL,
				},
				consensus: 200,
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				urls: []string{
					server.URL,
				},
				consensus: 0,
			},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeClientBalanceRequest(tt.args.clientID, tt.args.urls, tt.args.consensus)
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
	txn := Transaction{
		Hash:      encryption.Hash("data"),
		Signature: "signature",
	}

	server := httptest.NewServer(
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
	defer server.Close()

	invServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				if _, err := rw.Write([]byte("}{")); err != nil {
					t.Fatal(err)
				}
			},
		),
	)
	defer invServer.Close()

	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	defer errServer.Close()

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
	defer nilTxnServer.Close()

	type args struct {
		txnHash string
		urls    []string
		sf      int
	}
	tests := []struct {
		name    string
		args    args
		want    *Transaction
		wantErr bool
	}{
		{
			name: "ERR",
			args: args{
				urls: []string{
					"worng url",
					errServer.URL,
					invServer.URL,
					nilTxnServer.URL,
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				urls: []string{
					server.URL,
				},
			},
			want:    &txn,
			wantErr: false,
		},
		{
			name:    "Txn_Not_Found_ERR",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestMakeSCRestAPICall(t *testing.T) {
	errEntity := mocks.Serializable{}
	errEntity.On("Decode", mock.AnythingOfType("[]uint8")).Return(
		func(blob []byte) error {
			return errors.New("")
		},
	)

	entity := mocks.Serializable{}
	entity.On("Decode", mock.AnythingOfType("[]uint8")).Return(
		func(blob []byte) error {
			return nil
		},
	)

	server := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
			},
		),
	)
	defer server.Close()

	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	defer errServer.Close()

	type args struct {
		scAddress    string
		relativePath string
		params       map[string]string
		urls         []string
		entity       util.Serializable
		consensus    int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Client_ERR",
			args: args{
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
				urls: []string{
					errServer.URL,
				},
			},
			wantErr: true,
		},
		{
			name: "Response_Body_Decode_ERR",
			args: args{
				urls: []string{
					server.URL,
				},
				entity: &errEntity,
			},
			wantErr: true,
		},
		{
			name: "Consensus_Success_OK",
			args: args{
				urls: []string{
					server.URL,
				},
				entity: &entity,
			},
			wantErr: false,
		},
		{
			name: "Consensus_ERR",
			args: args{
				urls: []string{
					server.URL,
				},
				entity:    &entity,
				consensus: 200,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MakeSCRestAPICall(tt.args.scAddress, tt.args.relativePath, tt.args.params, tt.args.urls, tt.args.entity, tt.args.consensus); (err != nil) != tt.wantErr {
				t.Errorf("MakeSCRestAPICall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetBlockSummaryCall(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
			},
		),
	)
	defer server.Close()

	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	defer errServer.Close()

	type args struct {
		urls       []string
		consensus  int
		magicBlock bool
	}
	tests := []struct {
		name    string
		args    args
		want    *block.BlockSummary
		wantErr bool
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
				urls: []string{
					errServer.URL,
				},
			},
			wantErr: true,
		},
		{
			name: "Consensus_Success_OK",
			args: args{
				urls: []string{
					server.URL,
				},
			},
			want:    &block.BlockSummary{},
			wantErr: false,
		},
		{
			name: "Consensus_ERR",
			args: args{
				magicBlock: true,
				urls: []string{
					server.URL,
				},
				consensus: 200,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBlockSummaryCall(tt.args.urls, tt.args.consensus, tt.args.magicBlock)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockSummaryCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockSummaryCall() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMagicBlockCall(t *testing.T) {
	b := block.NewBlock("", 1)
	b.HashBlock()

	server := httptest.NewServer(
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
	defer server.Close()

	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusTooManyRequests)
			},
		),
	)
	defer errServer.Close()

	errBodyEncroded := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				if _, err := rw.Write([]byte("}{")); err != nil {
					t.Fatal(err)
				}
			},
		),
	)
	defer errBodyEncroded.Close()

	type args struct {
		urls             []string
		magicBlockNumber int64
		consensus        int
	}
	tests := []struct {
		name    string
		args    args
		want    *block.Block
		wantErr bool
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
				urls: []string{
					errServer.URL,
				},
			},
			wantErr: true,
		},
		{
			name: "Decode_Block_ERR",
			args: args{
				urls: []string{
					errBodyEncroded.URL,
				},
			},
			wantErr: true,
		},
		{
			name: "Consensus_Success_OK",
			args: args{
				urls: []string{
					server.URL,
				},
			},
			want: func() *block.Block {
				b := block.NewBlock("", 1)
				b.HashBlock()
				b.MagicBlock = block.NewMagicBlock()

				return b
			}(),
			wantErr: false,
		},
		{
			name: "Consensus_ERR",
			args: args{
				urls: []string{
					server.URL,
				},
				consensus: 200,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				txn: &Transaction{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := SendSmartContractTxn(tt.args.txn, tt.args.address, tt.args.value, tt.args.fee, tt.args.scData, tt.args.minerUrls); (err != nil) != tt.wantErr {
				t.Errorf("SendSmartContractTxn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
