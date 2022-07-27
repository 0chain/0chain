package zcnsc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/dbs/event"
	"gorm.io/gorm"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract"
)

//go:generate msgp -v -io=false -tests=false -unexported

// ------------- GlobalNode ------------------------

type ZCNSConfig struct {
	MinMintAmount      currency.Coin  `json:"min_mint"`
	MinBurnAmount      currency.Coin  `json:"min_burn"`
	MinStakeAmount     currency.Coin  `json:"min_stake"`
	MinLockAmount      currency.Coin  `json:"min_lock"`
	MinAuthorizers     int64          `json:"min_authorizers"`
	PercentAuthorizers float64        `json:"percent_authorizers"`
	MaxFee             currency.Coin  `json:"max_fee"`
	BurnAddress        string         `json:"burn_address"`
	OwnerId            string         `json:"owner_id"`
	Cost               map[string]int `json:"cost"`
	MaxDelegates       int            `json:"max_delegates"` // MaxDelegates per stake pool
}

type GlobalNode struct {
	*ZCNSConfig     `json:"zcnsc_config"`
	ID              string         `json:"id"`
	WZCNNonceMinted map[int64]bool `json:"user_nonce_minted"`
}

func (gn *GlobalNode) UpdateConfig(cfg *smartcontract.StringMap) (err error) {
	for key, value := range cfg.Fields {
		switch key {
		case MinMintAmount:
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to currency.Coin", key, value)
			}
			gn.MinMintAmount, err = currency.ParseZCN(amount)
			if err != nil {
				return err
			}
		case MinBurnAmount:
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to currency.Coin", key, value)
			}
			gn.MinBurnAmount, err = currency.ParseZCN(amount)
			if err != nil {
				return err
			}
		case BurnAddress:
			if value == "" {
				return fmt.Errorf("key %s is empty", key)
			}
			gn.BurnAddress = value
		case PercentAuthorizers:
			gn.PercentAuthorizers, err = strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to float64", key, value)
			}
		case MinAuthorizers:
			gn.MinAuthorizers, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to int64", key, value)
			}
		case MinStakeAmount:
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to currency.Coin", key, value)
			}
			gn.MinStakeAmount, err = currency.ParseZCN(amount)
			if err != nil {
				return err
			}
		case MaxFee:
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to currency.Coin", key, value)
			}
			gn.MaxFee, err = currency.ParseZCN(amount)
			if err != nil {
				return err
			}
		case OwnerID:
			gn.OwnerId = value
		case Cost:
			err = gn.setCostValue(Cost, value)
			if err != nil {
				return err
			}
		case MinLockAmount:
			minLockAmount, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to uint64", key, value)
			}
			gn.MinLockAmount = currency.Coin(minLockAmount)
		case MaxDelegates:
			gn.MaxDelegates, err = strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to int64", key, value)
			}
		default:
			return fmt.Errorf("key %s, unable to convert %v to currency.Coin", key, value)
		}
	}

	return nil
}

func (gn *GlobalNode) setCostValue(key, value string) error {
	if !strings.HasPrefix(key, fmt.Sprintf("%s.", Cost)) {
		return fmt.Errorf("key %s not recognised as setting", key)
	}

	costKey := strings.ToLower(strings.TrimPrefix(key, fmt.Sprintf("%s.", Cost)))
	for _, costFunction := range CostFunctions {
		if costKey != strings.ToLower(costFunction) {
			continue
		}
		costValue, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("key %s, unable to convert %v to integer", key, value)
		}

		if costValue < 0 {
			return fmt.Errorf("cost.%s contains invalid value %s", key, value)
		}

		gn.Cost[costKey] = costValue

		return nil
	}

	return fmt.Errorf("cost config setting %s not found", costKey)
}

func (gn *GlobalNode) Validate() error {
	const (
		Code = "failed to validate global node"
	)
	// todo stop using hard coded values here
	switch {
	case gn.MinStakeAmount < 1:
		return common.NewError(Code, fmt.Sprintf("min stake amount (%v) is less than 1", gn.MinStakeAmount))
	case gn.MinMintAmount < 1:
		return common.NewError(Code, fmt.Sprintf("min mint amount (%v) is less than 1", gn.MinMintAmount))
	case gn.MaxFee < 1:
		return common.NewError(Code, fmt.Sprintf("max fee (%v) is less than 1", gn.MaxFee))
	case gn.MinAuthorizers < 1:
		return common.NewError(Code, fmt.Sprintf("min quantity of authorizers (%v) is less than 1", gn.MinAuthorizers))
	case gn.MinBurnAmount < 1:
		return common.NewError(Code, fmt.Sprintf("min burn amount (%v) is less than 1", gn.MinBurnAmount))
	case gn.PercentAuthorizers < 0:
		return common.NewError(Code, fmt.Sprintf("min percentage of authorizers (%v) is less than 0", gn.PercentAuthorizers))
	case gn.BurnAddress == "":
		return common.NewError(Code, fmt.Sprintf("burn address (%v) is not valid", gn.BurnAddress))
	case gn.OwnerId == "":
		return common.NewError(Code, fmt.Sprintf("owner id (%v) is not valid", gn.OwnerId))
	case gn.MaxDelegates <= 0:
		return common.NewError(Code, fmt.Sprintf("max delegate count (%v) is less than 0", gn.MaxDelegates))
	case gn.MinLockAmount == 0:
		return common.NewError(Code, fmt.Sprintf("min lock amount (%v) is equal to 0", gn.MinLockAmount))
	}
	return nil
}

