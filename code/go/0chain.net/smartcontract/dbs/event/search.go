package event

import (
	"database/sql"
	"fmt"
	"strconv"
)

// GetGenericSearchType returns kind of search query
func (edb *EventDb) GetGenericSearchType(query string) (string, error) {
	var queryType string
	var round int

	roundSearchQueryString := ""
	round, err := strconv.Atoi(query)
	if err == nil {
		roundSearchQueryString = fmt.Sprintf(
			`WHEN EXISTS (
				SELECT Id FROM BLOCKS WHERE ROUND = %d
			) THEN 'BlockRound'`, round,
		)
	}

	res := edb.Store.Get().Raw(
		fmt.Sprintf(
			`
			SELECT 
			CASE 
			%s
			WHEN EXISTS (
				SELECT Id FROM USERS WHERE USER_ID = @query
			) THEN 'UserId' 
			WHEN EXISTS (
				SELECT Id FROM TRANSACTIONS WHERE HASH = @query
			) THEN 'TransactionHash' 
			WHEN EXISTS (
				SELECT Id FROM BLOCKS WHERE HASH = @query
			) THEN 'BlockHash' 
			WHEN EXISTS (
				SELECT Id FROM WRITE_MARKERS WHERE CONTENT_HASH = @query
			) THEN 'ContentHash' 
			WHEN EXISTS (
				SELECT Id FROM WRITE_MARKERS WHERE NAME = @query
			) THEN 'FileName'
			ELSE 'Not Found' 
			END AS queryType
		`, roundSearchQueryString,
		),
		sql.Named("query", query),
	).Scan(&queryType)

	return queryType, res.Error
}
