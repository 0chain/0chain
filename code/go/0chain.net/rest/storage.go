package rest

import (
	"net/http"

	"0chain.net/core/logging"

	"0chain.net/core/common"
)

type StorageRestHandler struct {
	*RestHandler
}

func NewStorageRestHandler(rh *RestHandler) *StorageRestHandler {
	return &StorageRestHandler{rh}
}

// GetBlobberCount swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_count get_blobber_count
// Get count of blobber
//
// responses:
//  200: intMap
//  400:
func (srh StorageRestHandler) GetBlobberCount(w http.ResponseWriter, r *http.Request) {
	blobberCount, err := srh.GetEventDB().GetBlobberCount()
	if err != nil {
		err := common.NewErrInternal("getting blobber count:" + err.Error())
		common.Respond(w, r, "", err)
		return
	}
	resp := intMap{
		"count": blobberCount,
	}

	common.Respond(w, r, resp, err)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber getBlobber
// Get blobber information
//
// parameters:
//    + name: blobber_id
//      description: blobber for which to return information
//      required: true
//
// responses:
//  200: StorageNode
//  400:
//  500:
func (srh StorageRestHandler) GetBlobber(w http.ResponseWriter, r *http.Request) {
	logging.Logger.Info("piers GetBlobber")
	var blobberID = r.URL.Query().Get("blobber_id")
	if blobberID == "" {
		err := common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
		common.Respond(w, r, "", err)
		return
	}

	blobber, err := srh.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		err := common.NewErrInternal("missing blobber" + blobberID)
		common.Respond(w, r, "", err)
		return
	}

	sn, err := blobberTableToStorageNode(*blobber)
	if err != nil {
		err := common.NewErrInternal("parsing blobber" + blobberID)
		common.Respond(w, r, "", err)
		return
	}
	common.Respond(w, r, sn, nil)
}
