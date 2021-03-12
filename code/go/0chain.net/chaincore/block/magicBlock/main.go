package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
)

type cmdMagicBlock struct {
	// magic block instance
	block *block.MagicBlock
	// yaml config instance
	yml *configYaml
	// dkgs collection
	dkgs map[string]*bls.DKG
}

func new() *cmdMagicBlock {
	return &cmdMagicBlock{dkgs: map[string]*bls.DKG{}}
}

// setupYaml method initalizes a configuration file based on yaml
func (cmd *cmdMagicBlock) setupYaml(config string) error {
	cmd.yml = newYaml()
	fPath := fmt.Sprintf("/0chain/go/0chain.net/docker.local/config/%v.yaml", config)
	if err := cmd.yml.readYaml(fPath); err != nil {
		return err
	}
	return nil
}

// setupBlock method creates a new blank magic block and then fill
// all data from yaml config file
func (cmd *cmdMagicBlock) setupBlock() {
	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Sharders = node.NewPool(node.NodeTypeSharder)

	mb.MagicBlockNumber = cmd.yml.MagicBlockNumber
	mb.StartingRound = cmd.yml.StartingRound
	mb.N = len(cmd.yml.Miners)
	mb.T = int(float64(mb.N) * (float64(cmd.yml.TPercent) / 100.0))
	mb.K = int(float64(mb.N) * (float64(cmd.yml.KPercent) / 100.0))
	cmd.block = mb
}

func (cmd *cmdMagicBlock) setupNodes() {
	for _, v := range cmd.yml.Miners {
		cmd.yml.MinersMap[v.ID] = v
		v.CreationDate = common.Now()
		v.Type = cmd.block.Miners.Type
		cmd.block.Miners.AddNode(&v.Node)
	}
	for _, v := range cmd.yml.Sharders {
		cmd.yml.ShardersMap[v.ID] = v
		v.CreationDate = common.Now()
		v.Type = cmd.block.Sharders.Type
		cmd.block.Sharders.AddNode(&v.Node)
	}
}

// setupMPKS setups
func (cmd *cmdMagicBlock) setupMPKS() {
	cmd.block.Mpks = block.NewMpks()
	for id := range cmd.block.Miners.NodesMap {
		cmd.dkgs[id] = bls.MakeDKG(cmd.block.T, cmd.block.N, id)
		mpk := &block.MPK{ID: id}
		for _, v := range cmd.dkgs[id].Mpk {
			mpk.Mpk = append(mpk.Mpk, v.SerializeToHexStr())
		}
		cmd.block.Mpks.Mpks[id] = mpk
	}
}

// createShareOrSigns method add new group of share or sign
func (cmd *cmdMagicBlock) createShareOrSigns() {
	cmd.block.ShareOrSigns = block.NewGroupSharesOrSigns()
	for mid, _ := range cmd.block.Miners.NodesMap {
		ss := block.NewShareOrSigns()
		ss.ID = mid
		var privateKey bls.Key
		privateKey.SetHexString(cmd.yml.MinersMap[mid].PrivateKey)
		for id := range cmd.block.Miners.NodesMap {
			otherPartyId := bls.ComputeIDdkg(id)
			share, _ := cmd.dkgs[mid].ComputeDKGKeyShare(otherPartyId)
			ss.ShareOrSigns[id] = &bls.DKGKeyShare{
				Message: cmd.yml.Message,
				Share:   share.GetHexString(),
				Sign:    privateKey.Sign(cmd.yml.Message).SerializeToHexStr()}
		}
		cmd.block.ShareOrSigns.Shares[mid] = ss
	}
}

// setupBlockHash generates a new hash for a block
func (cmd *cmdMagicBlock) setupBlockHash() {
	cmd.block.Hash = cmd.block.GetHash()
}

// jsonMB method return indented json of block
func (cmd *cmdMagicBlock) jsonMB() ([]byte, error) {
	return json.MarshalIndent(cmd.block, "", " ")
}

// saveBlock method saves a magic block on file storage
func (cmd *cmdMagicBlock) saveBlock() error {
	file, err := json.MarshalIndent(cmd.block, "", " ")
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/0chain/go/0chain.net/docker.local/config/%v.json", cmd.yml.MagicBlockFilename)
	if err := ioutil.WriteFile(path, file, 0644); err != nil {
		return err
	}
	return nil
}

func main() {
	magicBlockConfig := flag.String("config_file", "", "config_file")
	flag.Parse()

	cmd := new()
	if err := cmd.setupYaml(*magicBlockConfig); err != nil {
		log.Printf("Failed to read configuration file (%v) for magicBlock. Error: %v\n", *magicBlockConfig, err)
		return
	}

	cmd.setupBlock()
	cmd.setupNodes()
	cmd.setupMPKS()
	cmd.createShareOrSigns()

	cmd.setupBlockHash()

	if err := cmd.saveBlock(); err != nil {
		log.Printf("Error writing json file: %v\n", err.Error())
		return
	}
	log.Printf("Success: Magic block already created")
}
