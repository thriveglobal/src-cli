package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/sourcegraph/src-cli/internal/api"
)

func init() {
	usage := `
This command removes users from a Sourcegraph instance who have been inactive for 60 or more days. Admin accounts are omitted by default.
	
Examples:

	$ src users clean -days 182
	
	$ src users clean -remove-admin
`

	flagSet := flag.NewFlagSet("clean", flag.ExitOnError)
	usageFunc := func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of 'src users %s':\n", flagSet.Name())
		flagSet.PrintDefaults()
		fmt.Println(usage)
	}
	var (
		daysToDelete     = flagSet.Int("days", 60, "Days threshold on which to remove users, must be 60 days or greater and defaults to this value ")
		removeAdmin      = flagSet.Bool("remove-admin", false, "clean admin accounts")
		skipConfirmation = flagSet.Bool("force", false, "skips user confirmation step allowing programmatic use")
		apiFlags         = api.NewFlags(flagSet)
	)

	handler := func(args []string) error {
		if err := flagSet.Parse(args); err != nil {
			return err
		}
		if *daysToDelete < 60 {
			fmt.Println("-days flag must be set to 60 or greater")
			return nil
		}

		ctx := context.Background()
		client := cfg.apiClient(apiFlags, flagSet.Output())

		currentUserQuery := `
query {
	currentUser {
		username
	}
}
`
		var currentUserResult struct {
			Data struct {
				CurrentUser struct {
					Username string
				}
			}
		}
		if ok, err := cfg.apiClient(apiFlags, flagSet.Output()).NewRequest(currentUserQuery, nil).DoRaw(context.Background(), &currentUserResult); err != nil || !ok {
			return err
		}

		usersQuery := `
query InactiveUsers ($inactiveSince: DateTime!) {
  users (inactiveSince: $inactiveSince) {
    totalCount
    nodes {
      username
      siteAdmin
	  emails {
		email
	  }
    }
  }
}
`

		// get users to delete
		var usersResult struct {
			Users struct {
				Nodes []User
			}
		}
		vars := map[string]interface{}{
			"inactiveSince": computeInactiveSince(*daysToDelete).Format(time.RFC3339),
		}
		if ok, err := client.NewRequest(usersQuery, vars).Do(ctx, &usersResult); err != nil || !ok {
			return err
		}

		usersToDelete := make([]User, 0)
		for _, user := range usersResult.Users.Nodes {
			// never remove user issuing command
			if user.Username == currentUserResult.Data.CurrentUser.Username {
				continue
			}
			if !*removeAdmin && user.SiteAdmin {
				continue
			}
			usersToDelete = append(usersToDelete, user)
		}

		if *skipConfirmation {
			for _, user := range usersToDelete {
				if err := removeUser(user, client, ctx); err != nil {
					return err
				}
			}
			return nil
		}

		// confirm and remove users
		if confirmed, _ := confirmUserRemoval(usersToDelete); !confirmed {
			fmt.Println("Aborting removal")
			return nil
		} else {
			fmt.Println("REMOVING USERS")
			for _, user := range usersToDelete {
				if err := removeUser(user, client, ctx); err != nil {
					return err
				}
			}
		}

		return nil
	}

	// Register the command.
	usersCommands = append(usersCommands, &command{
		flagSet:   flagSet,
		handler:   handler,
		usageFunc: usageFunc,
	})
}

// compute inactive since arg
func computeInactiveSince(days int) time.Time {
	return time.Now().AddDate(0, 0, -days)
}

// Issue graphQL api request to remove user
func removeUser(user User, client api.Client, ctx context.Context) error {
	query := `mutation DeleteUser($user: ID!) {
  deleteUser(user: $user) {
    alwaysNil
  }
}`
	vars := map[string]interface{}{
		"user": user.ID,
	}
	if ok, err := client.NewRequest(query, vars).Do(ctx, nil); err != nil || !ok {
		return err
	}
	return nil
}

// Verify user wants to remove users with table of users and a command prompt for [y/N]
func confirmUserRemoval(usersToRemove []User) (bool, error) {
	fmt.Printf("Users to remove from instance at %s\n", cfg.Endpoint)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Username", "Email"})
	for _, user := range usersToRemove {
		if len(user.Emails) > 0 {
			t.AppendRow([]interface{}{user.Username, user.Emails[0].Email})
			t.AppendSeparator()
		} else {
			t.AppendRow([]interface{}{user.Username, ""})
			t.AppendSeparator()
		}
	}
	t.SetStyle(table.StyleRounded)
	t.Render()
	input := ""
	for strings.ToLower(input) != "y" && strings.ToLower(input) != "n" {
		fmt.Printf("Do you  wish to proceed with user removal [y/N]: ")
		if _, err := fmt.Scanln(&input); err != nil {
			return false, err
		}
	}
	return strings.ToLower(input) == "y", nil
}
