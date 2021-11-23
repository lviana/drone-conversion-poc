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

func GetConfigFile(filepath string, repo drone.Repo, build drone.Build, token string) ([]*resource, error) {
	newctx := context.Background()
	client := github.NewDefault()
	client.Client = &http.Client{
		Transport: &transport.BearerToken{
			Token: token,
		},
	}

	var result *scm.Response
	var err error

	content, _, err := client.Contents.Find(newctx, repo.Slug, filepath, build.After)
	if err != nil {
		return nil, err
	}

	GithubApiCount.Set(float64(result.Rate.Remaining))

	// fmt.Println(content.Path, content.Data)
	resources, err := unmarshal([]byte(content.Data))
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func prepareAdditionalConfigs(config string, repo drone.Repo, build drone.Build, token string) (string, error) {
	resources, err := unmarshal([]byte(config))
	if err != nil {
		return "", err
	}

	var appendedConfig []*resource

	for _, resource := range resources {
		switch resource.Kind {
		case "monorepo":
			for _, f := range resource.Projects {
				// TODO Validate execution
				documents, err := GetConfigFile(f, repo, build, token)
				if err != nil {
					return "", err
				}
				for _, document := range documents {
					appendedConfig = append(appendedConfig, document)
				}
			}
		}
	}

	output, _ := marshal(appendedConfig)
	return string(output), nil
}
