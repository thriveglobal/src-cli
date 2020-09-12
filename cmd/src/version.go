package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/sourcegraph/src-cli/internal/api"
	"github.com/sourcegraph/src-cli/internal/version"
)

// buildTag is the git tag at the time of build and is used to
// denote the binary's current version. This value is supplied
// as an ldflag at compile time in travis.
var buildTag = "dev"

func init() {
	usage := `
Examples:

  Get the src-cli version and the Sourcegraph instance's recommended version:

    	$ src version
`

	flagSet := flag.NewFlagSet("version", flag.ExitOnError)
	var (
		apiFlags = api.NewFlags(flagSet)
	)

	handler := func(args []string) error {
		client := cfg.apiClient(apiFlags, flagSet.Output())

		fmt.Printf("Current version: %s\n", buildTag)

		svc := version.VersionService{Client: client}

		recommendedVersion, err := svc.GetRecommendedVersion(context.Background())
		if err != nil {
			return err
		}
		if recommendedVersion == "" {
			fmt.Println("Recommended Version: <unknown>")
			fmt.Println("This Sourcegraph instance does not support this feature.")
			return nil
		}
		fmt.Printf("Recommended Version: %s\n", recommendedVersion)
		// validate := func(input string) error {
		// 	_, err := url.ParseRequestURI(input)
		// 	if err != nil {
		// 		return errors.New("Invalid URL")
		// 	}
		// 	return nil
		// }
		// {
		// 	prompt := promptui.Prompt{
		// 		Label:    "Sourcegraph URL",
		// 		Default:  "https://sourcegraph.com",
		// 		Validate: validate,
		// 	}

		// 	_, err := prompt.Run()
		// 	if err != nil {
		// 		return err
		// 	}
		// }
		// {
		// 	prompt := promptui.Prompt{
		// 		Label: "Access token",
		// 	}

		// 	_, err := prompt.Run()
		// 	if err != nil {
		// 		return err
		// 	}
		// }
		return nil
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
