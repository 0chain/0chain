package rest

import (
	"net/http"

	"0chain.net/core/common"
)

type StorageRestHandler struct {
	RestHandler
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7 get_blobber_count
// Get count of blobber
//
// security:
// - apiKey: []
// responses:
//  401: CommonError
//  200: GetBlobberCount
func (srh StorageRestHandler) GetBlobberCountHandler(w http.ResponseWriter, r *http.Request) {
	blobberCount, err := srh.GetEventDB().GetBlobberCount()
	resp := map[string]int64{
		"count": blobberCount,
	}

	common.Respond(w, r, resp, err)
}
