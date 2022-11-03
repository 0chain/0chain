package common

import (
	"net/url"
	"strconv"

	"0chain.net/core/common"
)

func GetStartEndBlock(values url.Values) (start int64, end int64, err error) {
	var (
		startBlockNum = values.Get("start")
		endBlockNum   = values.Get("end")
	)
	if endBlockNum == "" {
		return 0, 0, nil
	}
	start, err = strconv.ParseInt(startBlockNum, 10, 64)
	if err != nil {
		return 0, 0, common.NewErrBadRequest("start block number is not valid")
	}
	end, err = strconv.ParseInt(endBlockNum, 10, 64)
	if err != nil {
		return 0, 0, common.NewErrBadRequest("end block number is not valid")
	}

	if start > end {
		return 0, 0, common.NewErrBadRequest("start block number is greater than end block number")
	}
	return start, end, nil
}
