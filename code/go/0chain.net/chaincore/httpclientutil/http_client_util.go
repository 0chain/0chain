package httpclientutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

/*
  ToDo: This is adapted from blobber code. Need to find a way to reuse this
*/

const maxRetries = 5

//SleepBetweenRetries suggested time to sleep between retries
const SleepBetweenRetries = 500

//TxnConfirmationTime time to wait before checking the status
const TxnConfirmationTime = 15

const clientBalanceURL = "v1/client/get/balance?client_id="
const txnSubmitURL = "v1/transaction/put"
const txnVerifyURL = "v1/transaction/get/confirmation?hash="
const specificMagicBlockURL = "v1/block/magic/get?magic_block_number="
const scRestAPIURL = "v1/screst/"
const magicBlockURL = "v1/block/get/latest_finalized_magic_block"
const finalizeBlockURL = "v1/block/get/latest_finalized"

//RegisterClient path to RegisterClient
const RegisterClient = "/v1/client/put"

var httpClient *http.Client

func init() {
	var transport *http.Transport
	transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 1 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     1 * time.Second,
		MaxIdleConnsPerHost: 5,
	}
	httpClient = &http.Client{Transport: transport}
}

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
func NewHTTPRequest(method string, url string, data []byte, ID string, pkey string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Access-Control-Allow-Origin", "*")
	if ID != "" {
		req.Header.Set("X-App-Client-ID", ID)
	}
	if pkey != "" {
		req.Header.Set("X-App-Client-Key", pkey)
	}
	return req, err
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
	req, err := NewHTTPRequest(http.MethodPost, url, data, ID, pkey)
	if err != nil {
		Logger.Info("SendPostRequest failure", zap.String("url", url))
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if resp == nil || err != nil {
		Logger.Error("Failed after multiple retries", zap.Int("retried", maxRetries))
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
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
		response, err := httpClient.Get(urlString)
		if err != nil {
			Logger.Error("get transaction status -- failed", zap.Any("error", err))
			numErrs++
		} else {
			contents, err := ioutil.ReadAll(response.Body)
			if response.StatusCode != 200 {
				response.Body.Close()
				continue
			}
			if err != nil {
				Logger.Error("Error reading response from transaction confirmation", zap.Any("error", err))
				response.Body.Close()
				continue
			}
			var objmap map[string]*json.RawMessage
			err = json.Unmarshal(contents, &objmap)
			if err != nil {
				Logger.Error("Error unmarshalling response", zap.Any("error", err))
				errString = errString + urlString + ":" + err.Error()
				response.Body.Close()
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
			response.Body.Close()
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
		panic(err)
	}

	resp, err := client.Do(request)
	if err != nil {
		Logger.Info("Failed to run get", zap.Error(err))
		return
	}

	if resp.Body != nil {
		if resp.StatusCode == http.StatusOK {
			json.NewDecoder(resp.Body).Decode(result)
		}

		resp.Body.Close()
	} else {
		Logger.Info("resp.Body is nil")
	}
}

//MakeClientBalanceRequest to get a client's balance
func MakeClientBalanceRequest(clientID string, urls []string, consensus int) (state.Balance, error) {
	//ToDo: This looks a lot like GetTransactionConfirmation. Need code reuse?

	//maxCount := 0
	numSuccess := 0
	numErrs := 0

	var clientState state.State
	var errString string

	for _, sharder := range urls {
		url := fmt.Sprintf("%v/%v%v", sharder, clientBalanceURL, clientID)

		Logger.Info("Running GetClientBalance on", zap.String("url", url))

		response, err := http.Get(url)
		if err != nil {
			Logger.Error("Error getting response for sc rest api", zap.Any("error", err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
			continue
		}

		if response.StatusCode != 200 {
			Logger.Error("Error getting response from", zap.String("URL", sharder), zap.Any("response Status", response.StatusCode))
			numErrs++
			errString = errString + sharder + ": response_code: " + strconv.Itoa(response.StatusCode)
			continue
		}

		d := json.NewDecoder(response.Body)
		d.UseNumber()
		err = d.Decode(&clientState)
		response.Body.Close()
		if err != nil {
			Logger.Error("Error unmarshalling response", zap.Any("error", err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
			continue
		}

		numSuccess++
	}

	if numSuccess+numErrs == 0 {
		return 0, common.NewError("req_not_run", "Could not run the request") //why???
	}

	sr := int(math.Ceil((float64(numSuccess) * 100) / float64(numSuccess+numErrs)))

	// We've at least one success and success rate sr is at least same as consensus
	if numSuccess > 0 && sr >= consensus {
		return clientState.Balance, nil
	} else if numSuccess > 0 {
		//we had some successes, but not sufficient to reach consensus
		Logger.Error("Error Getting consensus", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return 0, common.NewError("err_getting_consensus", errString)
	} else if numErrs > 0 {
		//We have received only errors
		Logger.Error("Error running the request", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return 0, common.NewError("err_running_req", errString)
	}

	//this should never happen
	return 0, common.NewError("unknown_err", "Not able to run the request. unknown reason")
}

//MakeSCRestAPICall for smart contract REST API Call
func MakeSCRestAPICall(scAddress string, relativePath string, params map[string]string, urls []string, entity util.Serializable, consensus int) error {

	//ToDo: This looks a lot like GetTransactionConfirmation. Need code reuse?
	//responses := make(map[string]int)
	var retObj util.Serializable
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
		response, err := httpClient.Get(urlObj.String())
		if err != nil {
			Logger.Error("Error getting response for sc rest api", zap.Any("error", err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
		} else {
			if response.StatusCode != 200 {
				Logger.Error("Error getting response from", zap.String("URL", sharder), zap.Any("response Status", response.StatusCode))
				numErrs++
				errString = errString + sharder + ": response_code: " + strconv.Itoa(response.StatusCode)
				response.Body.Close()
				continue
			}
			bodyBytes, err := ioutil.ReadAll(response.Body)
			Logger.Info("sc rest", zap.Any("body", string(bodyBytes)), zap.Any("err", err), zap.Any("code", response.StatusCode))
			response.Body.Close()
			if err != nil {
				Logger.Error("Failed to read body response", zap.String("URL", sharder), zap.Any("error", err))
			}
			err = entity.Decode(bodyBytes)
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
	Logger.Info("sc rest consensus", zap.Any("success", numSuccess))
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

//MakeSCRestAPICall for smart contract REST API Call
func GetBlockSummaryCall(urls []string, consensus int, magicBlock bool) (*block.BlockSummary, error) {

	//ToDo: This looks a lot like GetTransactionConfirmation. Need code reuse?
	//responses := make(map[string]int)
	var retObj interface{}
	//maxCount := 0
	numSuccess := 0
	numErrs := 0
	var errString string
	summary := &block.BlockSummary{}
	// magicBlock := block.NewMagicBlock()

	//normally this goes to sharders
	for _, sharder := range urls {
		var blockUrl string
		if magicBlock {
			blockUrl = magicBlockURL
		} else {
			blockUrl = finalizeBlockURL
		}
		response, err := httpClient.Get(fmt.Sprintf("%v/%v", sharder, blockUrl))
		if err != nil {
			Logger.Error("Error getting response for sc rest api", zap.Any("error", err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
		} else {
			if response.StatusCode != 200 {
				Logger.Error("Error getting response from", zap.String("URL", sharder), zap.Any("response Status", response.StatusCode))
				numErrs++
				errString = errString + sharder + ": response_code: " + strconv.Itoa(response.StatusCode)
				response.Body.Close()
				continue
			}
			bodyBytes, err := ioutil.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				Logger.Error("Failed to read body response", zap.String("URL", sharder), zap.Any("error", err))
			}
			summary.Decode(bodyBytes)
			Logger.Info("get magic block -- entity", zap.Any("summary", summary))
			// Logger.Info("get magic block -- entity", zap.Any("magic_block", entity), zap.Any("string of magic block", string(bodyBytes)))
			if err != nil {
				Logger.Error("Error unmarshalling response", zap.Any("error", err))
				numErrs++
				errString = errString + sharder + ":" + err.Error()
				continue
			}
			retObj = summary
			numSuccess++
		}
	}

	if numSuccess+numErrs == 0 {
		return nil, common.NewError("req_not_run", "Could not run the request") //why???

	}
	sr := int(math.Ceil((float64(numSuccess) * 100) / float64(numSuccess+numErrs)))
	// We've at least one success and success rate sr is at least same as consensus
	if numSuccess > 0 && sr >= consensus {
		if retObj != nil {
			return summary, nil
		}
		return nil, common.NewError("err_getting_resp", errString)
	} else if numSuccess > 0 {
		//we had some successes, but not sufficient to reach consensus
		Logger.Error("Error Getting consensus", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return nil, common.NewError("err_getting_consensus", errString)
	} else if numErrs > 0 {
		//We have received only errors
		Logger.Error("Error running the request", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return nil, common.NewError("err_running_req", errString)
	}
	//this should never happen
	return nil, common.NewError("unknown_err", "Not able to run the request. unknown reason")

}

//GetMagicBlockCall for smart contract to get magic block
func GetMagicBlockCall(urls []string, magicBlockNumber int64, consensus int) (*block.Block, error) {
	var retObj interface{}
	numSuccess := 0
	numErrs := 0
	var errString string
	timeoutRetry := time.Millisecond * 500
	receivedBlock := datastore.GetEntityMetadata("block").Instance().(*block.Block)
	receivedBlock.MagicBlock = block.NewMagicBlock()

	for _, sharder := range urls {
		url := fmt.Sprintf("%v/%v%v", sharder, specificMagicBlockURL, strconv.FormatInt(magicBlockNumber, 10))

		retried := 0
		var response *http.Response
		var err error
		for {
			response, err = httpClient.Get(url)
			if err != nil || retried >= 4 || response.StatusCode != http.StatusTooManyRequests {
				break
			}
			response.Body.Close()
			Logger.Warn("attempt to retry the request",
				zap.Any("response Status", response.StatusCode),
				zap.Any("response Status text", response.Status), zap.String("URL", url),
				zap.Any("retried", retried+1))
			time.Sleep(timeoutRetry)
			retried++
		}

		if err != nil {
			Logger.Error("Error getting response for sc rest api", zap.Any("error", err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
		} else {
			if response.StatusCode != 200 {
				Logger.Error("Error getting response from", zap.String("URL", url),
					zap.Any("response Status", response.StatusCode),
					zap.Any("response Status text", response.Status))
				numErrs++
				errString = errString + sharder + ": response_code: " + strconv.Itoa(response.StatusCode)
				response.Body.Close()
				continue
			}
			bodyBytes, err := ioutil.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				Logger.Error("Failed to read body response", zap.String("URL", sharder), zap.Any("error", err))
			}
			err = receivedBlock.Decode(bodyBytes)
			if err != nil {
				Logger.Error("failed to decode block", zap.Any("error", err))
			}

			if err != nil {
				Logger.Error("Error unmarshalling response", zap.Any("error", err))
				numErrs++
				errString = errString + sharder + ":" + err.Error()
				continue
			}
			retObj = receivedBlock
			numSuccess++
		}
	}

	if numSuccess+numErrs == 0 {
		return nil, common.NewError("req_not_run", "Could not run the request")
	}
	sr := int(math.Ceil((float64(numSuccess) * 100) / float64(numSuccess+numErrs)))
	if numSuccess > 0 && sr >= consensus {
		if retObj != nil {
			return receivedBlock, nil
		}
		return nil, common.NewError("err_getting_resp", errString)
	} else if numSuccess > 0 {
		Logger.Error("Error Getting consensus", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return nil, common.NewError("err_getting_consensus", errString)
	} else if numErrs > 0 {
		Logger.Error("Error running the request", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return nil, common.NewError("err_running_req", errString)
	}
	return nil, common.NewError("unknown_err", "Not able to run the request. unknown reason")

}

func SendSmartContractTxn(txn *Transaction, address string, value, fee int64, scData *SmartContractTxnData, minerUrls []string) error {
	txn.ToClientID = address
	txn.Value = value
	txn.Fee = fee
	txn.TransactionType = TxnTypeSmartContract
	txnBytes, err := json.Marshal(scData)
	if err != nil {
		Logger.Error("Returning error", zap.Error(err))
		return err
	}
	txn.TransactionData = string(txnBytes)

	signer := func(hash string) (string, error) {
		return node.Self.Sign(hash)
	}

	err = txn.ComputeHashAndSign(signer)
	if err != nil {
		Logger.Info("Signing Failed during registering miner to the mining network", zap.Error(err))
		return err
	}
	SendTransaction(txn, minerUrls, node.Self.Underlying().GetKey(),
		node.Self.Underlying().PublicKey)
	return nil
}
