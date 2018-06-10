package config

import (
	"fmt"

	"0chain.net/common"
	"github.com/spf13/viper"
)

/*Config - all the config options passed from the command line*/
type Config struct {
	Host     string
	Port     int
	ChainID  string
	TestMode bool
}

/*Configuration of the system */
var Configuration Config

func init() {
	viper.SetConfigName("0chainConfig")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error cannot read the config file: %s \n", err))
	}
}

/*TestNet is the program running in TestNet mode? */
func TestNet() bool {

	return Configuration.TestMode

}

/*ErrSupportedChain error for indicating which chain is supported by the server */
var ErrSupportedChain error

/*MAIN_CHAIN - the main 0chain.net blockchain id */
const MAIN_CHAIN = "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe" // TODO:

/*GetMainChainID - get the main chain id */
func GetMainChainID() string {
	return MAIN_CHAIN
}

/*ServerChainID - the chain this server is responsible for */
var ServerChainID = ""

/*SetServerChainID  - set the chain this server is responsible for processing */
func SetServerChainID(chain string) {
	if chain == "" {
		ServerChainID = MAIN_CHAIN
	} else {
		ServerChainID = chain
	}
	ErrSupportedChain = common.NewError("supported_chain", fmt.Sprintf("chain %v is not supported by this server", chain))
}

/*GetServerChainID - get the chain this server is responsible for processing */
func GetServerChainID() string {
	if ServerChainID == "" {
		return MAIN_CHAIN
	}
	return ServerChainID
}

/*GetServerChainBlockSize - get the block size as specified in the config file */
func GetServerChainBlockSize() int32 {
	BlockSize := viper.GetInt32("BlockSize")
	return BlockSize
}

/*ValidChain - Is this the chain this server is supposed to process? */
func ValidChain(chain string) error {
	result := chain == ServerChainID || (chain == "" && ServerChainID == MAIN_CHAIN)
	if result {
		return nil
	}
	return ErrSupportedChain
}
