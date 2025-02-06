package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"gopkg.in/yaml.v2"
)

type cmdMagicBlock struct {
	// magic block instance
	block *block.MagicBlock
	// yaml config instance
	yml *configYaml
	// dkgs collection
	dkgs map[string]*bls.DKG
	// summaries collection
	summaries       map[int]*bls.DKGSummary
	originalIndices map[string]int
}

var (
	defaultTokenSize int64 = 10000000000
	rootPath               = "/config"

	output = fmt.Sprintf("%v/output", rootPath)
	input  = fmt.Sprintf("%v/input", rootPath)
)

func new() *cmdMagicBlock {
	return &cmdMagicBlock{dkgs: map[string]*bls.DKG{}, summaries: map[int]*bls.DKGSummary{}}
}

// setupYaml method initalizes a configuration file based on yaml
func (cmd *cmdMagicBlock) setupYaml(config string) error {
	cmd.yml = newYaml()
	fPath := fmt.Sprintf("%v.yaml", config)
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
	mb.T = int(math.Ceil(float64(mb.N) * (float64(cmd.yml.TPercent) / 100.0)))
	mb.K = int(math.Ceil(float64(mb.N) * (float64(cmd.yml.KPercent) / 100.0)))
	cmd.block = mb
}

func (cmd *cmdMagicBlock) setupNodes() error {
	for _, v := range cmd.yml.Miners {
		cmd.yml.MinersMap[v.ID] = v
		v.CreationDate = common.Now()
		v.Type = cmd.block.Miners.Type
		if err := cmd.block.Miners.AddNode(&v.Node); err != nil {
			return err
		}
	}
	for _, v := range cmd.yml.Sharders {
		cmd.yml.ShardersMap[v.ID] = v
		v.CreationDate = common.Now()
		v.Type = cmd.block.Sharders.Type
		if err := cmd.block.Sharders.AddNode(&v.Node); err != nil {
			return err
		}
	}
	return nil
}

