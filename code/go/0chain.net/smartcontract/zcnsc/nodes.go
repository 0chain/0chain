package zcnsc

import (
	"encoding/json"
	"fmt"
	"strconv"

	"0chain.net/chaincore/tokenpool"
	"0chain.net/smartcontract/dbs/event"
	"gorm.io/gorm"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract"
)

// ------------- GlobalNode ------------------------

type GlobalNode struct {
	ID     string      `json:"id"`
	Config *ZCNSConfig `json:"config"`
}

func (gn *GlobalNode) UpdateConfig(cfg *smartcontract.StringMap) error {
	var (
		err error
		c   *ZCNSConfig
	)

	if gn.Config == nil {
		gn.Config = new(ZCNSConfig)
	}

	c = gn.Config

	for key, value := range cfg.Fields {
		switch key {
		case MinMintAmount:
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to state.Balance", key, value)
			}
			c.MinMintAmount = state.Balance(amount * 1e10)
		case MinBurnAmount:
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to state.Balance", key, value)
			}
			c.MinBurnAmount = state.Balance(amount * 1e10)
		case BurnAddress:
			if value == "" {
				return fmt.Errorf("key %s is empty", key)
			}
			c.BurnAddress = value
		case PercentAuthorizers:
			c.PercentAuthorizers, err = strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to float64", key, value)
			}
		case MinAuthorizers:
			c.MinAuthorizers, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to int64", key, value)
			}
		case MinStakeAmount:
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to state.Balance", key, value)
			}
			c.MinStakeAmount = state.Balance(amount * 1e10)
		case MaxFee:
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("key %s, unable to convert %v to state.Balance", key, value)
			}
			c.MaxFee = state.Balance(amount * 1e10)
		case OwnerID:
			c.OwnerId = value
		default:
			return fmt.Errorf("key %s not recognised as setting", key)
		}
	}

	return nil
}

func (gn *GlobalNode) Validate() error {
	const (
		Code = "failed to validate global node"
	)

	switch {
	case gn.Config.MinStakeAmount < 1:
		return common.NewError(Code, fmt.Sprintf("min stake amount (%v) is less than 1", gn.Config.MinStakeAmount))
	case gn.Config.MinMintAmount < 1:
		return common.NewError(Code, fmt.Sprintf("min mint amount (%v) is less than 1", gn.Config.MinMintAmount))
	case gn.Config.MaxFee < 1:
		return common.NewError(Code, fmt.Sprintf("max fee (%v) is less than 1", gn.Config.MaxFee))
	case gn.Config.MinAuthorizers < 20:
		return common.NewError(Code, fmt.Sprintf("min quantity of authorizers (%v) is less than 20", gn.Config.MinAuthorizers))
	case gn.Config.MinBurnAmount < 1:
		return common.NewError(Code, fmt.Sprintf("min burn amount (%v) is less than 1", gn.Config.MinBurnAmount))
	case gn.Config.PercentAuthorizers < 70:
		return common.NewError(Code, fmt.Sprintf("min percentage of authorizers (%v) is less than 70", gn.Config.PercentAuthorizers))
	case gn.Config.BurnAddress == "":
		return common.NewError(Code, fmt.Sprintf("burn address (%v) is not valid", gn.Config.BurnAddress))
	case gn.Config.OwnerId == "":
		return common.NewError(Code, fmt.Sprintf("owner id (%v) is not valid", gn.Config.OwnerId))
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

// ----- AuthorizerNode --------------------

type AuthorizerNode struct {
	ID        string                    `json:"id"`
	PublicKey string                    `json:"public_key"`
	Staking   *tokenpool.ZcnLockingPool `json:"staking"`
	URL       string                    `json:"url"`
	Config    *AuthorizerConfig         `json:"config"`
}

// NewAuthorizer To review: tokenLock init values
// PK = authorizer node public key
// ID = authorizer node public id = Client ID
func NewAuthorizer(ID string, PK string, URL string) *AuthorizerNode {
	return &AuthorizerNode{
		ID:        ID,
		PublicKey: PK,
		URL:       URL,
		Staking: &tokenpool.ZcnLockingPool{
			ZcnPool: tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{
					ID:      "", // must be filled when DigPool is invoked. Usually this is a trx.Hash
					Balance: 0,  // filled when we dig pool
				},
			},
			TokenLockInterface: TokenLock{
				StartTime: 0,
				Duration:  0,
				Owner:     ID,
			},
		},
		Config: &AuthorizerConfig{
			Fee: 0,
		},
	}
}

func (an *AuthorizerNode) UpdateConfig(cfg *AuthorizerConfig) {
	if an.Config == nil {
		an.Config = new(AuthorizerConfig)
	}

	an.Config.Fee = cfg.Fee
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

	if an.Staking == nil {
		an.Staking = &tokenpool.ZcnLockingPool{
			ZcnPool: tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{},
			},
		}
	}

	staking, ok := objMap["staking"]
	if ok {
		tokenlock := &TokenLock{}
		err = an.Staking.Decode(*staking, tokenlock)
		if err != nil {
			return err
		}
	}

	rawCfg, ok := objMap["config"]
	if ok {
		var cfg = &AuthorizerConfig{}
		err = json.Unmarshal(*rawCfg, cfg)
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

func (an *AuthorizerNode) ToEvent() ([]byte, error) {
	if an.Config == nil {
		an.Config = new(AuthorizerConfig)
	}
	data, err := json.Marshal(&event.Authorizer{
		Model:           gorm.Model{},
		Fee:             an.Config.Fee,
		AuthorizerID:    an.ID,
		URL:             an.URL,
		Latitude:        0,
		Longitude:       0,
		LastHealthCheck: 0,
		DelegateWallet:  "",
		MinStake:        0,
		MaxStake:        0,
		NumDelegates:    0,
		ServiceCharge:   0,
	})
	if err != nil {
		return nil, fmt.Errorf("marshalling authorizer event: %v", err)
	}

	return data, nil
}

func AuthorizerFromEvent(buf []byte) (*AuthorizerNode, error) {
	ev := &event.Authorizer{}
	err := json.Unmarshal(buf, ev)
	if err != nil {
		return nil, err
	}

	return &AuthorizerNode{
		ID:        ev.AuthorizerID,
		URL:       ev.URL,
		PublicKey: "",  // fetch this from MPT
		Staking:   nil, // fetch this from MPT
	}, nil
}

// ----- UserNode ------------------

type UserNode struct {
	ID    string `json:"id"`
	Nonce int64  `json:"nonce"`
}

func NewUserNode(id string, nonce int64) *UserNode {
	return &UserNode{
		ID:    id,
		Nonce: nonce,
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
