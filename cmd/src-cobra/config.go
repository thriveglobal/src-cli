package main

import (
	"io"
	"os"
	"strings"

	"github.com/sourcegraph/src-cli/internal/api"
)

type config struct {
	Endpoint          string            `json:"endpoint"`
	AccessToken       string            `json:"accessToken"`
	AdditionalHeaders map[string]string `json:"additionalHeaders"`

	ConfigFilePath string
}

// apiClient returns an api.Client built from the configuration.
func (c *config) apiClient(flags *api.Flags, out io.Writer) api.Client {
	return api.NewClient(api.ClientOpts{
		Endpoint:          c.Endpoint,
		AccessToken:       c.AccessToken,
		AdditionalHeaders: c.AdditionalHeaders,
		Flags:             flags,
		Out:               out,
	})
}

var testHomeDir string // used by tests to mock the user's $HOME

// readConfig reads the config file from the given path.
func readConfig() (*config, error) {
	envToken := os.Getenv("SRC_ACCESS_TOKEN")
	envEndpoint := os.Getenv("SRC_ENDPOINT")

	cfg := config{}

	// Apply config overrides.
	if envToken != "" {
		cfg.AccessToken = envToken
	}
	if envEndpoint != "" {
		cfg.Endpoint = envEndpoint
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://sourcegraph.com"
	}

	cfg.AdditionalHeaders = parseAdditionalHeaders()

	cfg.Endpoint = cleanEndpoint(cfg.Endpoint)
	return &cfg, nil
}

func cleanEndpoint(urlStr string) string {
	return strings.TrimSuffix(urlStr, "/")
}
