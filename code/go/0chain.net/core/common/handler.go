package common

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

/*AppErrorHeader - a http response header to send an application error code */
const AppErrorHeader = "X-App-Error-Code"

/*ReqRespHandlerf - a type for the default hanlder signature */
type ReqRespHandlerf func(w http.ResponseWriter, r *http.Request)

/*JSONResponderF - a handler that takes standard request (non-json) and responds with a json response
* Useful for POST opertaion where the input is posted as json with
*    Content-type: application/json
* header
 */
type JSONResponderF func(ctx context.Context, r *http.Request) (interface{}, error)

/*JSONReqResponderF - a handler that takes a JSON request and responds with a json response
* Useful for GET operation where the input is coming via url parameters
 */
type JSONReqResponderF func(ctx context.Context, json map[string]interface{}) (interface{}, error)

/*Respond - respond either data or error as a response */
func Respond(w http.ResponseWriter, r *http.Request, data interface{}, err error) {
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		data := make(map[string]interface{}, 2)
		data["error"] = err.Error()
		if cerr, ok := err.(*Error); ok {
			data["code"] = cerr.Code
		}
		buf := bytes.NewBuffer(nil)
		json.NewEncoder(buf).Encode(data)
		http.Error(w, buf.String(), 400)
	} else {
		if data != nil {
			w.Header().Set("Content-Type", "application/json")
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				json.NewEncoder(w).Encode(data)
			} else {
				w.Header().Set("Content-Encoding", "gzip")
				gzw := gzip.NewWriter(w)
				defer gzw.Close()
				json.NewEncoder(gzw).Encode(data)
			}
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func getContext(r *http.Request) (context.Context, error) {
	ctx := r.Context()
	return ctx, nil
}

var domainRE = regexp.MustCompile(`^(?:https?:\/\/)?(?:[^@\/\n]+@)?(?:www\.)?([^:\/\n]+)`)

func getHost(origin string) (string, error) {
	url, err := url.Parse(origin)
	return url.Hostname(), err
}

func validOrigin(origin string) bool {
	host, err := getHost(origin)
	if err != nil {
		return false
	}
	if host == "localhost" || strings.HasPrefix(host, "file") {
		return true
	}
	if host == "0chain.net" ||
		strings.HasSuffix(host, ".0chain.net") ||
		strings.HasSuffix(host, ".alphanet-0chain.net") ||
		strings.HasSuffix(host, ".testnet-0chain.net") ||
		strings.HasSuffix(host, ".devnet-0chain.net") ||
		strings.HasSuffix(host, ".mainnet-0chain.net") {
		return true
	}
	return false
}

func CheckCrossOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if validOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		return true
	}
	return false
}

func SetupCORSResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Accept-Encoding")
}

/*ToJSONResponse - An adapter that takes a handler of the form
* func AHandler(r *http.Request) (interface{}, error)
* which takes a request object, processes and returns an object or an error
* and converts into a standard request/response handler
 */
func ToJSONResponse(handler JSONResponderF) ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		if !CheckCrossOrigin(w, r) {
			return
		}
		ctx := r.Context()
		data, err := handler(ctx, r)
		Respond(w, r, data, err)
	}
}

/*ToJSONReqResponse - An adapter that takes a handler of the form
* func AHandler(json map[string]interface{}) (interface{}, error)
* which takes a parsed json map from the request, processes and returns an object or an error
* and converts into a standard request/response handler
 */
func ToJSONReqResponse(handler JSONReqResponderF) ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		if !CheckCrossOrigin(w, r) {
			return
		}
		contentType := r.Header.Get("Content-type")
		if !strings.HasPrefix(contentType, "application/json") {
			http.Error(w, "Header Content-type=application/json not found", 400)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var jsonData map[string]interface{}
		err := decoder.Decode(&jsonData)
		if err != nil {
			http.Error(w, "Error decoding json", 500)
			return
		}
		ctx := r.Context()
		data, err := handler(ctx, jsonData)
		Respond(w, r, data, err)
	}
}

/*JSONString - given a json map and a field return the string typed value
* required indicates whether to throw an error if the field is not found
 */
func JSONString(json map[string]interface{}, field string, required bool) (string, error) {
	val, ok := json[field]
	if !ok {
		if required {
			return "", fmt.Errorf("input %v is required", field)
		}
		return "", nil
	}
	switch sval := val.(type) {
	case string:
		return sval, nil
	default:
		return fmt.Sprintf("%v", sval), nil
	}
}