func (gn *GlobalNode) GetKey() datastore.Key {
	return fmt.Sprintf("%s:%s:%s", ADDRESS, GlobalNodeType, gn.ID)
}

func (gn *GlobalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *GlobalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

func (gn *GlobalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *GlobalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, gn)
	return err
}

func (gn *GlobalNode) Save(balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(gn.GetKey(), gn)
	return
}

// ----- AuthorizerConfig --------------------

type AuthorizerConfig struct {
	Fee currency.Coin `json:"fee"`
}

func (c *AuthorizerConfig) Decode(input []byte) (err error) {
	err = json.Unmarshal(input, c)
	return
}

// ----- AuthorizerNode --------------------

// AuthorizerNode used in `UpdateAuthorizerConfig` functions
type AuthorizerNode struct {
	ID        string            `json:"id"`
	PublicKey string            `json:"public_key"`
	URL       string            `json:"url"`
	Config    *AuthorizerConfig `json:"config"`
}

// NewAuthorizer To review: tokenLock init values
// PK = authorizer node public key
// ID = authorizer node public id = Client ID
func NewAuthorizer(ID string, PK string, URL string) *AuthorizerNode {
	a := &AuthorizerNode{
		ID:        ID,
		PublicKey: PK,
		URL:       URL,
		Config: &AuthorizerConfig{
			Fee: 0,
		},
	}

	return a
}

func (an *AuthorizerNode) UpdateConfig(cfg *AuthorizerConfig) error {
	if cfg == nil {
		return errors.New("config not initialized")
	}

	an.Config = cfg

	return nil
}

func (an *AuthorizerNode) GetKey() string {
	return fmt.Sprintf("%s:%s:%s", ADDRESS, AuthorizerNodeType, an.ID)
}

func (an *AuthorizerNode) Encode() []byte {
	bytes, _ := json.Marshal(an)
	return bytes
}

func (an *AuthorizerNode) Decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}

	id, ok := objMap["id"]
	if ok {
		var idStr *string
		err = json.Unmarshal(*id, &idStr)
		if err != nil {
			return err
		}
		an.ID = *idStr
	}

	pk, ok := objMap["public_key"]
	if ok {
		var pkStr *string
		err = json.Unmarshal(*pk, &pkStr)
		if err != nil {
			return err
		}
		an.PublicKey = *pkStr
	}

	url, ok := objMap["url"]
	if ok {
		var urlStr *string
		err = json.Unmarshal(*url, &urlStr)
		if err != nil {
			return err
		}
		an.URL = *urlStr
	}

	rawCfg, ok := objMap["config"]
	if ok {
		var cfg = &AuthorizerConfig{}
		err = cfg.Decode(*rawCfg)
		if err != nil {
			return err
		}

		an.Config = cfg
	}

	return nil
}

func (an *AuthorizerNode) Save(ctx cstate.StateContextI) (err error) {
	_, err = ctx.InsertTrieNode(an.GetKey(), an)
	if err != nil {
		return common.NewError("save_auth_node_failed", "saving authorizer node: "+err.Error())
	}
	return nil
}

func (an *AuthorizerNode) ToEvent() *event.Authorizer {
	if an.Config == nil {
		an.Config = new(AuthorizerConfig)
	}
	return &event.Authorizer{
		Model:        gorm.Model{},
		Fee:          an.Config.Fee,
		AuthorizerID: an.ID,
		URL:          an.URL,
	}
}

func AuthorizerFromEvent(ev *event.Authorizer) (*AuthorizerNode, error) {

	return NewAuthorizer(ev.AuthorizerID, "", ev.URL), nil
}

// ----- UserNode ------------------

type UserNode struct {
	ID        string `json:"id"`
	BurnNonce int64  `json:"burn_nonce"`
}

func NewUserNode(id string) *UserNode {
	return &UserNode{
		ID: id,
	}
}

func (un *UserNode) GetKey() datastore.Key {
	return fmt.Sprintf("%s:%s:%s", ADDRESS, UserNodeType, un.ID)
}

func (un *UserNode) GetHash() string {
	return util.ToHex(un.GetHashBytes())
}

func (un *UserNode) GetHashBytes() []byte {
	return encryption.RawHash(un.Encode())
}

func (un *UserNode) Encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *UserNode) Decode(input []byte) error {
	err := json.Unmarshal(input, un)
	return err
}

func (un *UserNode) Save(balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(un.GetKey(), un)
	return
}
