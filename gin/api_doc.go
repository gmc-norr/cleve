package gin

import (
	"html/template"

	"github.com/go-yaml/yaml"
)

type APIDocs struct {
	BaseURL   string     `yaml:"base_url"`
	Sections  []Section  `yaml:"sections"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

type Section struct {
	Name        string        `yaml:"name"`
	Description template.HTML `yaml:"description"`
}

type Endpoint struct {
	Method      string        `yaml:"method"`
	Path        string        `yaml:"path"`
	Description template.HTML `yaml:"description"`
	Section     string        `yaml:"section"`
	Params      []Param       `yaml:"params"`
	QueryParams []QueryParam  `yaml:"query_params"`
	Headers     []Param       `yaml:"headers"`
}

type Param struct {
	Key         string        `yaml:"key"`
	Description template.HTML `yaml:"description"`
	Type        string        `yaml:"type"`
	Default     string        `yaml:"default"`
	Required    bool          `yaml:"required"`
	Examples    []string      `yaml:"examples"`
}

type QueryParam struct {
	Key         string        `yaml:"key"`
	Description template.HTML `yaml:"description"`
	Type        string        `yaml:"type"`
	Multiple    bool          `yaml:"multiple"`
}

func ParseAPIDocs(doc []byte) (*APIDocs, error) {
	var docs APIDocs
	err := yaml.Unmarshal(doc, &docs)
	if err != nil {
		return nil, err
	}
	return &docs, nil
}
