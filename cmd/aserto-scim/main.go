package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

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
		srv, err := app.NewSCIMServer(flagConfigPath, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}

		go func() {
			if err := srv.Run(); err != nil {
				log.Printf("Error running SCIM server: %v", err)
			}
		}()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, os.Kill)
		<-stop

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			return err
		}
		log.Println("SCIM server stopped")
		return nil
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
