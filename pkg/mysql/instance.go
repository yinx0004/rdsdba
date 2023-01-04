package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	ConnectionFmt = "%s:%s@tcp(%s:%d)/?timeout=%s&tmp_table_size=2147483648&max_heap_table_size=2147483648&maxAllowedPacket=0"
	SystemSchema  = " 'information_schema', 'innodb', 'mysql', 'performance_schema', 'sys' "
	pingTimeout   = 5 * time.Second
	connTimeout   = 10 * time.Second
)

var (
	ErrInstanceInitFailed = errors.New("instance initialise failed")
	ErrDBInitFailed       = errors.New("connection initialise failed %s")
)

type Config struct {
	Concurrency     int
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifeTime time.Duration
	Debug           bool
	Sleep           time.Duration
	DSN             struct {
		Host   string
		Port   int
		User   string
		Passwd string
	}
}

type Instance struct {
	Config Config
	logger zerolog.Logger
	DB     *sql.DB
}

func NewInstance(config Config) (*Instance, error) {
	logger := log.With().
		Str("user", config.DSN.User).
		Str("host", config.DSN.Host).
		Int("port", config.DSN.Port).
		Logger()

	i := &Instance{
		Config: config,
		logger: logger,
		DB:     &sql.DB{},
	}

	conn, err := i.Open()
	if err != nil {
		return nil, fmt.Errorf(ErrDBInitFailed.Error(), err)
	}
	i.DB = conn
	return i, nil
}

func (i *Instance) Open() (*sql.DB, error) {
	cfg := i.Config
	db, err := sql.Open("mysql",
		fmt.Sprintf(ConnectionFmt, cfg.DSN.User, cfg.DSN.Passwd, cfg.DSN.Host, cfg.DSN.Port, connTimeout))
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(cfg.ConnMaxLifeTime)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

func (i *Instance) Logger() zerolog.Logger {
	return i.logger
}

func (i *Instance) WarmUp(ctx context.Context, table Table) error {
	tableIdentifier := table.SchemaName + "." + table.TableName

	stmt1 := fmt.Sprintf("select count(*) from %s", tableIdentifier)
	stmt2 := fmt.Sprintf("ANALYZE TABLE %s", tableIdentifier)
	stmts := []string{stmt1, stmt2}
	for index := range stmts {
		_, _, _, err := Query(ctx, i.DB, stmts[index])
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Instance) GetUserTables(ctx context.Context) ([]Table, error) {
	var tables []Table
	stmt := fmt.Sprintf("select table_schema, table_name from information_schema.tables where table_schema not in (%s) and table_type='BASE TABLE'", SystemSchema)
	_, _, data, err := Query(ctx, i.DB, stmt)
	if err != nil {
		return nil, err
	}

	tables = make([]Table, 0, len(data))
	for index := range data {
		var table Table
		decodeConfig := &mapstructure.DecoderConfig{
			WeaklyTypedInput: true,
			Result:           &table,
		}
		decoder, decoderInitErr := mapstructure.NewDecoder(decodeConfig)
		if decoderInitErr != nil {
			return tables, decoderInitErr
		}
		err = decoder.Decode(data[index])
		if err != nil {
			return nil, err
		}

		tables = append(tables, table)
	}
	return tables, nil
}
