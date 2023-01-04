package mysql

import (
	"fmt"
	"strings"
)

type Table struct {
	SchemaName string `mapstructure:"table_schema"`
	TableName  string `mapstructure:"table_name"`
}

func TabStrToTabStruct(tablesStr []string) ([]Table, error) {
	//"schema_name.table_name"
	var tables []Table
	var tableName string
	var schemaName string
	err := fmt.Errorf("unexpected format")
	for _, table := range tablesStr {
		res := strings.Split(table, ".")
		if len(res) != 2 {
			return nil, err
		}
		schemaName = strings.TrimSpace(res[0])
		tableName = strings.TrimSpace(res[1])
		tables = append(tables, Table{SchemaName: schemaName, TableName: tableName})
	}

	return tables, nil
}
