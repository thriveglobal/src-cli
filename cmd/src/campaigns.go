package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/sourcegraph/src-cli/internal/api"
	"github.com/sourcegraph/src-cli/internal/version"
)

var campaignsCommands commander

func init() {
	usage := `'src campaigns' is a tool that manages campaigns on a Sourcegraph instance.

Usage:

	src campaigns command [command options]

The commands are:

	apply                 applies a campaign spec to create or update a campaign
	preview               creates a campaign spec to be previewed or applied
	repos,repositories    queries the exact repositories that a campaign spec
	                      will apply to
	validate              validates a campaign spec

Use "src campaigns [command] -h" for more information about a command.

`

	flagSet := flag.NewFlagSet("campaigns", flag.ExitOnError)
	var apiFlags = api.NewFlags(flagSet)

	handler := func(args []string) error {
		client := cfg.apiClient(apiFlags, flagSet.Output())
		svc := version.VersionService{Client: client}
		err := svc.MustNotBeOutdated(context.Background(), buildTag)
		if err != nil {
			return err
		}
		campaignsCommands.run(flagSet, "src campaigns", usage, args)
		return nil
	}

	// Register the command.
	commands = append(commands, &command{
		flagSet:   flagSet,
		aliases:   []string{"campaign"},
		handler:   handler,
		usageFunc: func() { fmt.Println(usage) },
	})
}
