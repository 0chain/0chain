package main

import (
	"gopkg.in/yaml.v2"
	"os"

	"0chain.net/chaincore/node"
)

type yamlNode struct {
	node.Node  `yaml:",inline"`
	PrivateKey string `yaml:"private_key"`
}

type yamlNames struct {
	Names map[string]string `yaml:"names"`
}

// yaml config file structure
type configYaml struct {
	Miners             []*yamlNode          `yaml:"miners"`
	MinersMap          map[string]*yamlNode `yaml:"-"`
	Sharders           []*yamlNode          `yaml:"sharders"`
	ShardersMap        map[string]*yamlNode `yaml:"-"`
	MagicBlockNumber   int64                `yaml:"magic_block_number"`
	StartingRound      int64                `yaml:"starting_round"`
	TPercent           int                  `yaml:"t_percent"`
	KPercent           int                  `yaml:"k_percent"`
	MagicBlockFilename string               `yaml:"magic_block_filename"`
	DKGSummaryFilename string               `yaml:"dkg_summary_filename"`
}

func newYaml() *configYaml {
	return &configYaml{MinersMap: make(map[string]*yamlNode), ShardersMap: make(map[string]*yamlNode)}
}

func (c *configYaml) readYaml(file string) error {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return err
	}
	return nil
}
