package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"0chain.net/core/logging"
	errors "github.com/0chain/errors"
	"go.uber.org/zap"
)

var (
	UseRecoverHandler = true
)

//Recover - recover after a handler panic
func Recover(handler ReqRespHandlerf) ReqRespHandlerf {
	if !UseRecoverHandler {
		return handler
	}
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logging.Logger.Error("panic handling message",
					zap.Any("error", err),
					zap.String("remote address", r.RemoteAddr),
					zap.String("method", r.Method),
					zap.String("request uri", r.RequestURI),
				)
				w.Header().Set("Content-Type", "application/json")
				data := make(map[string]interface{}, 2)

				if are, ok := err.(*errors.Error); ok {
					data["code"] = are.Code
					data["error"] = are.Error()
				} else {
					data["error"] = fmt.Sprintf("%v", err)
				}
				buf := bytes.NewBuffer(nil)
				json.NewEncoder(buf).Encode(data)
				w.WriteHeader(http.StatusInternalServerError)
				buf.WriteTo(w)
			}
		}()
		handler(w, r)
	}
}
