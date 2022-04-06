package rest

import (
	"net/http"

	"0chain.net/smartcontract/storagesc"

	"0chain.net/core/common"
)

type StorageRestHandler struct {
	*RestHandler
}

func NewStorageRestHandler(rh *RestHandler) *StorageRestHandler {
	return &StorageRestHandler{rh}
}

func SetupStorageRestHandler(rh *RestHandler) {
	srh := NewStorageRestHandler(rh)
	storage := "/v1/screst/" + storagesc.ADDRESS
	http.HandleFunc(storage+"/get_blobber_count", srh.getBlobberCount)
	http.HandleFunc(storage+"/getBlobber", srh.getBlobber)
	http.HandleFunc(storage+"/get_blobber_total_stakes", srh.getBlobberTotalStakes)
	http.HandleFunc(storage+"/get_blobber_lat_long", srh.getBlobberGeoLocation)
	http.HandleFunc(storage+"/getblobbers", srh.getBlobbers)
}

// GetBlobbersHandler returns list of all blobbers alive (e.g. excluding
// blobbers with zero capacity).

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers getblobbers
// Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity).
//
// responses:
//  200: body:storagesc.StorageNodes
//  500:
func (srh *StorageRestHandler) getBlobbers(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetBlobbers()
	if err != nil || len(blobbers) == 0 {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, "", err)
		return
	}

	var sns storagesc.StorageNodes
	for _, blobber := range blobbers {
		sn, err := blobberTableToStorageNode(blobber)
		if err != nil {
			err := common.NewErrInternal("parsing blobber" + blobber.BlobberID)
			common.Respond(w, r, "", err)
			return
		}
		ssn := storagesc.StorageNode(sn)
		sns.Nodes.Add(&ssn)
	}
	common.Respond(w, r, sns, nil)
}

// getBlobberGeoLocation swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_lat_long get_blobber_lat_long
// Gets list of latitude and longitude for all blobbers
//
// responses:
//  200: body:event.BlobberLatLong
//  500:
func (srh *StorageRestHandler) getBlobberGeoLocation(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetAllBlobberLatLong()
	if err != nil {
		err := common.NewErrInternal("cannot get blobber geolocation" + err.Error())
		common.Respond(w, r, "", err)
		return
	}

	common.Respond(w, r, blobbers, nil)
}

// GetBlobberTotalStakes swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_total_stakes get_blobber_total_stakes
// Gets total stake of all blobbers combined
//
// responses:
//  200: intMap
//  500:
func (srh *StorageRestHandler) getBlobberTotalStakes(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetAllBlobberId()
	if err != nil {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, "", err)
		return
	}
	var total int64
	for _, blobber := range blobbers {
		var sp storageStakePool
		if err := sp.get(blobber, *srh); err != nil {
			err := common.NewErrInternal("cannot get stake pool" + err.Error())
			common.Respond(w, r, "", err)
			return
		}
		total += int64(sp.Stake())
	}
	common.Respond(w, r, int64Map{
		"total": total,
	}, nil)
}

// GetBlobberCount swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_count get_blobber_count
// Get count of blobber
//
// responses:
//  200: intMap
//  400:
func (srh StorageRestHandler) getBlobberCount(w http.ResponseWriter, r *http.Request) {
	blobberCount, err := srh.GetEventDB().GetBlobberCount()
	if err != nil {
		err := common.NewErrInternal("getting blobber count:" + err.Error())
		common.Respond(w, r, "", err)
		return
	}

	common.Respond(w, r, int64Map{
		"count": blobberCount,
	}, nil)
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
func (srh StorageRestHandler) getBlobber(w http.ResponseWriter, r *http.Request) {
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
