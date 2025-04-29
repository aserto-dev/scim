package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aserto-dev/scim/pkg/app"
	"github.com/aserto-dev/scim/pkg/version"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var flagConfigPath string

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
		srv, err := app.NewSCIMServer(flagConfigPath, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}

		errGroup, ctx := errgroup.WithContext(signals.SetupSignalHandler())
		errGroup.Go(srv.Run)

		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			return err
		}

		if err := errGroup.Wait(); err != nil {
			log.Printf("Error: %v", err)
		}

		log.Println("SCIM server stopped")
		return nil
	},
}

func init() { //nolint: gochecknoinits
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
