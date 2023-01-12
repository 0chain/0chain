package common

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

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
		if errors.Is(err, ErrNotModified) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		data := make(map[string]interface{}, 2)
		data["error"] = err.Error()
		if cErr, ok := err.(*Error); ok {
			data["code"] = cErr.Code
		}

		switch {
		case errors.Is(err, ErrBadRequest):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, ErrInternal):
			w.WriteHeader(http.StatusInternalServerError)
		case errors.Is(err, ErrNoResource):
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}

		if err := json.NewEncoder(w).Encode(data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if data != nil {
		w.Header().Set("Content-Type", "application/json")
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			if err := json.NewEncoder(w).Encode(data); err != nil {
				Error500(w, "json encode failed")
				return
			}
		} else {
			w.Header().Set("Content-Encoding", "gzip")
			gzw := gzip.NewWriter(w)
			defer gzw.Close()
			if err := json.NewEncoder(gzw).Encode(data); err != nil {
				Error500(w, "json encode failed")
				return
			}
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Error500 response with a 500 error and msg
func Error500(w http.ResponseWriter, msg string) {
	errorAny(w, 500, msg)
}

// errorAny writes error message with status code
func errorAny(w http.ResponseWriter, status int, msg string) {
	httpMsg := fmt.Sprintf("%d %s", status, http.StatusText(status))
	if msg != "" {
		httpMsg = fmt.Sprintf("%s - %s", httpMsg, msg)
	}

	http.Error(w, httpMsg, status)
}

func getContext(r *http.Request) (context.Context, error) {
	ctx := r.Context()
	return ctx, nil
}

func SetupCORSResponse(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
