package zcnsc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"gorm.io/gorm"
)

//msgp:ignore AuthorizerNode
//go:generate msgp -io=false -tests=false -unexported -v

// ----- AuthorizerNode --------------------

type AuthorizerNode struct {
	ID        string                    `json:"id"`
	PublicKey string                    `json:"public_key"`
	Staking   *tokenpool.ZcnLockingPool `json:"staking"`
	URL       string                    `json:"url"`
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
		err = an.Staking.Decode(*staking, &TokenLock{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (an *AuthorizerNode) MarshalMsg(o []byte) ([]byte, error) {
	d := authorizerNodeDecode(*an)
	return d.MarshalMsg(o)
}

func (an *AuthorizerNode) UnmarshalMsg(data []byte) ([]byte, error) {
	d := authorizerNodeDecode{Staking: &tokenpool.ZcnLockingPool{TokenLockInterface: &TokenLock{}}}
	o, err := d.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}

	*an = AuthorizerNode(d)
	return o, nil
}

func (an *AuthorizerNode) Save(ctx cstate.StateContextI) (err error) {
	_, err = ctx.InsertTrieNode(an.GetKey(), an)
	if err != nil {
		return common.NewError("save_auth_node_failed", "saving authorizer node: "+err.Error())
	}
	return nil
}

func (an *AuthorizerNode) ToEvent() ([]byte, error) {
	data, err := json.Marshal(&event.Authorizer{
		Model:           gorm.Model{},
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

// CreateAuthorizer To review: tokenLock init values
// pk = authorizer node public key
// authId = authorizer node public id = Client ID
func CreateAuthorizer(authId string, pk string, url string) *AuthorizerNode {
	return &AuthorizerNode{
		ID:        authId,
		PublicKey: pk,
		URL:       url,
		Staking: &tokenpool.ZcnLockingPool{
			ZcnPool: tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{
					ID:      "", // must be filled when DigPool is invoked. Usually this is a trx.Hash
					Balance: 0,  // filled when we dig pool
				},
			},
			TokenLockInterface: &TokenLock{
				StartTime: 0,
				Duration:  0,
				Owner:     authId,
			},
		},
	}
}

type authorizerNodeDecode AuthorizerNode
