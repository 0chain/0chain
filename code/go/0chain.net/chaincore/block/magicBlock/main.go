package main

import (
    "encoding/json"
    "fmt"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "log"
    "flag"

    "0chain.net/chaincore/block"
    "0chain.net/chaincore/node"
    "0chain.net/chaincore/threshold/bls"
    "0chain.net/core/common"
)

func main() {
    magicBlockConfig := flag.String("config_file", "", "config_file")
    flag.Parse()
    if *magicBlockConfig != "" {
        c := NewConfigYaml()
        mb := block.NewMagicBlock()
        dkgs := make(map[string]*bls.DKG)
        err := c.readYaml(fmt.Sprintf("/0chain/go/0chain.net/docker.local/config/%v.yaml",*magicBlockConfig))
        if err == nil {
            setupMagicBlock(mb, c)
            setupNodes(mb, c)
            createMPKS(mb, dkgs)
            createShareOrSigns(mb, dkgs, c)

            mb.Hash = mb.GetHash()
            file, _ := json.MarshalIndent(mb, "", " ")
            err := ioutil.WriteFile(fmt.Sprintf("/0chain/go/0chain.net/docker.local/config/%v.json", c.MagicBlockFilename), file, 0644)
            if err != nil {
                log.Printf("Error writing json file: %v\n", err.Error())
            }
        } else {
            log.Printf("Failed to read configuration file (%v) for magicBlock. Error: %v\n", *magicBlockConfig, err)
        }
    } else {
        log.Println("Failed to find configuration file for magicBlock")
    }
}

func setupMagicBlock(mb *block.MagicBlock, c *configYaml) {
    mb.Miners = node.NewPool(node.NodeTypeMiner)
    mb.Sharders = node.NewPool(node.NodeTypeSharder)

    mb.MagicBlockNumber = c.MagicBlockNumber
    mb.StartingRound = c.StartingRound
    mb.N = len(c.Miners)
    mb.T = int(float64(mb.N) * (float64(c.TPercent) / 100.0))
    mb.K = int(float64(mb.N) * (float64(c.KPercent) / 100.0))
}

func setupNodes(mb *block.MagicBlock, c *configYaml) {
    for _, v := range c.Miners {
        c.MinersMap[v.ID] = v
        v.CreationDate = common.Now()
        v.Type = mb.Miners.Type
        mb.Miners.AddNode(&v.Node)
    }
    for _, v := range c.Sharders {
        c.ShardersMap[v.ID] = v
        v.CreationDate = common.Now()
        v.Type = mb.Sharders.Type
        mb.Sharders.AddNode(&v.Node)
    }
}

func createMPKS(mb *block.MagicBlock, dkgs map[string]*bls.DKG) {
    mb.Mpks = block.NewMpks()
    for id := range mb.Miners.NodesMap {
        dkgs[id] = bls.MakeDKG(mb.T, mb.N, id)
        mpk := &block.MPK{ID: id}
        for _, v := range dkgs[id].Mpk {
            mpk.Mpk = append(mpk.Mpk, v.SerializeToHexStr())
        }
        mb.Mpks.Mpks[id] = mpk
    }
}

func createShareOrSigns(mb *block.MagicBlock, dkgs map[string]*bls.DKG, c *configYaml) {
    mb.ShareOrSigns = block.NewGroupSharesOrSigns()
    for mid, _ := range mb.Miners.NodesMap {
        ss := block.NewShareOrSigns()
        ss.ID = mid
        var privateKey bls.Key
        privateKey.SetHexString(c.MinersMap[mid].PrivateKey)
        for id := range mb.Miners.NodesMap {
            otherPartyId := bls.ComputeIDdkg(id)
            share, _ := dkgs[mid].ComputeDKGKeyShare(otherPartyId)
            ss.ShareOrSigns[id] = &bls.DKGKeyShare{Message: c.Message, Share: share.GetHexString(), Sign: privateKey.Sign(c.Message).SerializeToHexStr()}
        }
        mb.ShareOrSigns.Shares[mid] = ss
    }
}

type configYaml struct {
    Miners             []*yamlNode `yaml:"miners"`
    MinersMap          map[string]*yamlNode
    Sharders           []*yamlNode `yaml:"sharders"`
    ShardersMap        map[string]*yamlNode
    Message            string `yaml:"message"`
    MagicBlockNumber   int64  `yaml:"magic_block_number"`
    StartingRound      int64  `yaml:"starting_round"`
    TPercent           int    `yaml:"t_percent"`
    KPercent           int    `yaml:"k_percent"`
    MagicBlockFilename string `yaml:"magic_block_filename"`
}

func NewConfigYaml() *configYaml {
    return &configYaml{MinersMap: make(map[string]*yamlNode), ShardersMap: make(map[string]*yamlNode)}
}

func (c *configYaml) readYaml(file string) error {
    yamlFile, err := ioutil.ReadFile(file)
    if err != nil {
        return err
    }
    err = yaml.Unmarshal(yamlFile, c)
    if err != nil {
        return err
    }
    return nil
}

type yamlNode struct {
    node.Node        `yaml:",inline"`
    PrivateKey string `yaml:"private_key"`
}
