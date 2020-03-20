package common

import (
	. "0chain.net/core/logging"
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
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
				Logger.Error("panic", zap.Any("error", err))
				w.Header().Set("Content-Type", "application/json")
				data := make(map[string]interface{}, 2)
				data["error"] = fmt.Sprintf("%v", err)
				if are, ok := err.(*Error); ok {
					data["code"] = are.Code
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
