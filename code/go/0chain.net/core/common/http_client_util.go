package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*
  ToDo: This is adapted from blobber code. Need to find a way to reuse this
*/
const maxRetries = 5

//SleepBetweenRetries suggested time to sleep between retries
const SleepBetweenRetries = 5

const txnSubmitURL = "v1/transaction/put"
const txnVerifyURL = "v1/transaction/get/confirmation?hash="

//RegisterClient path to RegisterClient
var RegisterClient = "/v1/client/put"

//Signer for the transaction hash
type Signer func(h string) (string, error)

//Transaction entity that encapsulates the transaction related data and meta data
type Transaction struct {
	Hash              string    `json:"hash,omitempty"`
	Version           string    `json:"version,omitempty"`
	ClientID          string    `json:"client_id,omitempty"`
	PublicKey         string    `json:"public_key,omitempty"`
	ToClientID        string    `json:"to_client_id,omitempty"`
	ChainID           string    `json:"chain_id,omitempty"`
	TransactionData   string    `json:"transaction_data,omitempty"`
	Value             int64     `json:"transaction_value,omitempty"`
	Signature         string    `json:"signature,omitempty"`
	CreationDate      Timestamp `json:"creation_date,omitempty"`
	TransactionType   int       `json:"transaction_type,omitempty"`
	TransactionOutput string    `json:"transaction_output,omitempty"`
	OutputHash        string    `json:"txn_output_hash"`
}

func NewTransactionEntity(ID string, chainID string, pkey string) *Transaction {
	txn := &Transaction{}
	txn.Version = "1.0"
	txn.ClientID = ID //node.Self.ID
	txn.CreationDate = Now()
	txn.ChainID = chainID //chain.GetServerChain().ID
	txn.PublicKey = pkey  //node.Self.PublicKey
	return txn
}

func (t *Transaction) ComputeHashAndSign(handler Signer) error {
	hashdata := fmt.Sprintf("%v:%v:%v:%v:%v", t.CreationDate, t.ClientID,
		t.ToClientID, t.Value, encryption.Hash(t.TransactionData))
	t.Hash = encryption.Hash(hashdata)
	var err error
	t.Signature, err = handler(t.Hash) //node.Self.Sign(t.Hash)
	if err != nil {
		return err
	}
	return nil
}

/////////////// Plain Transaction ///////////

type SmartContractTxnData struct {
	Name      string      `json:"name"`
	InputArgs interface{} `json:"input"`
}

//NewHTTPRequest to use in sending http requests
func NewHTTPRequest(method string, url string, data []byte, ID string, pkey string) (*http.Request, context.Context, context.CancelFunc, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Access-Control-Allow-Origin", "*")
	if ID != "" {
		req.Header.Set("X-App-Client-ID", ID)
	}
	if pkey != "" {
		req.Header.Set("X-App-Client-Key", pkey)
	}
	ctx, cncl := context.WithTimeout(context.Background(), time.Second*10)
	return req, ctx, cncl, err
}

//SendMultiPostRequest send same request to multiple URLs
func SendMultiPostRequest(urls []string, data []byte, ID string, pkey string) {
	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	for _, url := range urls {
		go SendPostRequest(url, data, ID, pkey, &wg)
	}
	wg.Wait()
}

//SendPostRequest function to send post requests
func SendPostRequest(url string, data []byte, ID string, pkey string, wg *sync.WaitGroup) ([]byte, error) {
	if wg != nil {
		defer wg.Done()
	}
	var resp *http.Response
	var err error
	for i := 0; i < maxRetries; i++ {
		req, ctx, cncl, err := NewHTTPRequest(http.MethodPost, url, data, ID, pkey)
		defer cncl()
		resp, err = http.DefaultClient.Do(req.WithContext(ctx))
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
				Logger.Info("Post call success", zap.Any("url", url))
				break
			}
			body, _ := ioutil.ReadAll(resp.Body)
			if resp.Body != nil {
				resp.Body.Close()
			}
			err = NewError("http_error", "Error from HTTP call. "+string(body))
		}
		//TODO: Handle ctx cncl
		Logger.Error("SendPostRequest Error", zap.String("error", err.Error()), zap.String("URL", url))
		time.Sleep(SleepBetweenRetries * time.Second)
	}
	if resp == nil || err != nil {
		Logger.Error("Failed after multiple retries", zap.Int("retried", maxRetries))
		return nil, err
	}
	if resp.Body == nil {
		return nil, NewError("empty_body", "empty body returned")
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, _ := ioutil.ReadAll(resp.Body)
	Logger.Info("SendPostRequest success", zap.String("url", url))
	return body, nil
}

func SendTransaction(txn *Transaction, urls []string, ID string, pkey string) {
	for _, url := range urls {
		txnURL := fmt.Sprintf("%v/%v", url, txnSubmitURL)
		go sendTransactionToURL(txnURL, txn, ID, pkey, nil)
	}
}

func sendTransactionToURL(url string, txn *Transaction, ID string, pkey string, wg *sync.WaitGroup) ([]byte, error) {
	if wg != nil {
		defer wg.Done()
	}
	jsObj, err := json.Marshal(txn)
	if err != nil {
		Logger.Error("Error in serializing the transaction", zap.String("error", err.Error()), zap.Any("transaction", txn))
		return nil, err
	}

	return SendPostRequest(url, jsObj, ID, pkey, nil)
}

//MakeGetRequest make a generic get request. url should have complete path.
func MakeGetRequest(url string, result interface{}) {

	Logger.Info(fmt.Sprintf("making GET request to %s", url))
	//ToDo: add parameter support
	client := http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := client.Do(request)
	if err != nil {
		Logger.Info("Failed to run get", zap.Error(err))
		return
	}

	if resp.Body != nil {
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			json.NewDecoder(resp.Body).Decode(result)
		}
	} else {
		Logger.Info("resp.Body is nil")
	}
}
