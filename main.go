package main

import (
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"

	"github.com/thespags/tflint-ruleset-naming/custom"
	"github.com/thespags/tflint-ruleset-naming/project"
	"github.com/thespags/tflint-ruleset-naming/rules"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &custom.RuleSet{
			BuiltinRuleSet: tflint.BuiltinRuleSet{
				Name:    project.Name,
				Version: project.Version,
				Rules:   rules.All(),
			},
		},
	})
}
