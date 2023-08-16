package main

import (
	"net/http"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

/*SetupHandlers - setup update config related handlers */
func SetupHandlers() {
	if config.Development() {
		http.HandleFunc("/_hash", common.Recover(encryption.HashHandler))
		http.HandleFunc("/_sign", common.Recover(common.ToJSONResponse(encryption.SignHandler)))
	}
}
