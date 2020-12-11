package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	apiQuery string
	apiVars  string

	apiCmd = &cobra.Command{
		Use:   "api",
		Short: "Interact with the Sourcegraph GraphQL API",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build the GraphQL request.
			if apiQuery == "" {
				// Read query from stdin instead.
				if isatty.IsTerminal(os.Stdin.Fd()) {
					return errors.New("expected query to be piped into 'src api' or -query flag to be specified")
				}
				data, err := ioutil.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				apiQuery = string(data)
			}

			// Determine which variables to use in the request.
			vars := map[string]interface{}{}
			if apiVars != "" {
				if err := json.Unmarshal([]byte(apiVars), &vars); err != nil {
					return err
				}
			}
			for _, arg := range args {
				idx := strings.Index(arg, "=")
				if idx == -1 {
					return errors.Errorf("parsing argument %q expected 'variable=value' syntax (missing equals)", arg)
				}
				key := arg[:idx]
				value := arg[idx+1:]
				vars[key] = value
			}

			// Perform the request.
			var result interface{}
			if ok, err := apiClient.NewRequest(apiQuery, vars).DoRaw(cmd.Context(), &result); err != nil || !ok {
				return err
			}

			// Print the formatted JSON.
			f, err := marshalIndent(result)
			if err != nil {
				return err
			}
			fmt.Println(string(f))
			return nil
		},
	}
)

func init() {
	apiCmd.Flags().StringVarP(&apiQuery, "query", "q", "", "GraphQL query to execute, e.g. 'query { currentUser { username } }' (stdin otherwise)")
	apiCmd.Flags().StringVarP(&apiVars, "vars", "V", "", `GraphQL query variables to include as JSON string, e.g. '{"var": "val", "var2": "val2"}'`)

	rootCmd.AddCommand(apiCmd)
}
