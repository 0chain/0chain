package vestingsc

import (
	"net/http"

	"0chain.net/core/common"
	"0chain.net/rest/restinterface"
	"0chain.net/smartcontract"
)

type VestingRestHandler struct {
	restinterface.RestHandlerI
}

func NewVestingRestHandler(rh restinterface.RestHandlerI) *VestingRestHandler {
	return &VestingRestHandler{rh}
}

func SetupRestHandler(rh restinterface.RestHandlerI) {
	vrh := NewVestingRestHandler(rh)
	miner := "/v1/screst/" + ADDRESS
	http.HandleFunc(miner+"/getPoolInfo", vrh.getPoolInfo)
	http.HandleFunc(miner+"/getClientPools", vrh.getClientPools)
	http.HandleFunc(miner+"/getConfig", vrh.getConfig)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getClientPools getClientPools
// get client pools
//
// responses:
//  200: vestingClientPools
//  500:
func (vrh *VestingRestHandler) getClientPools(w http.ResponseWriter, r *http.Request) {

	var (
		clientID = r.URL.Query().Get("client_id")
		cp       *clientPools
	)

	// just return empty list if not found
	if err := vrh.GetTrieNode(clientPoolsKey(ADDRESS, clientID), cp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get or create client pools"))
		return
	}

	common.Respond(w, r, cp, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPoolInfo getPoolInfo
// get vesting configuration settings
//
// responses:
//  200: vestingInfo
//  500:
func (vrh *VestingRestHandler) getPoolInfo(w http.ResponseWriter, r *http.Request) {
	var (
		poolID = r.URL.Query().Get("pool_id")
		vp     = new(vestingPool)
	)

	if err := vrh.GetTrieNode(poolID, vp); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get pool"))
		return
	}

	common.Respond(w, r, vp.info(common.Now()), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getConfig getConfig
// get vesting configuration settings
//
// responses:
//  200: StringMap
//  500:
func (vrh *VestingRestHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	var conf = new(config)
	err := vrh.GetTrieNode(scConfigKey(ADDRESS), conf)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get config", err.Error()))
		return
	}

	common.Respond(w, r, conf.getConfigMap(), nil)
}
