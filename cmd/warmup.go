package cmd

/*
Copyright © 2023 Yin Xi <sherry.yin@grabtaxi.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"rdsdba/internal"
	"rdsdba/pkg/mysql"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	mapset "github.com/deckarep/golang-set/v2"
)

var (
	cfg          mysql.Config
	logger       zerolog.Logger
	skip         []string
	only         []string
	userTables   []mysql.Table
	warmUpTables []mysql.Table
	progress     string

	WarmupCmd = &cobra.Command{
		Use:   "warmup",
		Short: "Warm up MySQL InnoDB buffer pool",
		Long: `Warm up RDS MySQL instance InnoDB buffer pool by reading data on disk and load into memory to
speed up upcoming traffic.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := run()
			if err != nil {
				os.Exit(1)
			}
		},
	}
)

func init() {
	RootCmd.AddCommand(WarmupCmd)

	WarmupCmd.Flags().StringVarP(&cfg.DSN.Host, "host", "H", "localhost", "mysql host")
	WarmupCmd.Flags().IntVarP(&cfg.DSN.Port, "port", "P", 3306, "mysql server port")
	WarmupCmd.Flags().StringVarP(&cfg.DSN.User, "user", "u", "root", "mysql user")
	WarmupCmd.Flags().StringVarP(&cfg.DSN.Passwd, "password", "p", "", "mysql password")
	WarmupCmd.Flags().IntVarP(&cfg.Concurrency, "thread", "t", 20, "number of threads")
	WarmupCmd.Flags().IntVarP(&cfg.MaxOpenConns, "max-connection", "c", 50, "max number of open mysql connections")
	WarmupCmd.Flags().IntVarP(&cfg.MaxIdleConns, "max-idle-connection", "i", 50, "max number of idle mysql connections")
	WarmupCmd.Flags().DurationVarP(&cfg.ConnMaxLifeTime, "connection-max-lifetime", "l", -1*time.Minute, "the maximum amount of time a connection may be reused, less than 0 means never timeout, support time duration [s|m|h], suggest keep default") // by default never timeout, for long-running queries
	WarmupCmd.Flags().BoolVarP(&cfg.Debug, "debug", "D", false, "show debug level log")
	WarmupCmd.Flags().StringSliceVarP(&skip, "skip", "s", nil, "skip cold tables to let them stay on disk, comma separated format:schema_name1.table_name1,schema_name2.table_name2, whitespaces between comma is allowed ")
	WarmupCmd.Flags().StringSliceVarP(&only, "only", "o", nil, "only load specific tables to memory, comma separated format:schema_name.table_name, schema_name2.table_name2, whitespaces between comma is allowed")
	//WarmupCmd.Flags().DurationVarP(&cfg.Sleep, "sleep", "S", 1*time.Second, "sleep time, support time duration [s|m|h]")
	WarmupCmd.MarkFlagsRequiredTogether("host", "user", "password")
	WarmupCmd.MarkFlagsMutuallyExclusive("skip", "only")
}

func warmUp(ctx context.Context, rds internal.RDS, table mysql.Table) error {
	err := rds.WarmUp(ctx, table)
	return err
}

func getUserTables(ctx context.Context, rds internal.RDS) ([]mysql.Table, error) {
	tables, err := rds.GetUserTables(ctx)
	return tables, err
}

func removeSkipTables(allUserTables []mysql.Table, skipTablesStr []string) ([]mysql.Table, error) {
	skipTables, err := mysql.TabStrToTabStruct(skipTablesStr)
	if err != nil {
		return nil, err
	}

	skipTableSet := mapset.NewSet[mysql.Table]()
	for _, skipTable := range skipTables {
		skipTableSet.Add(skipTable)
	}

	allUserTablesSet := mapset.NewSet[mysql.Table]()
	for _, table := range allUserTables {
		allUserTablesSet.Add(table)
	}

	warmUpTableSet := allUserTablesSet.Difference(skipTableSet)
	warmUpTables := warmUpTableSet.ToSlice()

	return warmUpTables, nil
}

func run() error {
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	logger = log.With().Logger()
	logger.Info().Msg("Warmup started")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	i, err := mysql.NewInstance(cfg)
	defer i.DB.Close()
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}

	logger.Info().Msg("Instance initialised")

	switch {
	case only != nil:
		warmUpTables, err = mysql.TabStrToTabStruct(only)
		if err != nil {

		}
	case skip != nil:
		userTables, err = getUserTables(ctx, i)
		warmUpTables, err = removeSkipTables(userTables, skip)
	default:
		warmUpTables, err = getUserTables(ctx, i)
	}

	chanSize := len(warmUpTables)
	if chanSize == 0 {
		logger.Warn().Msg("No tables to warm up, complete!")
		return nil
	}
	tableChan := make(chan mysql.Table, chanSize)

	go func() {
		for index := range warmUpTables {
			tableChan <- warmUpTables[index]
		}
		logger.Info().Msg("Load to queue finished")
	}()

	concurrency := i.Config.Concurrency
	if chanSize < i.Config.Concurrency {
		concurrency = chanSize
		logger.Info().Int("concurrency", concurrency).Msg("Changed concurrency as number of table less than specified concurrency")
	}

	x := 0
	for x < chanSize {
		var wg sync.WaitGroup

		for n := 0; n < concurrency; n++ {
			if x < chanSize {
				wg.Add(1)
				go func(x int, n int) {
					logger.Debug().Int("Job", x).Msg("Start")
					table := <-tableChan
					logger.Debug().Int("Job", x).Str("Schema", table.SchemaName).Str("Table", table.TableName).Msg("Table assigned to job")
					err := warmUp(ctx, i, table)
					if err != nil {
						logger.Warn().Int("Job", x).Str("Schema", table.SchemaName).Str("Table", table.TableName).Err(err).Msg("")
					} else {
						logger.Info().Int("Job", x).Str("Schema", table.SchemaName).Str("Table", table.TableName).Msg("Done")
					}
					wg.Done()
				}(x, n)
				x += 1
			}
		}
		wg.Wait()
		progress = strconv.Itoa(x) + "/" + strconv.Itoa(chanSize)
		logger.Info().Str("progress", progress).Msg("")
	}

	logger.Info().Int("Total warmup tables", chanSize).Int("Skipped tables", len(skip)).Msg("Warmup completed!")

	return nil
}
