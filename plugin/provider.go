package plugin

import (
	"context"
	"net/http"

	"github.com/drone/drone-go/drone"
	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/transport"

	"github.com/prometheus/client_golang/prometheus"
)

func GetGithubFilesChanged(repo drone.Repo, build drone.Build, token string) ([]string, error) {
	newctx := context.Background()
	client := github.NewDefault()
	client.Client = &http.Client{
		Transport: &transport.BearerToken{
			Token: token,
		},
	}

	var changes []*scm.Change
	var result *scm.Response
	var err error

	if build.Before == "" || build.Before == scm.EmptyCommit {
		changes, result, err = client.Git.ListChanges(newctx, repo.Slug, build.After, scm.ListOptions{})
		if err != nil {
			return nil, err
		}
	} else {
		changes, result, err = client.Git.CompareChanges(newctx, repo.Slug, build.Before, build.After, scm.ListOptions{})
		if err != nil {
			return nil, err
		}
	}

	GithubApiCount.Set(float64(result.Rate.Remaining))

	var files []string
	for _, c := range changes {
		files = append(files, c.Path)
	}

	return files, nil
}

var (
	GithubApiCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "github_api_calls_remaining",
			Help: "Total number of github api calls per hour remaining",
		})
)

func init() {
	prometheus.MustRegister(GithubApiCount)
}

func prepareExtendedConfig(config string, token string) (string, error) {
	resources, err := unmarshal([]byte(config))
	if err != nil {
		return "", err
	}
	for _, resource := range resources {
		switch resource.Kind {
		case "monorepo":
			for _, c := range resource.Includes {
				// TODO replace
				println(c)
				// TODO Implement function
				// GetConfigFiles(c, repo, build, token)
			}
		}
	}

	return "", nil
}
