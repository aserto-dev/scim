package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aserto-dev/scim/pkg/app"
	"github.com/aserto-dev/scim/pkg/version"
	"github.com/spf13/cobra"
)

var (
	flagConfigPath string
)

var rootCmd = &cobra.Command{
	Use:           "aserto-scim [flags]",
	SilenceErrors: true,
	SilenceUsage:  true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("aserto-scim %s\n", version.GetInfo().Version)
	},
}

var cmdRun = &cobra.Command{
	Use:   "run [args]",
	Short: "Start SCIM service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.Run(flagConfigPath, os.Stdout, os.Stderr)
	},
}

// nolint: gochecknoinits
func init() {
	cmdRun.Flags().StringVarP(&flagConfigPath, "config", "c", "", "config path")
	rootCmd.AddCommand(cmdRun)
}

func main() {
	rootCmd.AddCommand(
		versionCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}
