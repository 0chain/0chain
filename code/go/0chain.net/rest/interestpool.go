package rest

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"0chain.net/smartcontract"

	"0chain.net/core/common"

	"0chain.net/smartcontract/interestpoolsc"
)

type InterestPoolRestHandler struct {
	*RestHandler
}

func NewInterestPoolRestHandler(rh *RestHandler) *InterestPoolRestHandler {
	return &InterestPoolRestHandler{rh}
}

func SetupInterestPoolRestHandler(rh *RestHandler) {
	frh := NewInterestPoolRestHandler(rh)
	miner := "/v1/screst/" + interestpoolsc.ADDRESS
	http.HandleFunc(miner+"/getPoolsStats", frh.getPoolsStats)
	http.HandleFunc(miner+"/getLockConfig", frh.getLockConfig)
	http.HandleFunc(miner+"/getConfig", frh.getConfig)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getConfig getConfig
// get interest pool configuration settings
//
// responses:
//  200: StringMap
//  500:
func (irh *InterestPoolRestHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	gn, err := interestpoolsc.GetGlobalNode(irh)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get global node", err.Error()))
		return
	}

	fields := map[string]string{
		interestpoolsc.Settings[interestpoolsc.MinLock]:       fmt.Sprintf("%0v", gn.MinLock),
		interestpoolsc.Settings[interestpoolsc.MaxMint]:       fmt.Sprintf("%0v", gn.MaxMint),
		interestpoolsc.Settings[interestpoolsc.MinLockPeriod]: fmt.Sprintf("%0v", gn.MinLockPeriod),
		interestpoolsc.Settings[interestpoolsc.Apr]:           fmt.Sprintf("%0v", gn.APR),
		interestpoolsc.Settings[interestpoolsc.OwnerId]:       fmt.Sprintf("%v", gn.OwnerId),
	}

	for _, key := range interestpoolsc.CostFunctions {
		fields[fmt.Sprintf("cost.%s", key)] = fmt.Sprintf("%0v", gn.Cost[strings.ToLower(key)])
	}

	common.Respond(w, r, &smartcontract.StringMap{
		Fields: fields,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getLockConfig getLockConfig
// get lock configuration
//
// responses:
//  200: InterestPoolGlobalNode
//  500:
func (irh *InterestPoolRestHandler) getLockConfig(w http.ResponseWriter, r *http.Request) {
	gn, err := interestpoolsc.GetGlobalNode(irh)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get global node", err.Error()))
		return
	}
	common.Respond(w, r, gn, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPoolsStats getPoolsStats
// get pool stats
//
// responses:
//  200: poolStats
//  400:
//  500:
func (irh *InterestPoolRestHandler) getPoolsStats(w http.ResponseWriter, r *http.Request) {
	var un = new(interestpoolsc.UserNode)
	err := irh.GetTrieNode(un.GetKey(interestpoolsc.ADDRESS), un)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't user node", err.Error()))
		return
	}

	if len(un.Pools) == 0 {
		common.Respond(w, r, nil, common.NewErrNoResource("can't find user node"))
		return
	}

	t := time.Now()
	stats := &poolStats{}
	for _, pool := range un.Pools {
		stat, err := getPoolStats(pool, t)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal("can't get pool stats", err.Error()))
			return
		}
		stats.addStat(stat)
	}
	common.Respond(w, r, stats, nil)
}
