package httpclientutil

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"0chain.net/core/common"
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

//TxnConfirmationTime time to wait before checking the status
const TxnConfirmationTime = 15

const txnSubmitURL = "v1/transaction/put"
const txnVerifyURL = "v1/transaction/get/confirmation?hash="
const scRestAPIURL = "v1/screst/"

//RegisterClient path to RegisterClient
var RegisterClient = "/v1/client/put"

//Signer for the transaction hash
type Signer func(h string) (string, error)

//ComputeHashAndSign compute Hash and sign the transaction
func (t *Transaction) ComputeHashAndSign(handler Signer) error {
	hashdata := fmt.Sprintf("%v:%v:%v:%v:%v", t.CreationDate, t.ClientID,
		t.ToClientID, t.Value, encryption.Hash(t.TransactionData))
	t.Hash = encryption.Hash(hashdata)
	var err error
	t.Signature, err = handler(t.Hash)
	if err != nil {
		return err
	}
	return nil
}

/////////////// Plain Transaction ///////////

//NewHTTPRequest to use in sending http requests
func NewHTTPRequest(method string, url string, data []byte, ID string, pkey string) (*http.Request, context.Context, context.CancelFunc, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, nil, nil, err
	}
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
	//ToDo: Add more error handling
	if wg != nil {
		defer wg.Done()
	}
	var resp *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		req, ctx, cncl, err := NewHTTPRequest(http.MethodPost, url, data, ID, pkey)
		if err != nil {
			Logger.Info("SendPostRequest failure", zap.String("url", url))
			return nil, err
		}

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
			err = common.NewError("http_error", "Error from HTTP call. "+string(body))
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
		return nil, common.NewError("empty_body", "empty body returned")
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, _ := ioutil.ReadAll(resp.Body)
	Logger.Info("SendPostRequest success", zap.String("url", url))
	return body, nil
}

//SendTransaction send a transaction
func SendTransaction(txn *Transaction, urls []string, ID string, pkey string) {
	for _, url := range urls {
		txnURL := fmt.Sprintf("%v/%v", url, txnSubmitURL)
		go sendTransactionToURL(txnURL, txn, ID, pkey, nil)
	}
}

//GetTransactionStatus check the status of the transaction.
func GetTransactionStatus(txnHash string, urls []string, sf int) (*Transaction, error) {
	//ToDo: Add more error handling
	numSuccess := 0
	numErrs := 0
	var errString string
	var retTxn *Transaction

	// currently transaction information an be obtained only from sharders
	for _, sharder := range urls {
		urlString := fmt.Sprintf("%v/%v%v", sharder, txnVerifyURL, txnHash)

		response, err := http.Get(urlString)
		if err != nil {
			Logger.Error("Error getting transaction confirmation", zap.Any("error", err))
			numErrs++
		} else {
			if response.StatusCode != 200 {
				continue
			}
			defer response.Body.Close()
			contents, err := ioutil.ReadAll(response.Body)
			if err != nil {
				Logger.Error("Error reading response from transaction confirmation", zap.Any("error", err))
				continue
			}
			var objmap map[string]*json.RawMessage
			err = json.Unmarshal(contents, &objmap)
			if err != nil {
				Logger.Error("Error unmarshalling response", zap.Any("error", err))
				errString = errString + urlString + ":" + err.Error()
				continue
			}
			if *objmap["txn"] == nil {
				e := "No transaction information. Only block summary."
				Logger.Error(e)
				errString = errString + urlString + ":" + e
			}
			txn := &Transaction{}
			err = json.Unmarshal(*objmap["txn"], &txn)
			if err != nil {
				Logger.Error("Error unmarshalling to get transaction response", zap.Any("error", err))
				errString = errString + urlString + ":" + err.Error()
			}
			if len(txn.Signature) > 0 {
				retTxn = txn
			}

			numSuccess++
		}
	}

	sr := int(math.Ceil((float64(numSuccess) * 100) / float64(numSuccess+numErrs)))
	// We've at least one success and success rate sr is at least same as success factor sf
	if numSuccess > 0 && sr >= sf {
		if retTxn != nil {
			return retTxn, nil
		}
		return nil, common.NewError("err_finding_txn_status", errString)
	}
	return nil, common.NewError("transaction_not_found", "Transaction was not found on any of the urls provided")
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
func MakeGetRequest(remoteUrl string, result interface{}) {

	Logger.Info(fmt.Sprintf("making GET request to %s", remoteUrl))
	//ToDo: add parameter support
	client := http.Client{}
	request, err := http.NewRequest("GET", remoteUrl, nil)
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

//MakeSCRestAPICall for smart contract REST API Call
func MakeSCRestAPICall(scAddress string, relativePath string, params map[string]string, urls []string, entity interface{}, consensus int) error {

	//ToDo: This looks a lot like GetTransactionConfirmation. Need code reuse?
	//responses := make(map[string]int)
	var retObj interface{}
	//maxCount := 0
	numSuccess := 0
	numErrs := 0
	var errString string

	//normally this goes to sharders
	for _, sharder := range urls {
		urlString := fmt.Sprintf("%v/%v%v%v", sharder, scRestAPIURL, scAddress, relativePath)
		Logger.Info("Running SCRestAPI on", zap.String("urlString", urlString))
		urlObj, _ := url.Parse(urlString)
		q := urlObj.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		urlObj.RawQuery = q.Encode()
		h := sha1.New()
		response, err := http.Get(urlObj.String())
		if err != nil {
			Logger.Error("Error getting response for sc rest api", zap.Any("error", err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
		} else {
			if response.StatusCode != 200 {
				Logger.Error("Error getting response from", zap.String("URL", sharder), zap.Any("response Status", response.StatusCode))
				numErrs++
				errString = errString + sharder + ": response_code: " + strconv.Itoa(response.StatusCode)
				continue
			}
			defer response.Body.Close()
			tReader := io.TeeReader(response.Body, h)
			d := json.NewDecoder(tReader)
			d.UseNumber()
			err := d.Decode(entity)
			if err != nil {
				Logger.Error("Error unmarshalling response", zap.Any("error", err))
				numErrs++
				errString = errString + sharder + ":" + err.Error()
				continue
			}
			retObj = entity
			numSuccess++
			/*
				Todo: Incorporate hash verification
				hashBytes := h.Sum(nil)
				hash := hex.EncodeToString(hashBytes)
				responses[hash]++
				if responses[hash] > maxCount {
					maxCount = responses[hash]
					retObj = entity
				}
			*/
		}
	}

	if numSuccess+numErrs == 0 {
		return common.NewError("req_not_run", "Could not run the request") //why???

	}
	sr := int(math.Ceil((float64(numSuccess) * 100) / float64(numSuccess+numErrs)))
	// We've at least one success and success rate sr is at least same as consensus
	if numSuccess > 0 && sr >= consensus {
		if retObj != nil {
			return nil
		}
		return common.NewError("err_getting_resp", errString)
	} else if numSuccess > 0 {
		//we had some successes, but not sufficient to reach consensus
		Logger.Error("Error Getting consensus", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return common.NewError("err_getting_consensus", errString)
	} else if numErrs > 0 {
		//We have received only errors
		Logger.Error("Error running the request", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return common.NewError("err_running_req", errString)
	}
	//this should never happen
	return common.NewError("unknown_err", "Not able to run the request. unknown reason")

}
