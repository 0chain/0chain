package smartcontract

import (
	"errors"
	"fmt"
)

func NewError(err error, msgs ...interface{}) error {
	if len(msgs) == 0 {
		return err
	}

	msg := msgs[0]
	for i := 1; i < len(msgs); i++ {
		msg = fmt.Sprintf("%v: %v", msg, msgs[i])
	}
	return fmt.Errorf("%w: %v", err, msg)
}

const noResourceErrMsg = "can't get resource"

// noResourceErr represents error corresponds to http.StatusNotFound.
type noResourceErr struct {
	msg string
}

// Make sure noResourceErr implement error interface.
var _ error = (*noResourceErr)(nil)

func (e *noResourceErr) Error() string {
	return e.msg
}

func WrapErrNoResource(err error) error {
	return fmt.Errorf("%w: %s", NewErrNoResource(), err)
}

var noResource = &noResourceErr{msg: noResourceErrMsg}

func NewErrNoResource() error {
	return noResource
}

var (
	FailRetrievingLimitsErr         = errors.New("can't get limits")
	FailRetrievingConfigErr         = errors.New("can't get config")
	FailRetrievingStatsErr          = errors.New("can't get stats")
	MinerDoesntExistErr             = errors.New("unknown miner")
	FailRetrievingUserNodeErr       = errors.New("can't get user node")
	FailRetrievingMinerNodeErr      = errors.New("can't get miner node")
	FailRetrievingMinersListErr     = errors.New("can't get miners list")
	FailRetrievingShardersListErr   = errors.New("can't get sharders list")
	FailRetrievingPhaseNodeErr      = errors.New("can't get phase node")
	FailRetrievingMinersDKGListErr  = errors.New("can't get miners dkg list")
	FailRetrievingMinersMpksListErr = errors.New("can't get miners mpks list")
	FailRetrievingGroupErr          = errors.New("can't get group")
	FailRetrievingMagicBlockErr     = errors.New("can't get magic block")
	PoolStatsNotFoundErr            = errors.New("pool stats not found")
	RetrievingGlobalNodeErr         = errors.New("can't get global node")
	FailRetrievingReadMarker        = errors.New("can't get read marker")
	FailRetrievingAllocationErr     = errors.New("can't retrieve allocation")
	FailRetrievingAllocationList    = errors.New("can't get allocation list")
	FailAllocationMinLockErr        = errors.New("allocation min lock failed")
	NoRegisteredBlobberErr          = errors.New("no blobbers registered, failed to check min allocation lock")
	NotEnoughBlobbersErr            = errors.New("not enough blobbers to honor the allocation")
	FailRetrievingPreferredBlobbers = errors.New("can't get preferred blobbers")
	BlobberChallengeReadErr         = errors.New("error reading blobber challenge from DB")
	FailRetrievingBlobbersListErr   = errors.New("can't get blobbers list")
	FailRetrievingBlobberErr        = errors.New("can't get blobber")
	FailRetrievingReadPoolErr       = errors.New("can't get read pool")
	FailRetrievingWritePoolErr      = errors.New("can't get write pool")
	FailRetrievingStakePool         = errors.New("can't get related stake pool")
	FailRetrievingUserStakePoolErr  = errors.New("can't get user stake pools")
	FailRetrievingChallengePoolErr  = errors.New("can't get challenge pool")
	FailRetrievingPoolErr           = errors.New("can't get pool")
	FailGetOrCreateClientPoolsErr   = errors.New("can't get or create client pools")
)

const internalErrMsg = "internal err"

// internalErr represents error corresponds to http.StatusInternalServerError.
type internalErr struct {
	msg string
}

// Make sure internalErr implement error interface.
var _ error = (*internalErr)(nil)

func (e *internalErr) Error() string {
	return e.msg
}

func WrapErrInternal(err error) error {
	return fmt.Errorf("%w: %s", NewErrInternal(), err)
}

var internal = &internalErr{msg: internalErrMsg}

func NewErrInternal() error {
	return internal
}

var (
	DecodingErr                 = errors.New("can't decode resource")
	FailDecodingMinerErr        = errors.New("can't decode miner from passed params")
	FailDecodingMpksBytesErr    = errors.New("can't decode mpks bytes")
	FailDecodingGroupErr        = errors.New("can't decode group")
	FailDecodingMagicBlockErr   = errors.New("can't decode magic block")
	FailDecodingReadMarker      = errors.New("can't decode read marker")
	FailDecodingAllocationErr   = errors.New("can't decode allocation")
	BlobberChallengeDecodingErr = errors.New("fail decoding blobber challenge")
)

const invalidRequestErrMsg = "invalid request"

// invalidRequestErr represents error corresponds to http.StatusBadRequest.
type invalidRequestErr struct {
	msg string
}

// Make sure invalidRequestErr implement error interface.
var _ error = (*invalidRequestErr)(nil)

func (e *invalidRequestErr) Error() string {
	return e.msg
}

func WrapErrInvalidRequest(err error) error {
	return fmt.Errorf("%w: %s", NewErrInvalidRequest(), err)
}

var invalidRequest = &invalidRequestErr{msg: invalidRequestErrMsg}

func NewErrInvalidRequest() error {
	return invalidRequest
}
