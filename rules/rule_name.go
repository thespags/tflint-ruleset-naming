package rules

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"

	"github.com/thespags/tflint-ruleset-naming/custom"
	"github.com/thespags/tflint-ruleset-naming/project"
	"github.com/thespags/tflint-ruleset-naming/visit"
)

// NameRule enforces that resource and module names match the expected name
// derived from configured attribute values.
type NameRule struct {
	tflint.DefaultRule
}

// NewNameRule creates a new NameRule.
func NewNameRule() *NameRule {
	return &NameRule{}
}

// Name returns the name of the rule.
func (*NameRule) Name() string {
	return project.RuleName("name")
}

// Enabled returns whether the rule is enabled by default.
func (*NameRule) Enabled() bool {
	return true
}

// Severity returns the severity of the rule.
func (*NameRule) Severity() tflint.Severity {
	return tflint.ERROR
}

// Link returns the reference link for the rule.
func (r *NameRule) Link() string {
	return project.ReferenceLink(r.Name())
}

// Check verifies that resource/module names match configured name formats.
func (r *NameRule) Check(rr tflint.Runner) error {
	runner, ok := rr.(*custom.Runner)
	if !ok {
		return nil
	}

	return visit.Blocks(runner, func(_ string, block *hclsyntax.Block, src []byte) error {
		switch block.Type {
		case "resource", "data":
			return r.checkResource(runner, block, src)
		case "module":
			return r.checkModule(runner, block, src)
		default:
			return nil
		}
	})
}

func (r *NameRule) checkResource(runner *custom.Runner, block *hclsyntax.Block, src []byte) error {
	resource, known := runner.Resources[block.Labels[0]]
	if !known {
		return nil
	}

	return r.checkName(runner, block, block.Labels[1], block.LabelRanges[1], resource.NameFormat, src)
}

func (r *NameRule) checkModule(runner *custom.Runner, block *hclsyntax.Block, src []byte) error {
	module, known := runner.Modules[getSource(block)]
	if !known {
		return nil
	}

	return r.checkName(runner, block, block.Labels[0], block.LabelRanges[0], module.NameFormat, src)
}

func (r *NameRule) checkName(
	runner *custom.Runner,
	block *hclsyntax.Block,
	actualName string,
	labelRange hcl.Range,
	nameFormat string,
	src []byte,
) error {
	expectedName, err := resolveFormat(nameFormat, block.Body, src)
	if err != nil {
		return runner.EmitIssue(r, err.Error(), block.DefRange())
	}

	if expectedName == "" || actualName == expectedName {
		return nil
	}

	return runner.EmitIssueWithFix(
		r,
		fmt.Sprintf(
			"%s %q should be named %q based on its attributes",
			block.Type,
			actualName,
			expectedName,
		),
		labelRange,
		func(fixer tflint.Fixer) error {
			return fixer.ReplaceText(labelRange, fmt.Sprintf("%q", expectedName))
		},
	)
}
