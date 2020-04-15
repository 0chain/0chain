package vestingsc

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/util"
)

const logPartLength = 100

func lastPartKey(vscKey string) string {
	return vscKey + ":lastpart"
}

func partKey(vscKey string, part int64) string {
	return vscKey + ":logpart:" + strconv.FormatInt(part, 10)
}

type lastPart struct {
	Part int64 `json:"part"`
}

func (lp *lastPart) Encode() (p []byte) {
	var err error
	if p, err = json.Marshal(lp); err != nil {
		panic(err) // must not happen
	}
	return
}

func (lp *lastPart) Decode(p []byte) error {
	return json.Unmarshal(p, lp)
}

type logPart struct {
	Part int64    `json:"part"`
	Txns []string `json:"txns"`
}

func (lp *logPart) Encode() (p []byte) {
	var err error
	if p, err = json.Marshal(lp); err != nil {
		panic(err) // must not happen
	}
	return
}

func (lp *logPart) Decode(p []byte) error {
	return json.Unmarshal(p, lp)
}

func (vsc *VestingSmartContract) getLastPart(
	balances chainstate.StateContextI) (part int64, err error) {

	var partSeri util.Serializable
	partSeri, err = balances.GetTrieNode(lastPartKey(vsc.ID))
	if err != nil {
		if err == util.ErrValueNotPresent {
			return 0, nil
		}
		return
	}

	var lp lastPart
	if err = lp.Decode(partSeri.Encode()); err != nil {
		return
	}

	return lp.Part, nil
}

func (vsc *VestingSmartContract) setLastPart(part int64,
	balances chainstate.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(lastPartKey(vsc.ID), &lastPart{Part: part})
	return
}

func (vsc *VestingSmartContract) getLogPart(part int64,
	balances chainstate.StateContextI) (lp *logPart, err error) {

	var partSeri util.Serializable
	partSeri, err = balances.GetTrieNode(partKey(vsc.ID, part))
	if err != nil {
		if err == util.ErrValueNotPresent && part == 0 {
			return new(logPart), nil
		}
		return
	}

	lp = new(logPart)
	if err = lp.Decode(partSeri.Encode()); err != nil {
		return nil, err
	}
	return
}

func (vsc *VestingSmartContract) setLogPart(lp *logPart,
	balances chainstate.StateContextI) (err error) {

	_, err = balances.InsertTrieNode(partKey(vsc.ID, lp.Part), lp)
	return
}

func (vsc *VestingSmartContract) addTxnToVestingLog(txHash string,
	balances chainstate.StateContextI) (err error) {

	var part int64
	if part, err = vsc.getLastPart(balances); err != nil {
		return
	}

	var lp *logPart
	if lp, err = vsc.getLogPart(part, balances); err != nil {
		return
	}

	// got the limit, create new part
	if len(lp.Txns) == logPartLength {
		part++
		lp = new(logPart)
		lp.Part = part
		if err = vsc.setLastPart(part, balances); err != nil {
			return
		}
	}

	lp.Txns = append(lp.Txns, txHash)

	return vsc.setLogPart(lp, balances)
}

func (vsc *VestingSmartContract) getLastPartHandler(_ context.Context,
	_ url.Values, balances chainstate.StateContextI) (
	interface{}, error) {

	return vsc.getLastPart(balances)
}

func (vsc *VestingSmartContract) getPartHandler(ctx context.Context,
	params url.Values, balances chainstate.StateContextI) (
	resp interface{}, err error) {

	var (
		partNo = params.Get("part")
		part   int64
	)

	if part, err = strconv.ParseInt(partNo, 10, 64); err != nil {
		return
	}

	return vsc.getLogPart(part, balances)
}
