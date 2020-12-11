package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sourcegraph/src-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	cfg *config

	debugLog string
	verbose  bool

	apiClient  api.Client
	apiDump    bool
	apiGetCurl bool
	apiTrace   bool

	rootCmd = &cobra.Command{
		Use:   "src",
		Short: "src is a tool that provides access to Sourcegraph instances.",
		Long: `src is a tool that provides access to Sourcegraph instances.
For more information, see https://github.com/sourcegraph/src-cli`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if debugLog != "" {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)

				switch debugLog {
				case "-", "stdout":
					log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
				case "stderr":
					log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
				default:
					lw, err := os.Create(debugLog)
					if err != nil {
						log.Warn().Err(err).Str("debugLog", debugLog).Msg("cannot create debug log")
						return err
					}
					log.Logger = log.Output(lw)
				}
			}

			var err error
			cfg, err = readConfig()
			if err != nil {
				return err
			}

			apiClient = api.NewClient(api.ClientOpts{
				Endpoint:          cfg.Endpoint,
				AccessToken:       cfg.AccessToken,
				AdditionalHeaders: cfg.AdditionalHeaders,
				Flags: &api.Flags{
					Dump:    &apiDump,
					GetCurl: &apiGetCurl,
					Trace:   &apiTrace,
				},
				Out: cmd.OutOrStdout(),
			})

			return nil
		},
	}
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	rootCmd.PersistentFlags().StringVar(&debugLog, "debug-log", "", "write a debug log to this file; - or stdout will write to stdout; stderr will write to stderr")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "print verbose output")

	rootCmd.PersistentFlags().BoolVar(&apiDump, "dump-requests", false, "log GraphQL requests and responses to stdout")
	rootCmd.PersistentFlags().BoolVar(&apiGetCurl, "get-curl", false, "print curl commands corresponding to each request issued (WARNING: includes printing your access token!)")
	rootCmd.PersistentFlags().BoolVar(&apiTrace, "trace", false, "log the trace ID for requests; see https://docs.sourcegraph.com/admin/observability/tracing")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
