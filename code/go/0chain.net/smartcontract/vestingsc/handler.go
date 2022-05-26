package vestingsc

import (
	"0chain.net/smartcontract/rest"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/smartcontract"
)

type VestingRestHandler struct {
	rest.RestHandlerI
}

func NewVestingRestHandler(rh rest.RestHandlerI) *VestingRestHandler {
	return &VestingRestHandler{rh}
}

func SetupRestHandler(rh rest.RestHandlerI) {
	rh.Register(GetEndpoints(rh))
}

func GetEndpoints(rh rest.RestHandlerI) []rest.RestEndpoint {
	vrh := NewVestingRestHandler(rh)
	vesting := "/v1/screst/" + ADDRESS
	return []rest.RestEndpoint{
		{Name: vesting + "/getPoolInfo", Endpoint: vrh.getPoolInfo},
		{Name: vesting + "/getClientPools", Endpoint: vrh.getClientPools},
		{Name: vesting + "/getConfig", Endpoint: vrh.getConfig},
	}
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
		err      error
	)

	// just return empty list if not found
	if cp, err = getOrCreateClientPools(clientID, vrh.GetQueryStateContext()); err != nil {
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
		vp     *vestingPool
		err    error
	)

	if vp, err = getPool(poolID, vrh.GetQueryStateContext()); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get pool"))
		return
	}

	common.Respond(w, r, vp.info(common.Now()), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/vesting_config vesting_config
// get vesting configuration settings
//
// responses:
//  200: StringMap
//  500:
func (vrh *VestingRestHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	conf, err := getConfigReadOnly(vrh.GetQueryStateContext())
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get config", err.Error()))
		return
	}
	common.Respond(w, r, conf.getConfigMap(), nil)
}
