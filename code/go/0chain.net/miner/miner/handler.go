package main

import (
	"fmt"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
)

const updateConfigURL = "/v1/config/update"
const updateConfigAllURL = "/v1/config/update_all"

/*SetupHandlers - setup update config related handlers */
func SetupHandlers() {
	if config.Development() {
		http.HandleFunc("/_hash", common.Recover(encryption.HashHandler))
		http.HandleFunc("/_sign", common.Recover(common.ToJSONResponse(encryption.SignHandler)))
		http.HandleFunc(updateConfigURL, common.Recover(ConfigUpdateHandler))
		http.HandleFunc(updateConfigAllURL, common.Recover(ConfigUpdateAllHandler))
	}
}

/*ConfigUpdateHandler - update this miner's configuration */
func ConfigUpdateHandler(w http.ResponseWriter, r *http.Request) {
	updateConfig(w, r, updateConfigURL)
}

/*ConfigUpdateAllHandler - update all miners' configuration */
func ConfigUpdateAllHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logging.Logger.Error("failed to parse update config form", zap.Any("error", err))
		return
	}
	mb := chain.GetServerChain().GetCurrentMagicBlock()
	miners := mb.Miners.Nodes
	for _, miner := range miners {
		if node.Self.Underlying().PublicKey != miner.PublicKey {
			go func(miner *node.Node) {
				resp, err := http.PostForm(miner.GetN2NURLBase()+updateConfigURL, r.Form)
				if err != nil {
					logging.Logger.Error("failed to update other miner's config", zap.Any("miner", miner.GetKey()), zap.Any("response", resp), zap.Any("error", err))
					return
				}
				defer resp.Body.Close()
			}(miner)
		}
	}
	updateConfig(w, r, updateConfigAllURL)
}

/*updateConfig - updates the configuation for a particular miner and returns the user to the same page */
func updateConfig(w http.ResponseWriter, r *http.Request, updateUrl string) {
	newGenTimeout, _ := strconv.Atoi(r.FormValue("generate_timeout"))
	if newGenTimeout > 0 {
		chain.GetServerChain().SetGenerationTimeout(newGenTimeout)
		viper.Set("server_chain.block.generation.timeout", newGenTimeout)
	}
	newTxnWaitTime, _ := strconv.Atoi(r.FormValue("txn_wait_time"))
	if newTxnWaitTime > 0 {
		chain.GetServerChain().SetRetryWaitTime(newTxnWaitTime)
		viper.Set("server_chain.block.generation.retry_wait_time", newTxnWaitTime)
	}
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	fmt.Fprintf(w, "<form action='%s' method='post'>", updateUrl)
	fmt.Fprintf(w, "Generation Timeout (time till a miner makes a block with less than max blocksize): <input type='text' name='generate_timeout' value='%v'><br>", viper.Get("server_chain.block.generation.timeout"))
	fmt.Fprintf(w, "Retry Wait Time (time miner waits if there aren't enough transactions to reach max blocksize): <input type='text' name='txn_wait_time' value='%v'><br>", viper.Get("server_chain.block.generation.retry_wait_time"))
	fmt.Fprintf(w, "<input type='submit' value='Submit'>")
	fmt.Fprintf(w, "</form>")
}
