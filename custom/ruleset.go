package custom

import (
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"

	"github.com/thespags/tflint-ruleset-naming/config"
)

// RuleSet is the custom ruleset.
type RuleSet struct {
	tflint.BuiltinRuleSet

	config *config.Config
}

// ConfigSchema returns the ruleset plugin config schema.
func (r *RuleSet) ConfigSchema() *hclext.BodySchema {
	r.config = config.New()

	return hclext.ImpliedBodySchema(r.config)
}

// ApplyConfig applies the configuration to the ruleset.
func (r *RuleSet) ApplyConfig(body *hclext.BodyContent) error {
	diags := hclext.DecodeBody(body, nil, r.config)
	if diags.HasErrors() {
		return diags
	}

	return nil
}

// NewRunner creates a custom runner with the provided config.
func (r *RuleSet) NewRunner(runner tflint.Runner) (tflint.Runner, error) {
	return NewRunner(runner, r.config)
}
