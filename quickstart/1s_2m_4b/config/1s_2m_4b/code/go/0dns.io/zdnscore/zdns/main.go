package main

import (
	"0dns.io/core/state"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"0dns.io/core/common"
	"0dns.io/core/config"
	"0dns.io/core/logging"
	. "0dns.io/core/logging"

	"0dns.io/zdnscore/worker"

	"github.com/0chain/gosdk/core/block"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func initializeConfig() {
	config.Configuration.ChainID = viper.GetString("server_chain.id")
	config.Configuration.SignatureScheme = viper.GetString("server_chain.signature_scheme")
	config.Configuration.Port = viper.GetInt("port")

	config.Configuration.MagicBlockWorkerTimerInSeconds = viper.GetInt64("worker.magic_block_worker")

	config.Configuration.UseHTTPS = viper.GetBool("use_https")
	config.Configuration.UsePath = viper.GetBool("use_path")
}

func initializeMagicBlock(magicBlockFile string) {
	magicBlock, err := os.Open(magicBlockFile)
	if err != nil {
		Logger.Error("Failed to read magic block with error", zap.Error(err))
		panic("unable to read magic block file")
	}

	magicBlockBytes, err := ioutil.ReadAll(magicBlock)
	if err != nil {
		Logger.Error("Failed to read magic block with error", zap.Error(err))
		panic("unable to read magic block file")
	}

	var m block.MagicBlock
	err = json.Unmarshal(magicBlockBytes, &m)
	if err != nil {
		Logger.Error("Failed to unmarshal magic block bytes", zap.Error(err))
		panic("Unable to unmarshal magic block bytes")
	}

	state.SetFromCurrentMagicBlock(config.Configuration, &m)
}

func initHandlers(r *mux.Router) {

	r.Use(useCors)

	r.HandleFunc("/", common.UserRateLimit(HomePageHandler))
	r.HandleFunc("/network", common.UserRateLimit(NetworkDetailsHandler))
	r.HandleFunc("/magic_block", common.UserRateLimit(LatestMagicBlockHandler))
}

var startTime time.Time

func main() {
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	magicBlockFile := flag.String("magic_block", "", "magic_block")
	configDir := flag.String("config_dir", "", "config dir")
	logDir := flag.String("log_dir", "", "log output dir")
	flag.Parse()

	config.Configuration.DeploymentMode = byte(*deploymentMode)
	config.SetupDefaultConfig()
	config.SetupConfig(*configDir)

	if config.Development() {
		logging.InitLogging("development", *logDir)
	} else {
		logging.InitLogging("production", *logDir)
	}
	initializeConfig()

	common.SetupRootContext(context.Background())

	//checkForDBConnection(context.Background())

	initializeMagicBlock(*magicBlockFile)

	address := fmt.Sprintf(":%v", config.Configuration.Port)

	var server *http.Server
	r := mux.NewRouter()

	if config.Development() {
		server = &http.Server{
			Addr:           address,
			ReadTimeout:    30 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        r,
		}
	} else {
		server = &http.Server{
			Addr:           address,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        r,
		}
	}
	common.HandleShutdown(server)

	common.ConfigRateLimits()
	initHandlers(r)
	worker.SetupWorkers(context.Background())

	startTime = time.Now().UTC()
	Logger.Info("Ready to listen to the requests on ", zap.Any("port", config.Configuration.Port))
	log.Fatal(server.ListenAndServe())
}

func HomePageHandler(w http.ResponseWriter, r *http.Request) {
	s := state.Get()

	fmt.Fprintf(w, "<div>Running since %v ...\n", startTime)
	fmt.Fprintf(w, "<div>Working on the chain: %v</div>\n", config.Configuration.ChainID)
	fmt.Fprintf(w, "<div>I am 0dns with <ul><li>miners:%v</li><li>sharders:%v</li></ul></div>\n",
		s.Miners, s.Sharders)
}

func NetworkDetailsHandler(w http.ResponseWriter, r *http.Request) {
	var response struct {
		Miners   []string `json:"miners"`
		Sharders []string `json:"sharders"`
	}

	s := state.Get()

	response.Miners = s.Miners
	response.Sharders = s.Sharders

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func LatestMagicBlockHandler(w http.ResponseWriter, r *http.Request) {
	magicBlock := state.Get().CurrentMagicBlock

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(magicBlock)
}
