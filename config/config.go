package config

// Config is the configuration for the ruleset.
type Config struct {
	Resources []*Resource `hclext:"resource,block"`
	Modules   []*Resource `hclext:"module,block"`
}

// Resource is the custom configuration of the resource-specific behavior.
type Resource struct {
	// Kind is the resource type (for resources) or module source (for modules).
	Kind string `hclext:"name,label"`

	// NameFormat is the template for the expected block name.
	// Uses {attribute} interpolation, e.g. "{group_name}_{project_name}".
	// Nested attributes use dot notation: "{metadata.name}".
	NameFormat string `hclext:"name_format"`

	// FileFormat is the template for the expected file name (without .tf extension).
	// Defaults to the same template as NameFormat if not specified.
	FileFormat string `hclext:"file_format,optional"`
}

// New creates a new configuration structure.
func New() *Config {
	return &Config{
		Resources: []*Resource{},
		Modules:   []*Resource{},
	}
}
