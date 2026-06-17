package database

import (
	"strconv"
)

func PrepareBulkInsert(columns, values int) string {
	stmt := ""
	for i := range values {
		stmt += `(`
		n := i * columns
		for j := range columns {
			stmt += `$` + strconv.Itoa(n+j+1) + `,`
		}
		stmt = stmt[:len(stmt)-1] + `),`
	}

	if len(stmt) > 0 {

		// Remove last comma and Return the statement
		return stmt[:len(stmt)-1]
	}

	return stmt
}
