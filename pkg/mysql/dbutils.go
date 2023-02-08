package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	MaxRowsSize = 1000
)

type QueryResponseInfo struct {
	SQLStmt       string
	RowsAffected  int64
	ErrString     string
	Duration      int64
	ExecutionTime int64
}

func Query(ctx context.Context, db *sql.DB, sqlStmt string) (QueryResponseInfo, []string, []map[string]interface{}, error) {
	var cols []string
	var tableData []map[string]interface{}
	var result QueryResponseInfo

	startTime := time.Now()
	rows, err := db.QueryContext(ctx, sqlStmt)
	endTime := time.Now()
	duration := int64(endTime.Sub(startTime)) / 1000000 // millisecond
	if err != nil {
		errString := fmt.Sprint(err)
		result = QueryResponseInfo{
			SQLStmt:       sqlStmt,
			RowsAffected:  0,
			ErrString:     errString,
			Duration:      duration,
			ExecutionTime: startTime.Unix(),
		}
		return result, cols, tableData, err
	}
	_ = rows.Err()
	defer rows.Close()

	cols, _ = rows.Columns()
	count := len(cols)
	colsTypeRaw, _ := rows.ColumnTypes()
	var colsType = make([]string, count)
	for i, colsT := range colsTypeRaw {
		colsType[i] = colsT.DatabaseTypeName()
	}

	tableData = make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuesPtr := make([]interface{}, count)
	for i := range cols {
		valuesPtr[i] = &values[i]
	}

	currentRow := 1
	for rows.Next() {
		if currentRow > MaxRowsSize {
			break
		}
		_ = rows.Scan(valuesPtr...)
		record := make(map[string]interface{})
		for i := range cols {
			if values[i] == nil {
				record[cols[i]] = "NULL"
			} else {
				v := string(values[i].([]byte))
				record[cols[i]] = v
			}
		}
		tableData = append(tableData, record)
		currentRow++
	}
	result = QueryResponseInfo{
		SQLStmt:       sqlStmt,
		RowsAffected:  int64(len(tableData)),
		ErrString:     "",
		Duration:      duration,
		ExecutionTime: startTime.Unix(),
	}
	return result, cols, tableData, nil
}
