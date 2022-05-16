package interestpoolsc

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"0chain.net/rest/restinterface"

	"0chain.net/smartcontract"

	"0chain.net/core/common"
)

type InterestPoolRestHandler struct {
	restinterface.RestHandlerI
}

func NewInterestPoolRestHandler(rh restinterface.RestHandlerI) *InterestPoolRestHandler {
	return &InterestPoolRestHandler{rh}
}

func SetupRestHandler(rh restinterface.RestHandlerI) {
	frh := NewInterestPoolRestHandler(rh)
	miner := "/v1/screst/" + ADDRESS
	http.HandleFunc(miner+"/getPoolsStats", frh.getPoolsStats)
	http.HandleFunc(miner+"/getLockConfig", frh.getLockConfig)
	http.HandleFunc(miner+"/getConfig", frh.getConfig)
}

func GetRestNames() []string {
	return []string{
		"/getPoolsStats",
		"/getLockConfig",
		"/getConfig",
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getConfig getConfig
// get interest pool configuration settings
//
// responses:
//  200: StringMap
//  500:
func (irh *InterestPoolRestHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	gn, err := getGlobalNode(irh.GetStateContext())
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get global node", err.Error()))
		return
	}

	fields := map[string]string{
		Settings[MinLock]:       fmt.Sprintf("%0v", gn.MinLock),
		Settings[MaxMint]:       fmt.Sprintf("%0v", gn.MaxMint),
		Settings[MinLockPeriod]: fmt.Sprintf("%0v", gn.MinLockPeriod),
		Settings[Apr]:           fmt.Sprintf("%0v", gn.APR),
		Settings[OwnerId]:       fmt.Sprintf("%v", gn.OwnerId),
	}

	for _, key := range costFunctions {
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
	gn, err := getGlobalNode(irh.GetStateContext())
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
//  200: poolStat
//  400:
//  500:
func (irh *InterestPoolRestHandler) getPoolsStats(w http.ResponseWriter, r *http.Request) {
	var un = new(UserNode)
	err := irh.GetStateContext().GetTrieNode(un.getKey(ADDRESS), un)
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

func getPoolStats(pool *interestPool, t time.Time) (*poolStat, error) {
	stat := &poolStat{}
	statBytes := pool.LockStats(t)
	err := stat.decode(statBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	stat.ID = pool.ID
	stat.Locked = pool.IsLocked(t)
	stat.Balance = pool.Balance
	stat.APR = pool.APR
	stat.TokensEarned = pool.TokensEarned
	return stat, nil
}
