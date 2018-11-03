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
	newTxnWaitTime, _ := strconv.Atoi(r.FormValue("txn_wait_time"))
	if newTxnWaitTime > 0 {
		chain.GetServerChain().SetTxnWaitTime(newTxnWaitTime)
		viper.Set("server_chain.txn_wait_time", newTxnWaitTime)
	}
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	fmt.Fprintf(w, "<form action='/v1/miner/updateConfig' method='post'>")
	fmt.Fprintf(w, "Generation Timeout (time till a miner makes a block with less than max blocksize): <input type='text' name='generate_timeout' value='%v'><br>", viper.Get("server_chain.generate_timeout"))
	fmt.Fprintf(w, "Transaction Timeout (time a transaction remains vaild before being put into a block): <input type='text' name='txn_timeout' value='%v'><br>", viper.Get("server_chain.txn_timeout"))
	fmt.Fprintf(w, "Transaction Generation Rate (rate the miner will add transactions to create a block): <input type='text' name='generate_txn' value='%v'><br>", viper.Get("server_chain.generate_txn"))
	fmt.Fprintf(w, "Transaction Wait Time (time miner waits if there aren't enough transactions to reach max blocksize): <input type='text' name='txn_wait_time' value='%v'><br>", viper.Get("server_chain.txn_wait_time"))
	fmt.Fprintf(w, "<input type='submit' value='Submit'>")
	fmt.Fprintf(w, "</form>")
}
