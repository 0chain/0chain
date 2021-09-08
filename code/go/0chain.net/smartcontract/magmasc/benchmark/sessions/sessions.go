package sessions

import (
	"errors"
	"net/url"
	"strconv"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/util"
	"0chain.net/smartcontract/magmasc"
)

func CountActive(sc *magmasc.MagmaSmartContract, sci chain.StateContextI) (int, error) {
	nas, handler := 0, sc.RestHandlers["/acknowledgmentAccepted"]
	for i := 0; ; i++ {
		val := url.Values{}
		val.Set("id", GetSessionName(i, true))
		output, err := handler(nil, val, sci)
		if err != nil && errors.Is(err, util.ErrValueNotPresent) {
			break
		} else if err != nil {
			return nas, err
		}
		if output.(*zmc.Acknowledgment).Billing.CompletedAt == 0 {
			nas++
		}
	}

	return nas, nil
}

func CountInactive(sc *magmasc.MagmaSmartContract, sci chain.StateContextI) (int, error) {
	nis, handler := 0, sc.RestHandlers["/acknowledgmentAccepted"]
	for i := 0; ; i++ {
		val := url.Values{}
		val.Set("id", GetSessionName(i, false))
		output, err := handler(nil, val, sci)
		if err != nil && errors.Is(err, util.ErrValueNotPresent) {
			break
		} else if err != nil {
			return nis, err
		}
		if output.(*zmc.Acknowledgment).Billing.CompletedAt != 0 {
			nis++
		}
	}

	return nis, nil
}

const (
	sessionActPrefix   = "act_"
	sessionInactPrefix = "inact_"
	sessionName        = "session_"
)

func GetSessionName(num int, active bool) string {
	prefix := ""
	if active {
		prefix = sessionActPrefix
	} else {
		prefix = sessionInactPrefix
	}

	return prefix + sessionName + strconv.Itoa(num)
}
