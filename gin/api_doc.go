package gin

import (
	"github.com/go-yaml/yaml"
)

type APIDocs struct {
	BaseURL   string     `yaml:"base_url"`
	Sections  []Section  `yaml:"sections"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

type Section struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Endpoint struct {
	Method      string  `yaml:"method"`
	Path        string  `yaml:"path"`
	Description string  `yaml:"description"`
	Section     string  `yaml:"section"`
	Params      []Param `yaml:"params"`
	Headers     []Param `yaml:"headers"`
}

type Param struct {
	Key         string   `yaml:"key"`
	Description string   `yaml:"description"`
	Type        string   `yaml:"type"`
	Default     string   `yaml:"default"`
	Required    bool     `yaml:"required"`
	Examples    []string `yaml:"examples"`
}

func ParseAPIDocs(doc []byte) (*APIDocs, error) {
	var docs APIDocs
	err := yaml.Unmarshal(doc, &docs)
	if err != nil {
		return nil, err
	}
	return &docs, nil
}
