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

package cmd

import (
	"context"
	"errors"
	"fmt"
	WeightedRandomChoice "github.com/kontoulis/go-weighted-random-choice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"rdsdba/internal"
	"rdsdba/internal/utils"
	"rdsdba/pkg/mysql"
	"strconv"
	"strings"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/spf13/cobra"
)

const (
	waitQueueCap    = 2048
	sqlChanCap      = 10000
	prepareDuration = 30 * time.Second
)

var (
	// StressCmd represents the stress command
	StressCmd = &cobra.Command{
		Use:   "stress",
		Short: "Run stress test on MySQL",
		Long: `Run stress test on MySQL,
specified queries instead of random queries.
`,

		Run: func(cmd *cobra.Command, args []string) {
			err := stressRun()
			if err != nil {
				if err == ErrFlagMissing {
					cmd.Help()
					fmt.Println("at lease one of flag [query file] needed!")
				} else {
					cmd.Help()
				}
			}
		},
	}
	query          string
	file           string
	duration       time.Duration
	statement      string
	errNum         []int
	rtRes          []int
	start          time.Time
	ErrFlagMissing = errors.New("flag missing")
)

func init() {
	RootCmd.AddCommand(StressCmd)

	StressCmd.Flags().IntVarP(&cfg.Concurrency, "thread", "t", 1, "number of threads(connections)")
	StressCmd.Flags().DurationVarP(&duration, "time", "T", 30*time.Second, "stress test time, support time duration [s|m|h]")
	StressCmd.Flags().StringVarP(&query, "query", "q", "", "single query used for stress test, accepted in command line")
	StressCmd.Flags().StringVarP(&file, "file", "f", "", "the file which contains multiple queries used for stress test, for each query must provide a weighted, separated by ';'")
	StressCmd.MarkFlagsMutuallyExclusive("query", "file")
}

func stressRun() error {
	if len(file) == 0 && len(query) == 0 {
		return ErrFlagMissing
	}

	// increase max connections to RDS if thread is higher
	if cfg.Concurrency > cfg.MaxOpenConns {
		cfg.MaxOpenConns = cfg.Concurrency
	}

	logger := initLogger()
	logger.Debug().Msg("stress test started...")

	ctx := context.Background()

	i, err := mysql.NewInstance(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}
	defer i.DB.Close()
	logger.Debug().Msg("initialised")

	wp := workerpool.New(cfg.Concurrency)

	switch {
	case len(file) > 0:
		wrc := WeightedRandomChoice.New()
		stmts, err := processStmsFromFile(file)
		if err != nil {
			return err
		}
		wrc.AddElements(stmts)

		sqlChan := make(chan string, sqlChanCap)
		ctxCancel, cancelWorker := context.WithCancel(ctx)
		defer cancelWorker()

		// prepare sql queue based on weight to be run
		logger.Info().Msg("prepare")
		go stmtGen(ctxCancel, wrc, sqlChan)
		time.Sleep(prepareDuration)

		logger.Info().Msg("start")
		start = time.Now()
		ctxTimeout, timeoutCancel := context.WithTimeout(ctx, duration)
		defer timeoutCancel()
		errNum, rtRes = multiple(ctxTimeout, wp, i, sqlChan)
	case len(query) > 0:
		statement = query
		logger.Info().Msg("start")
		start = time.Now()
		ctxTimeout, timeoutCancel := context.WithTimeout(ctx, duration)
		defer timeoutCancel()
		errNum, rtRes = single(ctxTimeout, wp, i, statement)
	default:
		return err
	}

	wp.StopWait()
	logger.Info().Msg("end")
	end := time.Now()

	ignoredErr := len(errNum)
	totalQueries := len(rtRes)
	totalTime := end.Sub(start).Seconds()
	qps := float64(totalQueries) / totalTime

	latencyMin, latencyMax := utils.FindMinAndMax(rtRes)
	latencyAvg := utils.Avg(rtRes)
	latency95 := utils.NinetyFifth(rtRes)

	result := fmt.Sprintf(`
	Latency(ms):
		min:                %d
		avg:                %d
		max:                %d
		95th percentile:    %d

	General statistics:
		total time:         %fs
		total queries:      %d

	SQL statistics:
		qps:                %f
		ignored errors:     %d
`, latencyMin, latencyAvg, latencyMax, latency95, totalTime, totalQueries, qps, ignoredErr)
	fmt.Println(result)

	return nil
}

func initLogger() zerolog.Logger {
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	logger = log.With().Logger()
	return logger
}

func single(ctx context.Context, wp *workerpool.WorkerPool, rds internal.RDS, query string) ([]int, []int) {
	var errNum []int
	var rtRes []int

	for {
		select {
		case <-ctx.Done():
			wp.Stop()
			return errNum, rtRes
		default:
			// when waiting queue too long, do nothing
			if wp.WaitingQueueSize() > waitQueueCap {
				logger.Debug().Msg("worker pool waiting queue too long")
				continue
			}
			wp.Submit(func() {
				rt, err := rds.Stress(ctx, query)
				if err != nil {
					errNum = append(errNum, 1)
				} else {
					rtRes = append(rtRes, int(rt))
				}
			})
		}
	}
}

func stmtGen(ctx context.Context, wrc WeightedRandomChoice.WeightedRandomChoice, sqlchan chan string) {
	for {
		select {
		case <-ctx.Done():
			logger.Debug().Err(ctx.Err()).Msg("stop statement generator")
			return
		default:
			stmt := wrc.GetRandomChoice()
			select {
			case sqlchan <- stmt:
				continue
			default:
				// when sql channel full, do nothing, won't block
				logger.Debug().Msg("sql channel full")
				continue
			}
		}
	}
}

func multiple(ctx context.Context, wp *workerpool.WorkerPool, rds internal.RDS, sqlChan chan string) ([]int, []int) {
	var errNum []int
	var rtRes []int

	for {
		select {
		case <-ctx.Done():
			wp.Stop()
			return errNum, rtRes
		default:
			// when waiting queue too long, do nothing
			if wp.WaitingQueueSize() > waitQueueCap {
				logger.Debug().Msg("worker pool waiting queue too long")
				continue
			}
			select {
			case query := <-sqlChan:
				wp.Submit(func() {
					rt, err := rds.Stress(ctx, query)
					if err != nil {
						errNum = append(errNum, 1)
					} else {
						rtRes = append(rtRes, int(rt))
					}

				})
			default:
				// when sql channel empty, do nothing, won't block
				logger.Debug().Msg("sql channel empty")
				continue
			}
		}
	}
}

func processStmsFromFile(file string) (map[string]int, error) {
	lines, err := utils.FileLineByLine(file)
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}

	statements := make(map[string]int)

	for index, line := range lines {
		statementWeight := strings.Split(line, ";")
		if len(statementWeight) != 2 {
			logger.Fatal().Int("Line number", index).Msg("please check this line, must be delimiter separated")
		}

		stmt := statementWeight[0]
		weight, err := strconv.Atoi(strings.TrimSpace(statementWeight[1]))
		if err != nil {
			logger.Fatal().Err(err).Msg("")
		}
		statements[stmt] = weight

	}

	return statements, nil
}
