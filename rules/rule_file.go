package rules

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"

	"github.com/thespags/tflint-ruleset-naming/custom"
	"github.com/thespags/tflint-ruleset-naming/project"
	"github.com/thespags/tflint-ruleset-naming/visit"
)

// FileRule enforces that files containing configured resources or modules
// are named according to the expected name derived from attribute values.
type FileRule struct {
	tflint.DefaultRule
}

// NewFileRule creates a new FileRule.
func NewFileRule() *FileRule {
	return &FileRule{}
}

// Name returns the name of the rule.
func (*FileRule) Name() string {
	return project.RuleName("file")
}

// Enabled returns whether the rule is enabled by default.
func (*FileRule) Enabled() bool {
	return true
}

// Severity returns the severity of the rule.
func (*FileRule) Severity() tflint.Severity {
	return tflint.WARNING
}

// Link returns the reference link for the rule.
func (r *FileRule) Link() string {
	return project.ReferenceLink(r.Name())
}

// Check verifies that files are named to match the configured naming format.
func (r *FileRule) Check(rr tflint.Runner) error {
	runner, ok := rr.(*custom.Runner)
	if !ok {
		return nil
	}

	return visit.Blocks(runner, func(filename string, block *hclsyntax.Block, src []byte) error {
		switch block.Type {
		case "resource", "data":
			return r.checkResource(runner, filename, block, src)
		case "module":
			return r.checkModule(runner, filename, block, src)
		default:
			return nil
		}
	})
}

func (r *FileRule) checkResource(runner *custom.Runner, filename string, block *hclsyntax.Block, src []byte) error {
	kind := block.Labels[0]

	resource, known := runner.Resources[kind]
	if !known {
		return nil
	}

	return r.checkFile(runner, filename, resource.FileFormat, block, src)
}

func (r *FileRule) checkModule(runner *custom.Runner, filename string, block *hclsyntax.Block, src []byte) error {
	sourceStr := getSource(block)

	module, known := runner.Modules[sourceStr]
	if !known {
		return nil
	}

	return r.checkFile(runner, filename, module.FileFormat, block, src)
}

func (r *FileRule) checkFile(
	runner *custom.Runner,
	filename string,
	fileFormat string,
	block *hclsyntax.Block,
	src []byte,
) error {
	expectedBase, err := resolveFormat(fileFormat, block.Body, src)
	if err != nil {
		return runner.EmitIssue(r, err.Error(), block.DefRange())
	}

	if expectedBase == "" {
		return nil
	}

	expectedFilename := expectedBase + ".tf"
	actualBase := filepath.Base(filename)
	// Strip any leading directory path and compare just the filename.
	actualName := strings.TrimSuffix(actualBase, ".tf")

	if actualName == expectedBase {
		return nil
	}

	return runner.EmitIssue(
		r,
		fmt.Sprintf(
			"file %q should be named %q",
			actualBase,
			expectedFilename,
		),
		block.DefRange(),
	)
}
