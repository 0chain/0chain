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

func Count(sc *magmasc.MagmaSmartContract, sci chain.StateContextI) (act, inact int, err error) {
	var (
		handl = sc.RestHandlers["/acknowledgmentAccepted"]
	)

	for i := 0; ; i++ {
		val := url.Values{}
		val.Set("id", GetSessionName(i, true))
		output, err := handl(nil, val, sci)
		if errors.Is(err, util.ErrValueNotPresent) {
			break
		} else if err != nil {
			return 0, 0, err
		}

		outputAckn := output.(*zmc.Acknowledgment)
		if outputAckn.Billing.CompletedAt == 0 {
			act++
		} else {
			inact++
		}
	}

	for i := 0; ; i++ {
		val := url.Values{}
		val.Set("id", GetSessionName(i, false))
		output, err := handl(nil, val, sci)
		if errors.Is(err, util.ErrValueNotPresent) {
			break
		}
		if err != nil {
			return 0, 0, err
		}

		outputAckn := output.(*zmc.Acknowledgment)
		if outputAckn.Billing.CompletedAt == 0 {
			act++
		} else {
			inact++
		}
	}

	return act, inact, nil
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
