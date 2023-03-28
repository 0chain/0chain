package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/0chain/common/core/logging"
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
				errorMessage := fmt.Sprintf("%v", err)
				if 	strings.Contains(strings.ToLower(errorMessage), "panic") ||
					strings.Contains(strings.ToLower(errorMessage), "stack trace") {
					errorMessage = "Unknown Server Error"
				}
				data["error"] = errorMessage
				if are, ok := err.(*Error); ok {
					data["code"] = are.Code
				}
				buf := bytes.NewBuffer(nil)
				if err := json.NewEncoder(buf).Encode(data); err != nil {
					Error500(w, "json encode failed")
					return
				}

				w.WriteHeader(http.StatusInternalServerError)
				if _, err := buf.WriteTo(w); err != nil {
					logging.Logger.Error("http write failed", zap.Error(err))
				}
			}
		}()
		handler(w, r)
	}
}
