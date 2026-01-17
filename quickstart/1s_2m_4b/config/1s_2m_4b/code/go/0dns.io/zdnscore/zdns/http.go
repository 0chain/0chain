package main

import (
	"net/http"
	"strings"

	"0dns.io/core/logging"
	"go.uber.org/zap"
)

func useCors(h http.Handler) http.Handler {

	allowedOrigins := []string{"*"}

	allowedMethods := []string{"GET", "HEAD", "POST", "PUT",
		"DELETE", "OPTIONS"}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logging.Logger.Error("[recover]http", zap.String("url", r.URL.String()), zap.Any("err", err))
			}
		}()

		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Access-Control-Allow-Origin", strings.Join(allowedOrigins, ", "))
		w.Header().Add("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))

		// return directly for preflight request
		if r.Method == http.MethodOptions {
			w.Header().Add("Access-Control-Max-Age", "3600")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		h.ServeHTTP(w, r)
	})

}
