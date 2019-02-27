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

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*
  ToDo: This is adapted from blobber code. Need to find a way to reuse this
*/
const maxRetries = 5

//SleepBetweenRetries suggested time to sleep between retries
const SleepBetweenRetries = 5

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
