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
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

const version = "0.0.2"

// RootCmd represents the base command when called without any subcommands
var (
	RootCmd = &cobra.Command{
		Use:     "rdsdba",
		Short:   "RDS DBA CLI tool",
		Version: version,
		Long: `RDS DBA CLI plan to provide rich features to support general
database operations, troubleshooting, performance diagnose, etc`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	RootCmd.CompletionOptions.DisableDefaultCmd = true
	err := RootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing rdsdba CLI '%s'\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&cfg.DSN.Host, "host", "H", "localhost", "RDS host")
	RootCmd.PersistentFlags().IntVarP(&cfg.DSN.Port, "port", "P", 3306, "RDS port")
	RootCmd.PersistentFlags().StringVarP(&cfg.DSN.User, "user", "u", "root", "RDS user")
	RootCmd.PersistentFlags().StringVarP(&cfg.DSN.Passwd, "password", "p", "", "RDS password")
	RootCmd.PersistentFlags().IntVarP(&cfg.MaxOpenConns, "max-connection", "c", 50, "max number of open connections to RDS")
	RootCmd.PersistentFlags().IntVarP(&cfg.MaxIdleConns, "max-idle-connection", "i", 50, "max number of idle connections to RDS")
	RootCmd.PersistentFlags().DurationVarP(&cfg.ConnMaxLifeTime, "connection-max-lifetime", "l", -1*time.Minute, "the maximum amount of time a connection may be reused, less than 0 means never timeout, support time duration [s|m|h], suggest keep default") // by default never timeout, for long-running queries
	RootCmd.PersistentFlags().BoolVarP(&cfg.Debug, "debug", "D", false, "show debug level log")
	RootCmd.MarkFlagsRequiredTogether("host", "user", "password")

}
