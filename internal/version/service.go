package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"

	"github.com/coreos/go-semver/semver"
	"github.com/inconshreveable/go-update"
	"github.com/sourcegraph/src-cli/internal/api"
)

type VersionService struct {
	Client api.Client
}

func (s *VersionService) GetRecommendedVersion(ctx context.Context) (string, error) {
	req, err := s.Client.NewHTTPRequest(ctx, "GET", "/.api/src-cli/version", nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusNotFound {
			return "", nil
		}

		return "", fmt.Errorf("error: %s\n\n%s", res.Status, body)
	}

	payload := struct {
		Version string `json:"version"`
	}{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}

	return payload.Version, nil
}

func (s *VersionService) MustNotBeOutdated(ctx context.Context, buildTag string) error {
	recommendedVersion, err := s.GetRecommendedVersion(ctx)
	if err != nil {
		return err
	}
	if buildTag != "dev" {
		if have, want := semver.New(recommendedVersion), semver.New(buildTag); have.LessThan(*want) {
			return fmt.Errorf("outdated version of src-cli, this Sourcegraph instance requires %s, but is %s. Run src update to update to the latest version", want, have)
		}
	}
	return nil
}

func (s *VersionService) UpgradeToRecommended(ctx context.Context) error {
	version := fmt.Sprintf("src_%s_amd64", runtime.GOOS)

	req, err := s.Client.NewHTTPRequest(ctx, "GET", "/.api/src-cli/"+version, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("error: %s\n\n%s", res.Status, body)
	}

	return update.Apply(res.Body, update.Options{})
}
