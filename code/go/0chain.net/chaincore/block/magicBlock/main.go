package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strconv"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

type cmdMagicBlock struct {
	// magic block instance
	block *block.MagicBlock
	// yaml config instance
	yml *configYaml
	// dkgs collection
	dkgs map[string]*bls.DKG
	// summaries collection
	summaries map[int]*bls.DKGSummary
}

func new() *cmdMagicBlock {
	return &cmdMagicBlock{dkgs: map[string]*bls.DKG{}, summaries: map[int]*bls.DKGSummary{}}
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
	mb.K = int(math.Ceil(float64(cmd.yml.KPercent) / 100.0 * float64(mb.N)))
	mb.T = int(math.Ceil(float64(cmd.yml.TPercent) / 100.0 * float64(mb.N)))
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
			mpk.Mpk = append(mpk.Mpk, v.GetHexString())
		}
		cmd.block.Mpks.Mpks[id] = mpk
	}
}

// createShareOrSigns method add new group of share or sign
func (cmd *cmdMagicBlock) createShareOrSigns() {
	cmd.block.ShareOrSigns = block.NewGroupSharesOrSigns()
	cmd.setupDKGSummaries()
	for mid := range cmd.block.Miners.NodesMap {
		sos := block.NewShareOrSigns()
		sos.ID = mid
		partyId := bls.ComputeIDdkg(mid)
		for id, node := range cmd.block.Miners.NodesMap {
			otherPartyId := bls.ComputeIDdkg(id)
			share, err := cmd.dkgs[mid].ComputeDKGKeyShare(otherPartyId)
			if err != nil {
				panic(err)
			}
			cmd.summaries[node.SetIndex].SecretShares[partyId.GetHexString()] = share.GetHexString()
			if mid != id {
				var privateKey bls.Key
				privateKey.SetHexString(cmd.yml.MinersMap[id].PrivateKey)
				message := encryption.Hash(share.GetHexString())
				sos.ShareOrSigns[id] = &bls.DKGKeyShare{
					Message: message,
					Sign:    privateKey.Sign(message).GetHexString(),
				}
			}
		}
		cmd.block.ShareOrSigns.Shares[mid] = sos
	}
}

// setupDKGSummaries initializes the dkg summaries
func (cmd *cmdMagicBlock) setupDKGSummaries() {
	cmd.block.ShareOrSigns = block.NewGroupSharesOrSigns()
	for _, n := range cmd.block.Miners.NodesMap {
		dkg := &bls.DKGSummary{
			SecretShares:  make(map[string]string),
			StartingRound: cmd.block.StartingRound,
		}
		dkg.ID = strconv.FormatInt(cmd.block.MagicBlockNumber, 10)
		cmd.summaries[n.SetIndex] = dkg
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

// saveDKGSummaries method saves the dkg summaries on file storage
func (cmd *cmdMagicBlock) saveDKGSummaries() error {
	for _, n := range cmd.block.Miners.NodesMap {
		file, err := json.MarshalIndent(cmd.summaries[n.SetIndex], "", " ")
		if err != nil {
			return err
		}
		filename := fmt.Sprintf("b0mnode%v_dkg.json", n.SetIndex+1)
		if cmd.yml.DKGSummaryFilename != "" {
			filename = fmt.Sprintf("b0mnode%v_%v_dkg.json", n.SetIndex+1, cmd.yml.DKGSummaryFilename)
		}
		path := "/0chain/go/0chain.net/docker.local/config/" + filename
		if err := ioutil.WriteFile(path, file, 0644); err != nil {
			return err
		}
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
	log.Printf("Success: Magic block created")
	if err := cmd.saveDKGSummaries(); err != nil {
		log.Printf("Error writing json file: %v\n", err.Error())
		return
	}
	log.Printf("Success: DKG summaries created")
}
