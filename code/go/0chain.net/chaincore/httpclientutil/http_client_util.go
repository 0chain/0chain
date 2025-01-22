package httpclientutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"math"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	node2 "github.com/0chain/gosdk/core/node"
	"go.uber.org/zap"
)

/*
  ToDo: This is adapted from blobber code. Need to find a way to reuse this
*/

// SleepBetweenRetries suggested time to sleep between retries
const SleepBetweenRetries = 500

const clientBalanceURL = "v1/client/get/balance?client_id="
const txnSubmitURL = "v1/transaction/put"
const txnVerifyURL = "v1/transaction/get/confirmation?hash="
const txnPendingURL = "v1/transaction/get?hash="
const specificMagicBlockURL = "v1/block/magic/get?magic_block_number="
const scRestAPIURL = "v1/screst/"
const magicBlockURL = "v1/block/get/latest_finalized_magic_block"
const finalizeBlockURL = "v1/block/get/latest_finalized"
const syncTxnNonceThreshold = 1

var gSendTxnBufferC = make(chan struct{}, 1)
var ErrTxnSendBusy = errors.New("send transaction channel busy")

func AcquireTxnLock(timeout time.Duration) bool {
	tmr := time.NewTimer(timeout)
	select {
	case <-tmr.C:
		return false
	case gSendTxnBufferC <- struct{}{}:
		return true
	}
}

func ReleaseTxnLock() {
	select {
	case <-gSendTxnBufferC:
		logging.Logger.Debug("[mvc] release txn lock")
	default:
	}
}

var gTxnFailedCount int64

func TxnFailedCountReset() {
	atomic.StoreInt64(&gTxnFailedCount, 0)
}

func TxnFailedCountInc() {
	atomic.AddInt64(&gTxnFailedCount, 1)
}

func getTxnFailedCount() int64 {
	return atomic.LoadInt64(&gTxnFailedCount)
}

// needSyncNonce checks whether it's time to sync nonce
func needSyncNonce() bool {
	return getTxnFailedCount() >= syncTxnNonceThreshold
}

var httpClient *http.Client

