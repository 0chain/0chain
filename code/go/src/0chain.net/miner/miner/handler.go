package main

import (
	"fmt"
	"net/http"
	"strconv"

	"0chain.net/chain"
	"0chain.net/transaction"
	"github.com/spf13/viper"
)

/*SetupHandlers - setup config related handlers */
func SetupHandlers() {
	http.HandleFunc("/v1/miner/updateConfig", UpdateConfig)
}

/*SetConfig*/
func UpdateConfig(w http.ResponseWriter, r *http.Request) {
	newGenTimeout, _ := strconv.Atoi(r.FormValue("generate_timeout"))
	if newGenTimeout > 0 {
		chain.GetServerChain().SetGenerationTimeout(newGenTimeout)
		viper.Set("server_chain.generate_timeout", newGenTimeout)
	}
	newTxnTimeout, _ := strconv.ParseInt(r.FormValue("txn_timeout"), 10, 64)
	if newTxnTimeout > 0 {
		transaction.SetTxnTimeout(newTxnTimeout)
		viper.Set("server_chain.txn_timeout", newTxnTimeout)
	}
	newGenTxnRate, _ := strconv.ParseInt(r.FormValue("generate_txn"), 10, 32)
	if newGenTxnRate > 0 {
		SetTxnGenRate(int32(newGenTxnRate))
		viper.Set("server_chain.generate_txn", newGenTxnRate)
	}
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	fmt.Fprintf(w, "<form action='/v1/miner/updateConfig' method='post'>")
	fmt.Fprintf(w, "Generation Timeout: <input type='text' name='generate_timeout' value='%v'><br>", viper.Get("server_chain.generate_timeout"))
	fmt.Fprintf(w, "Transaction Timeout: <input type='text' name='txn_timeout' value='%v'><br>", viper.Get("server_chain.txn_timeout"))
	fmt.Fprintf(w, "Transaction Generation Rate: <input type='text' name='generate_txn' value='%v'><br>", viper.Get("server_chain.generate_txn"))
	fmt.Fprintf(w, "<input type='submit' value='Submit'>")
	fmt.Fprintf(w, "</form>")
}
