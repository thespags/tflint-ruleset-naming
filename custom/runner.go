package custom

import (
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"

	"github.com/thespags/tflint-ruleset-naming/config"
)

// Runner is a wrapper of RPC client with custom configuration.
type Runner struct {
	tflint.Runner

	// Resources stores the naming configuration keyed by resource type.
	Resources map[string]*Resource

	// Modules stores the naming configuration keyed by module source.
	Modules map[string]*Resource
}

// Resource is the parsed naming configuration for a resource or module.
type Resource struct {
	// NameFormat is the template for the expected block name.
	NameFormat string

	// FileFormat is the template for the expected file name (without .tf).
	FileFormat string
}

// NewRunner returns a new runner.
func NewRunner(runner tflint.Runner, customConfig *config.Config) (*Runner, error) {
	resources := map[string]*Resource{}

	for _, resource := range customConfig.Resources {
		resources[resource.Kind] = parseResource(resource)
	}

	modules := map[string]*Resource{}

	for _, module := range customConfig.Modules {
		modules[module.Kind] = parseResource(module)
	}

	return &Runner{
		Runner:    runner,
		Resources: resources,
		Modules:   modules,
	}, nil
}

func parseResource(resource *config.Resource) *Resource {
	fileFormat := resource.FileFormat
	if fileFormat == "" {
		fileFormat = resource.NameFormat
	}

	return &Resource{
		NameFormat: resource.NameFormat,
		FileFormat: fileFormat,
	}
}
