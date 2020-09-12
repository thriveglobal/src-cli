package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/sourcegraph/src-cli/internal/api"
	"github.com/sourcegraph/src-cli/internal/output"
	"github.com/sourcegraph/src-cli/internal/version"
)

func init() {
	usage := `
Examples:

  Update the src-cli version to the Sourcegraph instance's recommended version:

    	$ src update
`

	flagSet := flag.NewFlagSet("update", flag.ExitOnError)
	var apiFlags = api.NewFlags(flagSet)

	var approveFlag = flagSet.Bool("yes", false, "Specify --yes to approve the update. Required for non-TTY.")

	handler := func(args []string) error {
		ctx := context.Background()
		out := output.NewOutput(flagSet.Output(), output.OutputOpts{})
		client := cfg.apiClient(apiFlags, flagSet.Output())
		svc := version.VersionService{Client: client}
		if !*approveFlag {
			prompt := promptui.Prompt{IsConfirm: true, Label: "Update src-cli to the latest recommended version for your Sourcegraph instance"}
			_, err := prompt.Run()
			if err != nil {
				// err means "no".
				return err
			}
		}

		pending := out.Pending(output.Line("", output.StylePending, "Applying update"))
		defer pending.Complete(output.Line("", output.StyleSuccess, "Update applied"))

		return svc.UpgradeToRecommended(ctx)
	}

	// Register the command.
	commands = append(commands, &command{
		flagSet: flagSet,
		handler: handler,
		usageFunc: func() {
			fmt.Fprintf(flag.CommandLine.Output(), "Usage of 'src %s':\n", flagSet.Name())
			flagSet.PrintDefaults()
			fmt.Println(usage)
		},
	})
}
