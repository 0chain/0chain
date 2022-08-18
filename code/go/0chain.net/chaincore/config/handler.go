package config

import (
	"fmt"
	"net/http"

	"gopkg.in/yaml.v2"

	coreEndpoint "0chain.net/core/endpoint"
	"0chain.net/core/viper"
)

/*SetupHandlers - setup config related handlers */
func SetupHandlers() {
	http.HandleFunc(coreEndpoint.GetConfig, GetConfigHandler)
}

/*GetConfigHandler - display configuration */
func GetConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	c := viper.AllSettings()
	bs, err := yaml.Marshal(c)
	if err != nil {
		fmt.Fprint(w, err.Error())
	}
	fmt.Fprintf(w, "%v", string(bs))
}
