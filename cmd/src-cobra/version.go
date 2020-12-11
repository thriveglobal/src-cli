package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/spf13/cobra"
)

var (
	buildTag = "dev"

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Display and compare the src version against the recommended version for your instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Current version: %s\n", buildTag)

			recommendedVersion, err := getRecommendedVersion(cmd.Context())
			if err != nil {
				return err
			}
			if recommendedVersion == "" {
				fmt.Println("Recommended Version: <unknown>")
				fmt.Println("This Sourcegraph instance does not support this feature.")
				return nil
			}
			fmt.Printf("Recommended Version: %s\n", recommendedVersion)

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func getRecommendedVersion(ctx context.Context) (string, error) {
	req, err := apiClient.NewHTTPRequest(ctx, "GET", "/.api/src-cli/version", nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return "", nil
		}

		return "", fmt.Errorf("error: %s\n\n%s", resp.Status, body)
	}

	payload := struct {
		Version string `json:"version"`
	}{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}

	return payload.Version, nil
}
