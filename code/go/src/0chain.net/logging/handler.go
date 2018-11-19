package logging

import (
	"net/http"
	"strconv"
)

/*LogWriter - a handler to get recent logs*/
func LogWriter(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	detailLevel, _ := strconv.Atoi(queryValues.Get("detail"))
	MLogger.WriteLogs(w, detailLevel)
}

/*N2NLogWriter - a handler to get recent node to node logs*/
func N2NLogWriter(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	detailLevel, _ := strconv.Atoi(queryValues.Get("detail"))
	N2NMLogger.WriteLogs(w, detailLevel)
}

func MemLogWriter(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()
	detailLevel, _ := strconv.Atoi(queryValues.Get("detail"))
	MMLogger.WriteLogs(w, detailLevel)
}
