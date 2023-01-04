package internal

import (
	"context"
	"rdsdba/pkg/mysql"
)

type RDS interface {
	GetUserTables(ctx context.Context) ([]mysql.Table, error)
	WarmUp(ctx context.Context, table mysql.Table) error
}