// setupMPKS setups
func (cmd *cmdMagicBlock) setupMPKS() {
	cmd.block.Mpks = block.NewMpks()
	for id := range cmd.block.Miners.NodesMap {
		cmd.dkgs[id] = bls.MakeDKG(cmd.block.T, cmd.block.N, id)
		mpk := &block.MPK{ID: id}
		for _, v := range cmd.dkgs[id].GetMPKs() {
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
		for id, nd := range cmd.block.Miners.NodesMap {
			otherPartyId := bls.ComputeIDdkg(id)
			share, err := cmd.dkgs[mid].ComputeDKGKeyShare(otherPartyId)
			if err != nil {
				log.Panic(err)
			}
			cmd.summaries[nd.SetIndex].SecretShares[partyId.GetHexString()] = share.GetHexString()
			if mid != id {
				var privateKey bls.Key
				privateKeyBytes, err := hex.DecodeString(cmd.yml.MinersMap[id].PrivateKey)
				if err != nil {
					log.Panic(err)
				}

				if err := privateKey.SetLittleEndian(privateKeyBytes); err != nil {
					log.Panic(err)
				}

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

func verifyKeys(hexSecKey, hexPubKey, hexId string) error {
	var privateKey bls.Key
	if len(hexSecKey) > 0 {
		privateKeyBytes, _ := hex.DecodeString(hexSecKey)
		if err := privateKey.SetLittleEndian(privateKeyBytes); err != nil {
			fmt.Println(err.Error())
			return errors.New("sec key is not valid")
		}
		pubRaw := privateKey.GetPublicKey()
		pub := pubRaw.SerializeToHexStr()

		if pub != hexPubKey {
			return errors.New("pub keys is not valid")
		}
	}

	decodeString, _ := hex.DecodeString(hexPubKey)
	id := encryption.Hash(decodeString)
	if id != hexId {
		return errors.New("id is not valid")
	}
	return nil
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
func (cmd *cmdMagicBlock) jsonMB() ([]byte, error) { //nolint
	return json.MarshalIndent(cmd.block, "", " ")
}

// saveBlock method saves a magic block on file storage
func (cmd *cmdMagicBlock) saveBlock() error {
	file, err := json.MarshalIndent(cmd.block, "", " ")
	if err != nil {
		return err
	}
	name := getMagicBlockFileName(cmd.yml.MagicBlockFilename)
	path := fmt.Sprintf("%v/%v", output, name)
	if err := os.WriteFile(path, file, 0644); err != nil {
		return err
	}
	return nil
}

func getMagicBlockFileName(name string) string {
	return fmt.Sprintf("%v.json", name)
}

// saveDKGSummaries method saves the dkg summaries on file storage
func (cmd *cmdMagicBlock) saveDKGSummaries() error {
	for _, n := range cmd.block.Miners.NodesMap {
		name := getSummariesName(n.SetIndex)
		if _, err := cmd.saveDKGSummary(n.SetIndex, name); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *cmdMagicBlock) saveDKGSummary(index int, name string) (string, error) {
	file, err := json.MarshalIndent(cmd.summaries[index], "", " ")
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("%v/%v", output, name)
	if err := os.WriteFile(path, file, 0644); err != nil {
		return "", err
	}
	return path, nil
}

func (cmd *cmdMagicBlock) removeDKGSummary(name string) error {
	path := fmt.Sprintf("%v/%v", output, name)
	return os.Remove(path)
}

func getSummariesName(index int) string {
	return fmt.Sprintf("b0mnode%v_dkg.json", index+1)
}

func main() {
	magicBlockConfig := flag.String("config_file", "", "config_file")
	mainnet := flag.Bool("mainnet", false, "mainnet")
	logging.InitLogging("development", "")

	flag.Parse()

	var emails []string
	if *mainnet {
		log.Println("Preparing files...")

		passes := loadPasswords()
		for e := range passes {
			emails = append(emails, e)
		}
		configs, nodesToEmail := readConfigs(magicBlockConfig, passes)
		merged, origInd := mergeConfigs(configs)
		mbfile := fmt.Sprintf("%v/%v", output, *magicBlockConfig)
		writeMergedYAml(&mbfile, merged)
		artifacts, err := generateArtifacts(&mbfile, emails)
		if err != nil {
			log.Panic(err)
		}
		artifacts.originalIndices = origInd
		generateStates(artifacts)
		zipArtifacts(passes, nodesToEmail, artifacts)
	} else {
		mbfile := fmt.Sprintf("%v/%v", rootPath, *magicBlockConfig)
		output = rootPath
		if _, err := generateArtifacts(&mbfile, emails); err != nil {
			log.Panic(err)
		}
	}

	log.Printf("Now sleeping for 60 sec")
	time.Sleep(60 * time.Second)
}

func generateStates(artifacts *cmdMagicBlock) {
	fmt.Println("Generating initial states")
	path := fmt.Sprintf("%v/%v", input, getStatesFileName())

	file, err := os.ReadFile(path)
	states := &state.InitStates{}
	if err == nil {
		err = yaml.Unmarshal(file, states)
		if err != nil {
			log.Panic(err)
		}
	} else if !os.IsNotExist(err) {
		log.Panic(err)
	}

	for _, miner := range artifacts.block.Miners.Nodes {
		s := state.InitState{
			ID:     miner.ID,
			Tokens: currency.Coin(defaultTokenSize),
		}

		states.States = append(states.States, s)
	}

	marshal, err := yaml.Marshal(states)
	if err != nil {
		log.Panic(err)
	}

	outPath := fmt.Sprintf("%v/%v", output, getStatesFileName())
	if err := os.WriteFile(outPath, marshal, 0755); err != nil {
		log.Panic(err)
	}

}

func getStatesFileName() string {
	return "initial-states.yaml"
}

func zipArtifacts(passes map[string]string, nodesToEmail map[datastore.Key]string, cmd *cmdMagicBlock) {
	log.Println("Preparing archives...")
	//rename summaries for test purpose use
	for _, miner := range cmd.block.Miners.Nodes {
		name := getSummariesName(miner.SetIndex)
		path := fmt.Sprintf("%v/%v", output, name)
		newPath := fmt.Sprintf("%v/%v_%v", output, miner.ID[:8], name)
		if err := os.Rename(path, newPath); err != nil {
			return
		}
	}
	for email, pass := range passes {
		var summaries []string
		mappedNames := make(map[string]string)

		for _, miner := range cmd.block.Miners.Nodes {
			if nodesToEmail[miner.ID] == email {
				index := cmd.originalIndices[miner.ID]
				name := getSummariesName(index)
				_, err := cmd.saveDKGSummary(miner.SetIndex, name)
				if err != nil {
					log.Panic(err)
				}
				summaries = append(summaries, name)
				mappedNames[name] = miner.ID
			}
		}

		log.Printf("collected %v dkg summaries for %v", len(summaries), email)

		writeNames(mappedNames)

		file := fmt.Sprintf("%v.zip", email)
		args := []string{"-e", file, getMagicBlockFileName(cmd.yml.MagicBlockFilename)}
		args = append(args, summaries...)
		args = append(args, getStatesFileName())
		args = append(args, getNamesFileName())
		args = append(args, "--password", pass)

		c := exec.Command("zip", args...)
		c.Dir = output
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		fmt.Printf("Creating %v\n", file)

		err := c.Run()
		if err != nil {
			log.Panic(err)
		}

		renameNames(email)
		for _, name := range summaries {
			if err := cmd.removeDKGSummary(name); err != nil {
				log.Panic(err)
			}
		}
	}
}

func renameNames(email string) {
	pathNames := fmt.Sprintf("%v/%v", output, "names.yaml")
	pathNamesNew := fmt.Sprintf("%v/%v", output, getNamesEmailFileName(email))
	err := os.Rename(pathNames, pathNamesNew)
	if err != nil {
		log.Panic(err)
	}
}

func writeNames(names map[string]string) string {
	path := fmt.Sprintf("%v/%v", output, "names.yaml")
	y := yamlNames{Names: names}
	marshal, err := yaml.Marshal(y)
	if err != nil {
		log.Panic(err)
	}
	if err := os.WriteFile(path, marshal, 0755); err != nil {
		log.Panic(err)
	}
	return path
}

func getNamesFileName() string {
	return "names.yaml"
}

func getNamesEmailFileName(email string) string {
	return fmt.Sprintf("%v_names.yaml", email)
}

func generateArtifacts(magicBlockConfig *string, emails []string) (*cmdMagicBlock, error) {
	cmd := new()
	if err := cmd.setupYaml(*magicBlockConfig); err != nil {
		log.Printf("Failed to read configuration file (%v) for magicBlock. Error: %v\n", *magicBlockConfig, err)
		return nil, err
	}
	client.SetClientSignatureScheme("bls0chain")
	cmd.setupBlock()
	if err := cmd.setupNodes(); err != nil {
		log.Printf("Failed to setup nodes, %v", err)
		return nil, err
	}
	cmd.setupMPKS()
	cmd.createShareOrSigns()

	cmd.setupBlockHash()

	if err := cmd.saveBlock(); err != nil {
		log.Printf("Error writing json file: %v\n", err.Error())
		return nil, err
	}
	log.Printf("Success: Magic block created")
	if err := cmd.saveDKGSummaries(); err != nil {
		log.Printf("Error writing json file: %v\n", err.Error())
		return nil, err
	}
	log.Printf("Success: DKG summaries created")
	return cmd, nil
}

func readConfigs(magicBlockConfig *string, passes map[string]string) ([]*configYaml, map[datastore.Key]string) {
	var configs []*configYaml
	fmt.Println("unzipping archives")

	nodesToEmail := make(map[datastore.Key]string)
	nodesYaml := fmt.Sprintf("%v/%v.yaml", input, *magicBlockConfig)
	for email, pass := range passes {
		file := fmt.Sprintf("%v.zip", email)
		fmt.Printf("unzipping %v\n", file)

		cmd := exec.Command("unzip", "-P", pass, file)
		cmd.Dir = input
		cmd.Stdin = bytes.NewReader([]byte("y"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			log.Panic(err)
		}
		err = cmd.Wait()
		if err != nil {
			log.Panic(err)
		}

		conf := newYaml()
		err = conf.readYaml(nodesYaml)
		if err != nil {
			log.Panic(err)
		}

		for _, m := range conf.Miners {
			if err := verifyKeys(m.PrivateKey, m.PublicKey, m.ID); err != nil {
				fmt.Printf("bad miner %v\n", m.ID)
				log.Panic(err)
			}
			if !isValidDescription(m.Description) {
				fmt.Printf("miner %v has too long description\n", m.ID)
				log.Panic(err)
			}
			nodesToEmail[m.ID] = email
		}
		for _, s := range conf.Sharders {
			if err := verifyKeys(s.PrivateKey, s.PublicKey, s.ID); err != nil {
				fmt.Printf("bad sharder with %v\n", s.ID)
				log.Panic(err)
			}
			if !isValidDescription(s.Description) {
				fmt.Printf("sharder %v has too long description\n", s.ID)
				log.Panic(err)
			}
			nodesToEmail[s.ID] = email
		}
		configs = append(configs, conf)

		e := os.Remove(nodesYaml)
		if e != nil {
			log.Panic(err)
		}
	}
	fmt.Printf("parsed %v nodes.yaml files\n", len(configs))
	return configs, nodesToEmail
}

func isValidDescription(s string) bool {
	words := strings.Fields(s)
	return len(words) < 200
}

func writeMergedYAml(magicBlockConfig *string, merged *configYaml) {
	fmt.Println("writing yaml file")
	mergedYaml := fmt.Sprintf("%v.yaml", *magicBlockConfig)
	marshal, err := yaml.Marshal(merged)
	if err != nil {
		log.Panic(err)
	}

	_, err = os.Stat(output)
	if err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(output, 0755); err != nil {
			log.Panic(err)
		}
	}

	if err := os.WriteFile(mergedYaml, marshal, 0755); err != nil {
		log.Panic(err)
	}
}

func mergeConfigs(configs []*configYaml) (*configYaml, map[string]int) {
	fmt.Println("merging yaml file")
	origInd := make(map[string]int)
	merged := newYaml()
	merged.MagicBlockNumber = 1
	merged.StartingRound = 0
	merged.StartingRound = 0
	merged.TPercent = 75
	merged.KPercent = 81
	merged.MagicBlockFilename = "b0magicBlock"
	merged.DKGSummaryFilename = "dkg_summary"

	index := 0
	for _, conf := range configs {
		for _, miner := range conf.Miners {
			merged.Miners = append(merged.Miners, miner)
			merged.MinersMap[miner.ID] = miner
			origInd[miner.ID] = miner.SetIndex
			miner.SetIndex = index
			index++
		}

		for _, sharder := range conf.Sharders {
			merged.Sharders = append(merged.Sharders, sharder)
			merged.ShardersMap[sharder.ID] = sharder
		}
	}

	return merged, origInd
}

func loadPasswords() map[string]string {
	fmt.Println("Loading passwords from password.txt")

	passPath := fmt.Sprintf("%v/input/%v", rootPath, "password.yaml")
	passFile, err := os.ReadFile(passPath)
	if err != nil {
		log.Panic(err)
	}

	data := make(map[interface{}]interface{})
	err2 := yaml.Unmarshal(passFile, &data)
	if err2 != nil {
		log.Panic(err2)
	}

	res := make(map[string]string)

	for k, v := range data {
		res[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
	}
	return res
}
