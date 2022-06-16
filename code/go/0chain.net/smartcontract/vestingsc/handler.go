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

func GetEndpoints(rh rest.RestHandlerI) []rest.Endpoint {
	vrh := NewVestingRestHandler(rh)
	vesting := "/v1/screst/" + ADDRESS
	return []rest.Endpoint{
		rest.MakeEndpoint(vesting+"/pool-info", vrh.getPoolInfo),
		rest.MakeEndpoint(vesting+"/client-pools", vrh.getClientPools),
		rest.MakeEndpoint(vesting+"/vesting-config", vrh.getConfig),
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/client-pools getClientPools
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

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/pool-info getPoolInfo
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

	vpInfo, err := vp.info(common.Now())
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true,
			"can't get vesting pool info"))
		return
	}
	common.Respond(w, r, vpInfo, nil)
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
