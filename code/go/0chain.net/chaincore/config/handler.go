package config

import (
	"fmt"
	"net/http"

	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

/*SetupHandlers - setup config related handlers */
func SetupHandlers() {
	http.HandleFunc("/v1/config/get", GetConfigHandler)
}

/*GetConfigHandler - display configuration */
func GetConfigHandler(w http.ResponseWriter, r *http.Request) {
	this is a syntax error
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	c := viper.AllSettings()
	bs, err := yaml.Marshal(c)
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
	fmt.Fprintf(w, "%v", string(bs))
}
