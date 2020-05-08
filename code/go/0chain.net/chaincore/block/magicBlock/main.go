package main

import (
    "encoding/hex"
    "encoding/json"
    "fmt"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "log"
    "flag"
    "sort"
    "strconv"
    "time"

    "github.com/herumi/bls/ffi/go/bls"
    "golang.org/x/crypto/sha3"
)

func main() {
    magicBlockConfig := flag.String("config_file", "", "config_file")
    flag.Parse()
    if *magicBlockConfig != "" {
        c := &configYaml{}
        mb := &magicBlock{Miners: &nodePool{}, Sharders: &nodePool{}}
        dkgs := make(map[string]*DKG)
        err := c.readYaml(fmt.Sprintf("/0chain/config/%v.yaml",*magicBlockConfig))
        if err == nil {
            setupMagicBlock(mb, c)
            setupNodes(mb, c)
            createMPKS(mb, dkgs)
            createShareOrSigns(mb, dkgs, c.Message)

            mb.Hash = mb.GetHash()
            file, _ := json.MarshalIndent(mb, "", " ")
            err := ioutil.WriteFile(fmt.Sprintf("/0chain/config/%v.json", c.MagicBlockFilename), file, 0644)
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

func setupMagicBlock(mb *magicBlock, c *configYaml) {
    mb.Miners.Type = 0
    mb.Sharders.Type = 1
    mb.Miners.Nodes = make(map[string]node)
    mb.Sharders.Nodes = make(map[string]node)

    mb.MagicBlockNumber = c.MagicBlockNumber
    mb.StartingRound = c.StartingRound
    mb.N = len(c.Miners)
    mb.T = int(float64(mb.N) * (float64(c.TPercent) / 100.0))
    mb.K = int(float64(mb.N) * (float64(c.KPercent) / 100.0))
}

func setupNodes(mb *magicBlock, c *configYaml) {
    mb.Miners.Nodes = make(map[string]node)
    mb.Sharders.Nodes = make(map[string]node)
    for _, v := range c.Miners {
        v.CreationDate = time.Now().Unix()
        v.Type = mb.Miners.Type
        mb.Miners.Nodes[v.ID] = v
    }
    for _, v := range c.Sharders {
        v.CreationDate = time.Now().Unix()
        v.Type = mb.Sharders.Type
        mb.Sharders.Nodes[v.ID] = v
    }
}

func createMPKS(mb *magicBlock, dkgs map[string]*DKG) {
    mb.Mpks = NewMpks()
    for id := range mb.Miners.Nodes {
        dkgs[id] = MakeDKG(mb.T, mb.N, id)
        mpk := &MPK{ID: id}
        for _, v := range dkgs[id].Mpk {
            mpk.Mpk = append(mpk.Mpk, v.SerializeToHexStr())
        }
        mb.Mpks.Mpks[id] = mpk
    }
}

func createShareOrSigns(mb *magicBlock, dkgs map[string]*DKG, message string) {
    mb.ShareOrSigns = NewGroupSharesOrSigns()
    for mid, n := range mb.Miners.Nodes {
        ss := NewShareOrSigns()
        ss.ID = mid
        var privateKey Key
        privateKey.SetHexString(n.PrivateKey)
        for id := range mb.Miners.Nodes {
            otherPartyId := ComputeIDdkg(id)
            share, _ := dkgs[mid].ComputeDKGKeyShare(otherPartyId)
            ss.ShareOrSigns[id] = &DKGKeyShare{Message: message, Share: share.GetHexString(), Sign: privateKey.Sign(message).SerializeToHexStr()}
        }
        mb.ShareOrSigns.Shares[mid] = ss
    }
}

type node struct {
    ID           string `yaml:"id" json:"id"`
    Version      string `yaml:"version" json:"version"`
    CreationDate int64  `json:"creation_date"`
    PublicKey    string `yaml:"public_key" json:"public_key"`
    PrivateKey   string `yaml:"private_key" json:"-"`
    N2NHost      string `yaml:"n2n_ip" json:"n2n_host"`
    Host         string `yaml:"public_ip" json:"host"`
    Port         int    `yaml:"port" json:"port"`
    Type         int    `json:"type"`
    Description  string `yaml:"description" json:"description"`
    SetIndex     int    `yaml:"set_index" json:"set_index"`
    Status       int    `json:"status"`
    Info         info   `json:"info"`
}

type info struct {
    BuildTag string `json:"build_tag"`
}

type configYaml struct {
    Miners             []node `yaml:"miners"`
    Sharders           []node `yaml:"sharders"`
    Message            string `yaml:"message"`
    MagicBlockNumber   int64  `yaml:"magic_block_number"`
    StartingRound      int64  `yaml:"starting_round"`
    TPercent           int    `yaml:"t_percent"`
    KPercent           int    `yaml:"k_percent"`
    MagicBlockFilename string `yaml:"magic_block_filename"`
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

type nodePool struct {
    Type  int             `json:"type"`
    Nodes map[string]node `json:"nodes"`
}

func (np *nodePool) Keys() []string {
    var keys []string
    for k := range np.Nodes {
        keys = append(keys, k)
    }
    return keys
}

type magicBlock struct {
    Hash                   string              `json:"hash"`
    PreviousMagicBlockHash string              `json:"previous_hash"`
    MagicBlockNumber       int64               `json:"magic_block_number"`
    StartingRound          int64               `json:"starting_round"`
    Miners                 *nodePool           `json:"miners"`   //this is the pool of miners participating in the blockchain
    Sharders               *nodePool           `json:"sharders"` //this is the pool of sharders participaing in the blockchain
    ShareOrSigns           *GroupSharesOrSigns `json:"share_or_signs"`
    Mpks                   *Mpks               `json:"mpks"`
    T                      int                 `json:"t"`
    K                      int                 `json:"k"`
    N                      int                 `json:"n"`
}

func (mb *magicBlock) GetHash() string {
    data := []byte(strconv.FormatInt(mb.MagicBlockNumber, 10))
    data = append(data, []byte(mb.PreviousMagicBlockHash)...)
    data = append(data, []byte(strconv.FormatInt(mb.StartingRound, 10))...)
    var minerKeys, sharderKeys, mpkKeys []string
    // miner info
    minerKeys = mb.Miners.Keys()
    sort.Strings(minerKeys)
    for _, v := range minerKeys {
        data = append(data, []byte(v)...)
    }
    // sharder info
    sharderKeys = mb.Sharders.Keys()
    sort.Strings(sharderKeys)
    for _, v := range sharderKeys {
        data = append(data, []byte(v)...)
    }
    // share info
    shareBytes, _ := hex.DecodeString(mb.ShareOrSigns.GetHash())
    data = append(data, shareBytes...)
    // mpk info
    for k := range mb.Mpks.Mpks {
        mpkKeys = append(mpkKeys, k)
    }
    sort.Strings(mpkKeys)
    for _, v := range sharderKeys {
        data = append(data, []byte(v)...)
    }
    data = append(data, []byte(strconv.Itoa(mb.T))...)
    data = append(data, []byte(strconv.Itoa(mb.N))...)
    // return hex.EncodeToString(data)
    return hex.EncodeToString(RawHash(data))
}

type GroupSharesOrSigns struct {
    Shares map[string]*ShareOrSigns `json:"shares"`
}

func NewGroupSharesOrSigns() *GroupSharesOrSigns {
    return &GroupSharesOrSigns{Shares: make(map[string]*ShareOrSigns)}
}

func (gsos *GroupSharesOrSigns) GetHash() string {
    var data []byte
    var keys []string
    for k := range gsos.Shares {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    for _, k := range keys {
        bytes, _ := hex.DecodeString(gsos.Shares[k].Hash())
        data = append(data, bytes...)
    }
    return hex.EncodeToString(RawHash(data))
}

type ShareOrSigns struct {
    ID           string                  `json:"id"`
    ShareOrSigns map[string]*DKGKeyShare `json:"share_or_sign"`
}

func NewShareOrSigns() *ShareOrSigns {
    return &ShareOrSigns{ShareOrSigns: make(map[string]*DKGKeyShare)}
}

func (sos *ShareOrSigns) Hash() string {
    data := sos.ID
    var keys []string
    for k := range sos.ShareOrSigns {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    for _, k := range keys {
        data += string(sos.ShareOrSigns[k].Encode())
    }
    return hex.EncodeToString(RawHash(data))
}

type Mpks struct {
    Mpks map[string]*MPK
}

func NewMpks() *Mpks {
    return &Mpks{Mpks: make(map[string]*MPK)}
}

type MPK struct {
    ID  string
    Mpk []string
}

type DKGKeyShare struct {
    ID      string `json:"id"`
    Message string `json:"message"`
    Share   string `json:"share"`
    Sign    string `json:"sign"`
}

func (dkgs *DKGKeyShare) Encode() []byte {
    buff, _ := json.Marshal(dkgs)
    return buff
}

type DKG struct {
    T int
    N int

    Msk []Key

    Mpk []PublicKey
}

func init() {
    err := bls.Init(bls.CurveFp254BNb)
    if err != nil {
        panic(fmt.Errorf("bls initialization error: %v", err))
    }
}

func MakeDKG(t, n int, id string) *DKG {
    dkg := &DKG{T: t, N: n}
    var secKey Key
    secKey.SetByCSPRNG()

    dkg.Msk = secKey.GetMasterSecretKey(t)
    dkg.Mpk = bls.GetMasterPublicKey(dkg.Msk)
    return dkg
}

func ComputeIDdkg(minerID string) PartyID {
    var forID PartyID
    if err := forID.SetHexString("1" + minerID[:31]); err != nil {
        fmt.Printf("Error while computing ID %s\n", forID.GetHexString())
    }
    return forID
}

func (dkg *DKG) ComputeDKGKeyShare(forID PartyID) (Key, error) {
    var secVec Key
    err := secVec.Set(dkg.Msk, &forID)
    if err != nil {
        return Key{}, err
    }
    return secVec, nil
}

func RawHash(data interface{}) []byte {
    var databuf []byte
    switch dataImpl := data.(type) {
    case []byte:
        databuf = dataImpl
    case HashBytes:
        databuf = dataImpl[:]
    case string:
        databuf = []byte(dataImpl)
    default:
        panic("unknown type")
    }
    hash := sha3.New256()
    hash.Write(databuf)
    var buf []byte
    return hash.Sum(buf)
}

type PublicKey = bls.PublicKey

type Key = bls.SecretKey

type Sign = bls.Sign

type PartyID = bls.ID

type HashBytes [32]byte
