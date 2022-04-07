package interestpoolsc

import (
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
)

//msgp:ignore InterestPool
//go:generate msgp -io=false -tests=false -unexported=true -v

type InterestPool struct {
	*tokenpool.ZcnLockingPool `json:"pool"`
	APR                       float64       `json:"apr"`
	TokensEarned              state.Balance `json:"tokens_earned"`
}

func newInterestPool() *InterestPool {
	return &InterestPool{ZcnLockingPool: &tokenpool.ZcnLockingPool{TokenLockInterface: &TokenLock{}}}
}

func (ip *InterestPool) encode() []byte {
	buff, _ := json.Marshal(ip)
	return buff
}

func (ip *InterestPool) decode(input []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	ir, ok := objMap["apr"]
	if ok {
		var rate float64
		err = json.Unmarshal(*ir, &rate)
		if err != nil {
			return err
		}
		ip.APR = rate
	}
	ie, ok := objMap["tokens_earned"]
	if ok {
		var earned state.Balance
		err = json.Unmarshal(*ie, &earned)
		if err != nil {
			return err
		}
		ip.TokensEarned = earned
	}
	p, ok := objMap["pool"]
	if ok {
		err = ip.ZcnLockingPool.Decode(*p, &TokenLock{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ie *InterestPool) MarshalMsg(o []byte) ([]byte, error) {
	d := interestPoolDecode(*ie)

	return d.MarshalMsg(o)
}

func (ie *InterestPool) UnmarshalMsg(b []byte) ([]byte, error) {
	d := interestPoolDecode{ZcnLockingPool: &tokenpool.ZcnLockingPool{TokenLockInterface: &TokenLock{}}}
	o, err := d.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	*ie = InterestPool(d)
	return o, nil
}

type interestPoolDecode InterestPool
