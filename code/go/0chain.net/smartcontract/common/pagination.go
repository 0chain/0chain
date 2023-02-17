package common

import (
	"fmt"
	"net/url"
	"strconv"

	"0chain.net/core/common"
)

const DefaultQueryLimit = 20
const MaxQueryLimit = 50

type Pagination struct {
	Offset       int
	Limit        int
	IsDescending bool
}

func GetOffsetLimitOrderParam(values url.Values) (Pagination, error) {
	var (
		offsetString = values.Get("offset")
		limitString  = values.Get("limit")
		sort         = values.Get("sort")

		limit        = DefaultQueryLimit
		offset       = 0
		isDescending = false
		err          error
	)

	if offsetString != "" {
		offset, err = strconv.Atoi(offsetString)
		if err != nil {
			return Pagination{Limit: DefaultQueryLimit}, common.NewErrBadRequest("offset parameter is not valid")
		}
	}

	if limitString != "" {
		limit, err = strconv.Atoi(limitString)
		if err != nil {
			return Pagination{Limit: DefaultQueryLimit}, common.NewErrBadRequest("limit parameter is not valid")
		}

		if limit > MaxQueryLimit {
			msg := fmt.Sprintf("limit %d too high, cannot exceed %d", limit, DefaultQueryLimit)
			return Pagination{Limit: MaxQueryLimit}, common.NewErrBadRequest(msg)
		}
	}

	if sort != "" {
		switch sort {
		case "desc":
			isDescending = true
		case "asc":
			isDescending = false
		default:
			return Pagination{Limit: DefaultQueryLimit}, err
		}
	}

	return Pagination{Offset: offset, Limit: limit, IsDescending: isDescending}, nil
}

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
