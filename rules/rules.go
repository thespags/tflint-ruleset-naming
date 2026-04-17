package rules

import (
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// All returns all the rules in this ruleset.
func All() []tflint.Rule {
	return []tflint.Rule{
		NewNameRule(),
		NewFileRule(),
	}
}
