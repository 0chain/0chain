package common

import (
	"fmt"
	"net/url"
	"strconv"

	"0chain.net/core/common"
)

const DefaultQueryLimit = 20

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

		if limit > DefaultQueryLimit {
			msg := fmt.Sprintf("limit %d too high, cannot exceed %d", limit, DefaultQueryLimit)
			return Pagination{Limit: DefaultQueryLimit}, common.NewErrBadRequest(msg)
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