func init() {
	transport := &http.Transport{
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

// Signer for the transaction hash
type Signer func(h string) (string, error)

// ComputeHashAndSign compute Hash and sign the transaction
func (t *Transaction) ComputeHashAndSign(handler Signer) error {
	hashdata := fmt.Sprintf("%v:%v:%v:%v:%v:%v", t.CreationDate, t.Nonce, t.ClientID,
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

// NewHTTPRequest to use in sending http requests
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

// SendMultiPostRequest send same request to multiple URLs
func SendMultiPostRequest(urls []string, data []byte, ID string, pkey string) {
	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	for _, u := range urls {
		go func(url string) {
			if _, err := SendPostRequest(url, data, ID, pkey, &wg); err != nil {
				logging.N2n.Error("send post request failed",
					zap.String("url", url),
					zap.Error(err))
			}
		}(u)
	}
	wg.Wait()
}

// SendPostRequest function to send post requests
func SendPostRequest(url string, data []byte, ID string, pkey string, wg *sync.WaitGroup) ([]byte, error) {
	//ToDo: Add more error handling
	if wg != nil {
		defer wg.Done()
	}
	req, err := NewHTTPRequest(http.MethodPost, url, data, ID, pkey)
	if err != nil {
		logging.N2n.Info("SendPostRequest failure", zap.String("url", url))
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if resp == nil || err != nil {
		logging.N2n.Error("Failed after multiple retries",
			zap.String("url", url),
			zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, err
}

// SendTransaction send a transaction
func SendTransaction(txn *Transaction, urls []string, ID string, pkey string) {
	// make sure the txn is submitted to miners themselve so that txn pool can be checked to fast invalid txns
	if !node.Self.IsSharder() {
		// include 127.0.0.1:port in urls
		urls = append(urls, node.Self.GetN2NURLBaseLocal())
	}

	for _, u := range urls {
		txnURL := fmt.Sprintf("%v/%v", u, txnSubmitURL)
		go func(url string) {
			if _, err := sendTransactionToURL(url, txn, ID, pkey, nil); err != nil {
				logging.Logger.Error("send transaction failed",
					zap.String("url", url),
					zap.Error(err))
				logging.N2n.Error("send transaction failed",
					zap.String("url", url),
					zap.Error(err))
			}
		}(txnURL)
	}
}

// GetTransactionStatus check the status of the transaction.
func GetTransactionStatus(txnHash string, sharders []string, sf int) (*Transaction, error) {
	//ToDo: Add more error handling
	numSuccess := 0
	numErrs := 0
	var errString string
	var retTxn *Transaction

	// currently transaction information an be obtained only from sharders
	for _, sharder := range sharders {
		urlString := fmt.Sprintf("%v/%v%v", sharder, txnVerifyURL, txnHash)
		response, err := httpClient.Get(urlString)
		if err != nil {
			logging.N2n.Error("get transaction status -- failed", zap.Error(err))
			numErrs++
		} else {
			contents, err := io.ReadAll(response.Body)
			if response.StatusCode != 200 {
				// logging.Logger.Error("transaction confirmation response code",
				// 	zap.Any("code", response.StatusCode))
				response.Body.Close()
				continue
			}
			if err != nil {
				logging.Logger.Error("Error reading response from transaction confirmation", zap.Error(err))
				response.Body.Close()
				continue
			}
			var objmap map[string]*json.RawMessage
			err = json.Unmarshal(contents, &objmap)
			if err != nil {
				logging.Logger.Error("Error unmarshalling response", zap.Error(err))
				errString = errString + urlString + ":" + err.Error()
				response.Body.Close()
				continue
			}
			if *objmap["txn"] == nil {
				e := "No transaction information. Only block summary."
				logging.Logger.Error(e)
				errString = errString + urlString + ":" + e
			}
			txn := &Transaction{}
			err = json.Unmarshal(*objmap["txn"], &txn)
			if err != nil {
				logging.Logger.Error("Error unmarshalling to get transaction response", zap.Error(err))
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

func GetTransactionPendingStatus(hash string, miners []string) (*Transaction, error) {
	var (
		numSuccess int
		numErrs    int
		errString  string
		retTxn     *Transaction
	)

	for _, miner := range miners {
		urlString := fmt.Sprintf("%v/%v%v", miner, txnPendingURL, hash)
		response, err := httpClient.Get(urlString)
		if err != nil {
			logging.N2n.Error("get transaction status -- failed", zap.Error(err))
			numErrs++
		} else {
			if response.StatusCode != 200 {
				// logging.Logger.Error("transaction confirmation response code",
				// 	zap.Any("code", response.StatusCode))
				response.Body.Close()
				continue
			}

			contents, err := io.ReadAll(response.Body)
			if err != nil {
				logging.Logger.Error("Error reading response from transaction confirmation", zap.Error(err))
				response.Body.Close()
				continue
			}

			txn := &Transaction{}
			if err := json.Unmarshal(contents, &txn); err != nil {
				logging.Logger.Error("Error unmarshalling response", zap.Error(err))
				errString = errString + urlString + ":" + err.Error()
				response.Body.Close()
				continue
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
	sf := 1
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
		logging.Logger.Error("Error in serializing the transaction", zap.String("error", err.Error()), zap.Any("transaction", txn))
		return nil, err
	}

	return SendPostRequest(url, jsObj, ID, pkey, nil)
}

// MakeGetRequest make a generic get request. URL should have complete path.
// It allows 200 responses only, returning error for all other, even successful.
func MakeGetRequest(remoteUrl string, result interface{}) (err error) {
	logging.N2n.Info("make GET request", zap.String("url", remoteUrl))

	var (
		client http.Client
		rq     *http.Request
	)

	rq, err = http.NewRequest(http.MethodGet, remoteUrl, nil)
	if err != nil {
		return fmt.Errorf("make GET: can't create HTTP request "+
			"on given URL %q: %v", remoteUrl, err)
	}

	var resp *http.Response
	if resp, err = client.Do(rq); err != nil {
		return fmt.Errorf("make GET: requesting %q: %v", remoteUrl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("make GET: non-200 response code %d: %s",
			resp.StatusCode, resp.Status)
	}

	if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("make GET: decoding response: %v", err)
	}

	return // ok
}

func MakeClientBalanceRequest(clientID string, urls []string) (currency.Coin, error) {
	//balance, _, err := zcncore.GetBalance(clientID, "balance", urls)
	//return currency.Coin(balance), err
	consensus := len(urls)
	if consensus > 3 {
		consensus = 3
	}
	holder := node2.NewHolder(urls, consensus)
	balance, _, err2 := holder.GetBalanceFieldFromSharders(clientID, "balance")
	coin := currency.Coin(balance)
	return coin, err2
}

func MakeClientNonceRequest(clientID string, urls []string) (int64, error) {
	consensus := len(urls)
	if consensus > 3 {
		consensus = 3
	}
	holder := node2.NewHolder(urls, consensus)
	sharders, _, err2 := holder.GetNonceFromSharders(clientID)
	return sharders, err2
}

// MakeClientStateRequest to get a client's balance
func MakeClientStateRequest(ctx context.Context, clientID string, urls []string, consensus int) (state.State, error) {
	//ToDo: This looks a lot like GetTransactionConfirmation. Need code reuse?

	//maxCount := 0
	numSuccess := 0
	numErrs := 0
	numNotFound := 0

	var clientState state.State
	var errString string

	for _, sharder := range urls {
		u := fmt.Sprintf("%v/%v%v", sharder, clientBalanceURL, clientID)

		logging.N2n.Info("Running GetClientBalance on", zap.String("url", u))

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			logging.N2n.Error("Error creating request for sc rest api", zap.Error(err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
			continue
		}

		response, err := httpClient.Do(req)
		if err != nil {
			logging.N2n.Error("Error getting response for sc rest api", zap.Error(err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
			continue
		}

		if response.StatusCode == 400 {
			logging.N2n.Error("Node is not registered yet", zap.String("URL", sharder))
			numNotFound++
			response.Body.Close()
			continue
		}
		if response.StatusCode != 200 {
			logging.N2n.Error("Error getting response from", zap.String("URL", sharder), zap.Int("response Status", response.StatusCode))
			numErrs++
			errString = errString + sharder + ": response_code: " + strconv.Itoa(response.StatusCode)
			response.Body.Close()
			continue
		}

		d := json.NewDecoder(response.Body)
		d.UseNumber()
		err = d.Decode(&clientState)
		response.Body.Close()
		if err != nil {
			logging.Logger.Error("Error unmarshalling response", zap.Error(err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
			continue
		}

		numSuccess++
	}

	total := numSuccess + numErrs + numNotFound
	if total == 0 {
		return clientState, common.NewError("req_not_run", "Could not run the request") //why???
	}

	nr := int(math.Ceil((float64(numNotFound) * 100) / float64(total)))
	if numNotFound > 0 && nr >= consensus {
		return state.State{}, nil
	}

	sr := int(math.Ceil((float64(numSuccess) * 100) / float64(total)))

	// We've at least one success and success rate sr is at least same as consensus
	if numSuccess > 0 && sr >= consensus {
		return clientState, nil
	} else if numSuccess > 0 {
		//we had some successes, but not sufficient to reach consensus
		logging.Logger.Error("Error Getting consensus", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return clientState, common.NewError("err_getting_consensus", errString)
	} else if numErrs > 0 {
		//We have received only errors
		logging.Logger.Error("Error running the request", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return clientState, common.NewError("err_running_req", errString)
	}

	//this should never happen
	return clientState, common.NewError("unknown_err", "Not able to run the request. unknown reason")
}

// MakeSCRestAPICall for smart contract REST API Call
func MakeSCRestAPICall(ctx context.Context, scAddress string, relativePath string, params map[string]string, urls []string, entity util.Serializable, consensus int) error {

	//ToDo: This looks a lot like GetTransactionConfirmation. Need code reuse?
	var (
		numSuccess int32
		numErrs    int32
		errStringC = make(chan string, len(urls))
		respDataC  = make(chan []byte, len(urls))
	)

	// get the entity type
	if entity == nil {
		return common.NewError("SCRestAPI - decode failed", "empty entity")
	}

	entityType := reflect.TypeOf(entity).Elem()

	//normally this goes to sharders
	wg := &sync.WaitGroup{}
	for _, sharder := range urls {
		wg.Add(1)
		go func(sharderURL string) {
			defer wg.Done()
			urlString := fmt.Sprintf("%v/%v%v%v", sharderURL, scRestAPIURL, scAddress, relativePath)
			logging.N2n.Info("Running SCRestAPI on", zap.String("urlString", urlString))
			urlObj, _ := url.Parse(urlString)
			q := urlObj.Query()
			for k, v := range params {
				q.Add(k, v)
			}
			urlObj.RawQuery = q.Encode()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlObj.String(), nil)
			if err != nil {
				logging.N2n.Error("SCRestAPI - create http request with context failed", zap.Error(err))
			}

			rsp, err := httpClient.Do(req)
			if err != nil {
				logging.N2n.Error("SCRestAPI - error getting response for sc rest api", zap.Error(err))
				atomic.AddInt32(&numErrs, 1)
				errStringC <- sharderURL + ":" + err.Error()
				return
			}
			defer rsp.Body.Close()
			if rsp.StatusCode != 200 {
				logging.N2n.Error("SCRestAPI Error getting response from", zap.String("URL", sharderURL), zap.Int("response Status", rsp.StatusCode))
				atomic.AddInt32(&numErrs, 1)
				errStringC <- sharderURL + ": response_code: " + strconv.Itoa(rsp.StatusCode)
				return
			}

			bodyBytes, err := io.ReadAll(rsp.Body)
			if err != nil {
				logging.Logger.Error("SCRestAPI - failed to read body response", zap.String("URL", sharderURL), zap.Error(err))
			}
			newEntity := reflect.New(entityType).Interface().(util.Serializable)
			if err := newEntity.Decode(bodyBytes); err != nil {
				logging.Logger.Error("SCRestAPI - error unmarshalling response", zap.Error(err))
				atomic.AddInt32(&numErrs, 1)
				errStringC <- sharderURL + ":" + err.Error()
				return
			}
			respDataC <- bodyBytes
			atomic.AddInt32(&numSuccess, 1)

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
		}(sharder)
	}

	wg.Wait()
	close(errStringC)
	close(respDataC)
	errStrs := make([]string, 0, len(urls))
	for s := range errStringC {
		errStrs = append(errStrs, s)
	}

	errStr := strings.Join(errStrs, " ")

	nSuccess := atomic.LoadInt32(&numSuccess)
	nErrs := atomic.LoadInt32(&numErrs)
	logging.Logger.Info("SCRestAPI - sc rest consensus", zap.Int32("success", nSuccess))
	if nSuccess+nErrs == 0 {
		return common.NewError("req_not_run", "Could not run the request") //why???
	}
	sr := int(math.Ceil((float64(nSuccess) * 100) / float64(nSuccess+nErrs)))
	// We've at least one success and success rate sr is at least same as consensus
	if nSuccess > 0 && sr >= consensus {
		// choose the first returned entity
		select {
		case data := <-respDataC:
			if err := entity.Decode(data); err != nil {
				logging.Logger.Error("SCRestAPI - decode failed", zap.Error(err))
				return nil
			}
		default:
		}
		return nil
	} else if nSuccess > 0 {
		//we had some successes, but not sufficient to reach consensus
		logging.N2n.Error("SCRestAPI - error Getting consensus",
			zap.Int32("Success", nSuccess),
			zap.Int32("Errs", nErrs),
			zap.Int("consensus", consensus))
		return common.NewError("err_getting_consensus", errStr)
	} else if nErrs > 0 {
		//We have received only errors
		logging.N2n.Error("SCRestAPI - error running the request",
			zap.Int32("Success", nSuccess),
			zap.Int32("Errs", nErrs),
			zap.Int("consensus", consensus))
		return common.NewError("err_running_req", errStr)
	}
	//this should never happen
	return common.NewError("unknown_err", "Not able to run the request. unknown reason")
}

// MakeSCRestAPICall for smart contract REST API Call
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
			logging.N2n.Error("Error getting response for sc rest api", zap.Error(err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
		} else {
			if response.StatusCode != 200 {
				logging.N2n.Error("Error getting response from", zap.String("URL", sharder), zap.Int("response Status", response.StatusCode))
				numErrs++
				errString = errString + sharder + ": response_code: " + strconv.Itoa(response.StatusCode)
				response.Body.Close()
				continue
			}
			bodyBytes, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				logging.Logger.Error("Failed to read body response", zap.String("URL", sharder), zap.Error(err))
			}
			err = summary.Decode(bodyBytes)
			if err != nil {
				logging.Logger.Error("Error unmarshalling response", zap.Error(err))
				numErrs++
				errString = errString + sharder + ":" + err.Error()
				continue
			}
			logging.Logger.Info("get magic block -- entity", zap.String("hash", summary.Hash), zap.Int64("round", summary.Round))
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
		logging.Logger.Error("Error Getting consensus", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return nil, common.NewError("err_getting_consensus", errString)
	} else if numErrs > 0 {
		//We have received only errors
		logging.Logger.Error("Error running the request", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return nil, common.NewError("err_running_req", errString)
	}
	//this should never happen
	return nil, common.NewError("unknown_err", "Not able to run the request. unknown reason")

}

// FetchMagicBlockFromSharders fetchs magic blocks from sharders
func FetchMagicBlockFromSharders(ctx context.Context, sharderURLs []string, number int64,
	verifyBlock func(mb *block.Block) bool) (*block.Block, error) {
	if len(sharderURLs) == 0 {
		return nil, common.NewError("fetch_magic_block_from_sharders", "empty sharder URLs")
	}

	wg := &sync.WaitGroup{}
	recv := make(chan *block.Block, len(sharderURLs))
	cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	for _, sharder := range sharderURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
			if err != nil {
				logging.Logger.Error("fetch_magic_block_from_sharders - new request failed",
					zap.String("url", url),
					zap.Error(err))
				return
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				logging.Logger.Error("fetch_magic_block_from_sharders - send request failed",
					zap.String("url", url),
					zap.Error(err))
				return
			}

			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logging.Logger.Error("fetch_magic_block_from_sharders - read data failed",
					zap.String("url", url),
					zap.Error(err))
				return
			}

			b := datastore.GetEntityMetadata("block").Instance().(*block.Block)
			if err := b.Decode(body); err != nil {
				logging.Logger.Error("fetch_magic_block_from_sharders - decode data failed",
					zap.String("url", url),
					zap.Error(err))
				return
			}

			if b.MagicBlock != nil && b.MagicBlockNumber == number {
				if !verifyBlock(b) {
					logging.Logger.Error("fetch_magic_block_from_sharders - failed to verify magic block",
						zap.String("from", url),
						zap.Int64("magic_block_number", number))
					return
				}

				select {
				case recv <- b:
				default:
				}
			}
		}(fmt.Sprintf("%v/%v%v", sharder, specificMagicBlockURL, number))
	}

	go func() {
		wg.Wait()
		close(recv)
	}()

	select {
	case <-cctx.Done():
		return nil, common.NewError("fetch_magic_block_from_sharders - could not get magic block from sharders", cctx.Err().Error())
	case b, ok := <-recv:
		if !ok {
			return nil, common.NewErrorf("fetch_magic_block_from_sharders", "could not get magic block from sharders")
		}
		cancel()
		logging.Logger.Info("fetch_magic_block_from_sharders success", zap.Int64("magic_block_number", number))
		return b, nil
	}
}

// GetMagicBlockCall for smart contract to get magic block
// TODO not used, remove this func
func GetMagicBlockCall(urls []string, magicBlockNumber int64, consensus int) (*block.Block, error) {
	var retObj interface{}
	numSuccess := 0
	numErrs := 0
	var errString string
	timeoutRetry := time.Millisecond * 500
	receivedBlock := datastore.GetEntityMetadata("block").Instance().(*block.Block)
	receivedBlock.MagicBlock = block.NewMagicBlock()

	for _, sharder := range urls {
		u := fmt.Sprintf("%v/%v%v", sharder, specificMagicBlockURL, strconv.FormatInt(magicBlockNumber, 10))

		retried := 0
		var response *http.Response
		var err error
		for {
			response, err = httpClient.Get(u)
			if err != nil {
				break
			}
			if retried >= 4 || response.StatusCode != http.StatusTooManyRequests {
				response.Body.Close()
				break
			}
			response.Body.Close()
			logging.N2n.Warn("attempt to retry the request",
				zap.Int("response Status", response.StatusCode),
				zap.String("response Status text", response.Status), zap.String("URL", u),
				zap.Int("retried", retried+1))
			time.Sleep(timeoutRetry)
			retried++
		}

		if err != nil {
			logging.N2n.Error("Error getting response for sc rest api", zap.Error(err))
			numErrs++
			errString = errString + sharder + ":" + err.Error()
		} else {
			if response.StatusCode != 200 {
				logging.N2n.Error("Error getting response from", zap.String("URL", u),
					zap.Int("response Status", response.StatusCode),
					zap.String("response Status text", response.Status))
				numErrs++
				errString = errString + sharder + ": response_code: " + strconv.Itoa(response.StatusCode)
				response.Body.Close()
				continue
			}
			bodyBytes, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				logging.Logger.Error("Failed to read body response", zap.String("URL", sharder), zap.Error(err))
			}
			err = receivedBlock.Decode(bodyBytes)
			if err != nil {
				logging.Logger.Error("failed to decode block", zap.Error(err))
			}

			if err != nil {
				logging.Logger.Error("Error unmarshalling response", zap.Error(err))
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
		logging.Logger.Error("Error Getting consensus", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return nil, common.NewError("err_getting_consensus", errString)
	} else if numErrs > 0 {
		logging.Logger.Error("Error running the request", zap.Int("Success", numSuccess), zap.Int("Errs", numErrs), zap.Int("consensus", consensus))
		return nil, common.NewError("err_running_req", errString)
	}
	return nil, common.NewError("unknown_err", "Not able to run the request. unknown reason")

}

func syncClientNonce(sharders []string) (int64, error) {
	return MakeClientNonceRequest(node.Self.Underlying().GetKey(), sharders)
}

func SendSmartContractTxn(txn *Transaction, minerUrls []string, sharderUrls []string) error {
	if txn.Nonce == 0 {
		nonce, err := syncClientNonce(sharderUrls)
		if err != nil {
			logging.Logger.Error("[mvc] nonce can't get nonce from remote", zap.Error(err))
		}
		node.Self.SetNonce(nonce)
		nextNonce := node.Self.GetNextNonce()
		txn.Nonce = nextNonce
		logging.Logger.Debug("[mvc] nonce, sync in send smart txn", zap.Int64("nonce", nextNonce))
	}

	signer := func(hash string) (string, error) {
		return node.Self.Sign(hash)
	}

	err := txn.ComputeHashAndSign(signer)
	if err != nil {
		logging.Logger.Error("Signing Failed during registering miner to the mining network", zap.Error(err))
		return err
	}

	logging.Logger.Debug("[mvc] send transaction", zap.Int64("txn nonce", txn.Nonce), zap.String("txn hash", txn.Hash))

	SendTransaction(txn, minerUrls, node.Self.Underlying().GetKey(),
		node.Self.Underlying().PublicKey)
	return nil
}
