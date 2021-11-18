package plugin

import (
	filepath "github.com/bmatcuk/doublestar"

	"bytes"
	"io"

	"github.com/buildkite/yaml"
	"github.com/drone/drone-go/drone"
	"github.com/sirupsen/logrus"
)

// returns true if the string matches the include and
// doesn't match exclude patterns.
func (c *condition) match(v string) bool {
	if c.excludes(v) {
		return false
	}
	if c.includes(v) {
		return true
	}
	if len(c.Include) == 0 {
		return true
	}
	return false
}

func (c *condition) includes(v string) bool {
	for _, pattern := range c.Include {
		if ok, _ := filepath.Match(pattern, v); ok {
			return true
		}
	}
	return false
}

func (c *condition) excludes(v string) bool {
	for _, pattern := range c.Exclude {
		if ok, _ := filepath.Match(pattern, v); ok {
			return true
		}
	}
	return false
}

func pathSeen(data string) (bool, error) {
	pathSeen := false
	resources, err := unmarshal([]byte(data))
	if err != nil {
		return false, err
	}

	for _, resource := range resources {
		switch resource.Kind {
		case "pipeline":
			if len(append(resource.Trigger.Paths.Include, resource.Trigger.Paths.Exclude...)) > 0 {
				pathSeen = true
				break
			}
			for _, step := range resource.Steps {
				if step == nil {
					continue
				}
				if len(append(step.When.Paths.Include, step.When.Paths.Exclude...)) > 0 {
					pathSeen = true
					break
				}
			}
		}
	}
	return pathSeen, nil
}

func parsePipelines(data string, build drone.Build, repo drone.Repo, changedFiles []string) ([]*resource, error) {

	// set some default fields for logs
	requestLogger := logrus.WithFields(logrus.Fields{
		"build_after":    build.After,
		"build_before":   build.Before,
		"repo_namespace": repo.Namespace,
		"repo_name":      repo.Name,
	})

	resources, err := unmarshal([]byte(data))
	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		switch resource.Kind {
		case "pipeline":
			// there must be a better way to check whether paths.include or paths.exclude is set
			if len(append(resource.Trigger.Paths.Include, resource.Trigger.Paths.Exclude...)) > 0 {
				skipPipeline := true
				if len(changedFiles) > 0 {
					for _, p := range changedFiles {
						got, want := resource.Trigger.Paths.match(p), true
						if got == want {
							requestLogger.Infoln("including pipeline", resource.Attrs["name"])

							skipPipeline = false
							break
						}
					}
					// in case of a '--allow-empty' commit, don't skip the pipeline
				} else {
					skipPipeline = false
				}
				if skipPipeline {
					requestLogger.Infoln("excluding pipeline", resource.Attrs["name"])

					// if only Trigger.Paths is set, Trigger.Attrs will be unset, so it must be initialized
					if resource.Trigger.Attrs == nil {
						resource.Trigger.Attrs = make(map[string]interface{})
					}
					resource.Trigger.Attrs["event"] = map[string][]string{"exclude": []string{"*"}}
				}
			}

			for _, step := range resource.Steps {
				if step == nil {
					continue
				}
				// there must be a better way to check whether paths.include or paths.exclude is set
				if len(append(step.When.Paths.Include, step.When.Paths.Exclude...)) > 0 {
					skipStep := true
					if len(changedFiles) > 0 {
						for _, i := range changedFiles {
							got, want := step.When.Paths.match(i), true
							if got == want {
								requestLogger.Infoln("including step", step.Attrs["name"])

								skipStep = false
								break
							}
						}
						// in case of a '--allow-empty' commit, don't skip the step
					} else {
						skipStep = false
					}
					if skipStep {
						requestLogger.Infoln("excluding step", step.Attrs["name"])

						// if only When.Paths is set, When.Attrs will be unset, so it must be initialized
						if step.When.Attrs == nil {
							step.When.Attrs = make(map[string]interface{})
						}
						step.When.Attrs["event"] = map[string][]string{"exclude": []string{"*"}}
					}
				}
			}
		}
	}
	return resources, nil
}

// UnmarshalYAML supports implicit and optional include
func (c *condition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var out1 string
	var out2 []string
	var out3 = struct {
		Include []string
		Exclude []string
	}{}

	if err := unmarshal(&out1); err == nil {
		c.Include = []string{out1}
		return nil
	}

	_ = unmarshal(&out2)
	_ = unmarshal(&out3)

	c.Exclude = out3.Exclude
	c.Include = append(out3.Include, out2...)

	return nil
}

func unmarshal(b []byte) ([]*resource, error) {
	buf := bytes.NewBuffer(b)
	res := []*resource{}
	dec := yaml.NewDecoder(buf)
	for {
		out := new(resource)
		err := dec.Decode(out)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		res = append(res, out)
	}
	return res, nil
}

func marshal(in []*resource) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := yaml.NewEncoder(buf)
	for _, res := range in {
		err := enc.Encode(res)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
