package rules

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"

	"github.com/thespags/tflint-ruleset-naming/config"
	"github.com/thespags/tflint-ruleset-naming/custom"
)

func runTests(t *testing.T, rules ...tflint.Rule) {
	t.Helper()

	_, testFilename, _, ok := runtime.Caller(1)
	require.True(t, ok, "Could not get the caller of `runTests` from runtime")

	dir := path.Join(
		path.Dir(testFilename),
		"tests",
		strings.TrimSuffix(path.Base(testFilename), path.Ext(testFilename)),
	)

	err := filepath.Walk(dir, func(tfFilename string, tfInfo fs.FileInfo, _err error) error {
		if _err != nil {
			return _err
		}

		if tfInfo.IsDir() {
			return nil
		}

		ext := path.Ext(tfFilename)
		if ext != ".tf" {
			return nil
		}

		terraform := readTerraform(t, tfFilename)
		expectedFixes := readFixes(t, tfFilename)

		helperRunner := helper.TestRunner(
			t,
			map[string]string{tfFilename: terraform},
		)
		runner, err := custom.NewRunner(helperRunner, readConfig(t, tfFilename))
		require.NoError(t, err)

		t.Run(filepath.Base(tfFilename), func(t *testing.T) {
			t.Parallel()

			for _, rule := range rules {
				err = rule.Check(runner)
				require.NoError(t, err)
			}

			if len(rules) == 1 {
				expectedIssues := readIssues(t, rules[0], tfFilename)
				helper.AssertIssues(t, expectedIssues, helperRunner.Issues)
			}

			helper.AssertChanges(t, expectedFixes, helperRunner.Changes())
		})

		return nil
	})
	require.NoError(t, err)
}

func readConfig(t *testing.T, tfFilename string) *config.Config {
	t.Helper()

	configFilename := strings.TrimSuffix(tfFilename, ".tf") + ".config"

	content, err := os.ReadFile(configFilename)
	if errors.Is(err, os.ErrNotExist) {
		return config.New()
	}

	require.NoError(t, err)

	file, diags := hclsyntax.ParseConfig(content, configFilename, hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "Failed to parse config: %s", diags.Error())

	schema := hclext.ImpliedBodySchema(&config.Config{})
	body, diags := hclext.Content(file.Body, schema)
	require.False(t, diags.HasErrors(), "Failed to get config content: %s", diags.Error())

	cfg := config.New()
	diags = hclext.DecodeBody(body, nil, cfg)
	require.False(t, diags.HasErrors(), "Failed to decode config: %s", diags.Error())

	return cfg
}

func readTerraform(t *testing.T, tfFilename string) string {
	t.Helper()

	tfBytes, err := os.ReadFile(tfFilename)
	require.NoError(t, err)

	return string(tfBytes)
}

func readIssues(t *testing.T, rule tflint.Rule, tfFilename string) helper.Issues {
	t.Helper()

	content, err := os.ReadFile(strings.TrimSuffix(tfFilename, ".tf") + ".issues")
	if errors.Is(err, os.ErrNotExist) {
		return helper.Issues{}
	}

	require.NoError(t, err)

	issue := helper.Issues{}
	err = json.Unmarshal(content, &issue)
	require.NoError(t, err)

	for _, i := range issue {
		i.Rule = rule
		i.Range.Filename = tfFilename
	}

	return issue
}

func readFixes(t *testing.T, tfFilename string) map[string]string {
	t.Helper()

	content, err := os.ReadFile(strings.TrimSuffix(tfFilename, ".tf") + ".fixes")
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}
	}

	require.NoError(t, err)

	return map[string]string{
		tfFilename: string(content),
	}
}
