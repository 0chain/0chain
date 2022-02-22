package interestpoolsc

import (
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
)

//msgp:ignore interestPool
//go:generate msgp -io=false -tests=false -unexported=true -v

type interestPool struct {
	*tokenpool.ZcnLockingPool `json:"pool"`
	APR                       float64       `json:"apr"`
	TokensEarned              state.Balance `json:"tokens_earned"`
}

func newInterestPool() *interestPool {
	return &interestPool{ZcnLockingPool: &tokenpool.ZcnLockingPool{TokenLockInterface: &TokenLock{}}}
}

func (ip *interestPool) encode() []byte {
	buff, _ := json.Marshal(ip)
	return buff
}

func (ip *interestPool) decode(input []byte) error {
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

type interestPoolDecode interestPool

func (ip *interestPool) MarshalMsg(o []byte) ([]byte, error) {
	ipd := interestPoolDecode(*ip)
	return ipd.MarshalMsg(o)
}

func (ip *interestPool) UnmarshalMsg(data []byte) ([]byte, error) {
	nip := newInterestPool()
	nipd := interestPoolDecode(*nip)
	d, err := nipd.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}

	*ip = interestPool(nipd)
	return d, nil
}

func (ip *interestPool) Msgsize() int {
	ipd := interestPoolDecode(*ip)
	return ipd.Msgsize()
}
