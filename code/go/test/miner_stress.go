package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"0chain.net/core/common"
	"0chain.net/encryption"
)

func GetClient(maxConnections int) *http.Client {
	tr := &http.Transport{
		MaxIdleConns:        5 * maxConnections,
		MaxIdleConnsPerHost: 2 * maxConnections,
		IdleConnTimeout:     90 * time.Second, // more than the frequency of checking will ensure always on
		DisableCompression:  true,
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	return client
}

var httpclient *http.Client

func SendRequest(httpclient *http.Client, url string, entity interface{}) bool {
	buffer := new(bytes.Buffer)
	json.NewEncoder(buffer).Encode(entity)
	req, err := http.NewRequest("POST", url, buffer)
	if err != nil {
		return false
	}
	defer req.Body.Close()

	req.Header.Set("Content-type", "application/json; charset=utf-8")
	resp, err := httpclient.Do(req)
	if err != nil {
		msg := err.Error()
		fmt.Printf("error: %v\n", msg)
		return false
	}
	if resp.StatusCode != http.StatusOK || resp.StatusCode == 400 || resp.StatusCode == 500 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		fmt.Printf("resp code: %v: %v\n", resp.StatusCode, bodyString)
		return false
	}
	io.Copy(ioutil.Discard, resp.Body)
	return true
}

type Client struct {
	publicKey  string
	clientID   string
	privateKey string
}

var serverAddress string

func GetURL(uri string) string {
	return fmt.Sprintf("%v%v", serverAddress, uri)
}

func CreateClients(numClients int) []Client {
	clients := make([]Client, numClients)
	for i := 0; i < numClients; i++ {
		publicKey, privateKey := encryption.GenerateKeys()
		client := make(map[string]string)
		client["public_key"] = publicKey
		clientID := encryption.Hash(publicKey)
		client["id"] = clientID
		// for true {
		// 	ok := SendRequest(httpclient, GetURL("/v1/client/put"), client)
		// 	if ok {
		// 		time.Sleep(5 * time.Millisecond)
		// 		break
		// 	}
		// }
		clients[i] = Client{publicKey, clientID, privateKey}
	}
	return clients
}

func CreateTransaction(httpclient *http.Client, client Client) bool {
	txn := make(map[string]interface{})
	value := rand.Int63n(1000000000000)
	txn["client_id"] = client.clientID
	txn["transaction_value"] = value
	data := fmt.Sprintf("Pay me %v zchn.cents", value)
	txn["transaction_data"] = data
	for true {
		ts := common.Now()
		txn["creation_date"] = ts
		hashdata := fmt.Sprintf("%v:%v:%v:%v:%v", ts, client.clientID, "", value, data)
		hash := encryption.Hash(hashdata)
		signature, err := encryption.Sign(client.privateKey, hash)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			break
		}
		txn["hash"] = hash
		txn["signature"] = signature
		ok := SendRequest(httpclient, GetURL("/v1/transaction/put"), txn)
		if ok {
			// time.Sleep(50 * time.Millisecond)
			return true
		}
	}
	return true
}

func GetHash(httpclient *http.Client, data string) bool {
	url := fmt.Sprintf("/_hash?text=%v", data)
	resp, err := http.Get(GetURL(url))

	if err != nil {
		fmt.Printf("debug error: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	/*
		var rbuf bytes.Buffer
		rbuf.ReadFrom(resp.Body)
		fmt.Printf("%v: %v\n", data, rbuf.String())
	*/
	if resp.StatusCode == 200 {
		return true
	}
	return false
}

func main() {
	address := flag.String("address", "127.0.0.1:7071", "address")
	numClients := flag.Int("num_clients", 100, "num_clients")
	numTxns := flag.Int("num_txns", 1000, "num_txns")
	maxConcurrentClients := flag.Int("max_concurrent_users", 100, "max_concurrent_users")
	flag.Parse()
	if *numTxns < *numClients {
		*numClients = *numTxns
	}
	if *numClients < *maxConcurrentClients {
		*maxConcurrentClients = *numClients
	}
	serverAddress = fmt.Sprintf("http://%v", *address)
	fmt.Printf("server address: %v\n", serverAddress)
	httpclient = GetClient(*maxConcurrentClients)

	fmt.Printf("creating clients\n")
	clients := CreateClients(*numClients)
	time.Sleep(time.Second)
	fmt.Printf("clients created: %v\n", len(clients))
	ticketCannel := make(chan bool, *maxConcurrentClients)
	doneChannel := make(chan bool, *maxConcurrentClients)
	for i := 0; i < *maxConcurrentClients; i++ {
		ticketCannel <- true
	}
	fmt.Printf("starting transactions\n")
	start := time.Now()
	count := 0
	for i := 0; i < *maxConcurrentClients; i++ {
		go func() {
			for _ = range ticketCannel {
				CreateTransaction(httpclient, clients[rand.Intn(len(clients))])
				//GetHash(httpclient, "hello-world")
				doneChannel <- true
			}
		}()
	}
	for _ = range doneChannel {
		count++
		if count+*maxConcurrentClients <= *numTxns {
			ticketCannel <- true
		}
		if count == *numTxns {
			fmt.Printf("Elapsed time: %v\n", time.Since(start))
			break
		}

	}
	close(doneChannel)
}
