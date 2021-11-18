// Copyright 2019 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"context"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/converter"

	"github.com/sirupsen/logrus"
)

// New returns a new conversion plugin.
func New(token, provider string) converter.Plugin {
	return &plugin{
		token:    token,
		provider: provider,
	}
}

func (p *plugin) Convert(ctx context.Context, req *converter.Request) (*drone.Config, error) {
	// TODO Add relevant data and generate access log entries
	requestLogger := logrus.WithFields(logrus.Fields{
		"build_after":    req.Build.After,
		"build_before":   req.Build.Before,
		"repo_namespace": req.Repo.Namespace,
		"repo_name":      req.Repo.Name,
	})

	// get the configuration file from the request.
	config := req.Config.Data

	// prepare a new config file based on the plugin execution
	var newConfig string

	// check for any Paths.Include/Exclude fields in Trigger or Steps
	pathSeen, err := pathSeen(config)
	if err != nil {
		requestLogger.Errorln(err)
		return nil, err
	}

	if pathSeen {
		requestLogger.Infoln("selective path settings found")

		var changedFiles []string

		switch p.provider {
		case "github":
			changedFiles, err = GetGithubFilesChanged(req.Repo, req.Build, p.token)
			if err != nil {
				return nil, err
			}
		default:
			requestLogger.Errorln("skipping conversion, unsupported provider: ", p.provider)
			return nil, nil
		}

		extendedConfig, err := prepareAdditionalConfigs(config, req.Repo, req.Build, p.token)
		if err != nil {
			requestLogger.Errorln("could not retrieve extended configs, skipping: ", err)
			return nil, err
		}

		// merge all .drone.yml before processing paths changed
		config = config + "\n" + extendedConfig

		resources, err := parsePipelines(config, req.Build, req.Repo, changedFiles)
		if err != nil {
			requestLogger.Errorln(err)
			return nil, err
		}

		c, err := marshal(resources)
		if err != nil {
			return nil, err
		}
		newConfig = string(c)

	} else {
		requestLogger.Infoln("no paths fields seen")
		newConfig = config
	}

	return &drone.Config{
		Data: newConfig,
	}, nil

}
