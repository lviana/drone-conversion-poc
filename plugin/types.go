package plugin

type plugin struct {
	token    string
	provider string
}

type resource struct {
	Kind     string
	Type     string
	Steps    []*step                `yaml:"steps,omitempty"`
	Trigger  conditions             `yaml:"trigger,omitempty"`
	Attrs    map[string]interface{} `yaml:",inline"`
	Includes []string               `yaml:"includes,omitempty"`
}

type step struct {
	When  conditions             `yaml:"when,omitempty"`
	Attrs map[string]interface{} `yaml:",inline"`
}

type conditions struct {
	Paths condition              `yaml:"paths,omitempty"`
	Attrs map[string]interface{} `yaml:",inline"`
}

type condition struct {
	Exclude []string `yaml:"exclude,omitempty"`
	Include []string `yaml:"include,omitempty"`
}
