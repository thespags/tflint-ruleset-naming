package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseBody(t *testing.T, src []byte) *hclsyntax.Body {
	t.Helper()

	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), diags.Error())

	body, ok := file.Body.(*hclsyntax.Body)
	require.True(t, ok)

	return body
}

func TestSanitizeValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"my-group", "my_group"},
		{"foo/bar", "foo_bar"},
		{"baz.qux", "baz_qux"},
		{"already_valid", "already_valid"},
		{"UPPER-Case", "upper_case"},
		{"/leading-slash", "leading_slash"},
		{"trailing-slash/", "trailing_slash"},
		{"multi---dash", "multi_dash"},
		{"foo/bar/baz", "foo_bar_baz"},
		{"simple", "simple"},
		{"with spaces", "with_spaces"},
		{"a--b//c..d", "a_b_c_d"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, sanitizeValue(tt.input))
		})
	}
}

func TestResolveFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   string
		src      string
		expected string
		errMsg   string
	}{
		{
			name:     "simple literal",
			format:   "{name}",
			src:      `name = "foo"`,
			expected: "foo",
		},
		{
			name:     "multiple fields",
			format:   "{group}_{project}",
			src:      "group = \"my-group\"\nproject = \"my-project\"",
			expected: "my_group_my_project",
		},
		{
			name:     "special chars sanitized",
			format:   "{group}_{name}",
			src:      "group = \"foo/bar\"\nname = \"baz.qux\"",
			expected: "foo_bar_baz_qux",
		},
		{
			name:     "no interpolation",
			format:   "static_name",
			src:      `name = "anything"`,
			expected: "static_name",
		},
		{
			name:     "missing attribute returns empty",
			format:   "{name}",
			src:      `other = "foo"`,
			expected: "",
		},
		{
			name:     "optional field present",
			format:   "{?group}_{name}",
			src:      "group = \"eng\"\nname = \"app\"",
			expected: "eng_app",
		},
		{
			name:     "optional field missing",
			format:   "{?group}_{name}",
			src:      `name = "app"`,
			expected: "app",
		},
		{
			name:     "all optional fields missing",
			format:   "{?group}_{?team}",
			src:      `name = "irrelevant"`,
			expected: "",
		},
		{
			name:   "variable ref errors",
			format: "{name}",
			src:    `name = var.foo`,
			errMsg: "unresolvable reference",
		},
		{
			name:   "each ref errors",
			format: "{name}",
			src:    `name = each.value.field`,
			errMsg: "unresolvable reference",
		},
		{
			name:     "optional variable ref skipped",
			format:   "{?group}_{name}",
			src:      "group = var.foo\nname = \"app\"",
			expected: "app",
		},
		{
			name:     "module ref resolved",
			format:   "{name}",
			src:      `name = module.foo.id`,
			expected: "foo",
		},
		{
			name:     "local ref resolved",
			format:   "{name}",
			src:      `name = local.bar`,
			expected: "bar",
		},
		{
			name:     "naming comment override",
			format:   "{name}",
			src:      "name = each.value.username // naming: engineers",
			expected: "engineers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := []byte(tt.src)
			body := parseBody(t, src)
			result, err := resolveFormat(tt.format, body, src)

			if tt.errMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestResolveFieldValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    string
		src      string
		expected string
		errMsg   string
	}{
		{
			name:     "nested block found",
			field:    "metadata.name",
			src:      "metadata {\n  name = \"foo\"\n}",
			expected: "foo",
		},
		{
			name:     "nested block missing",
			field:    "metadata.name",
			src:      `name = "foo"`,
			expected: "",
		},
		{
			name:     "nested attribute missing",
			field:    "metadata.name",
			src:      "metadata {\n  other = \"foo\"\n}",
			expected: "",
		},
		{
			name:     "deeply nested block",
			field:    "spec.template.name",
			src:      "spec {\n  template {\n    name = \"bar\"\n  }\n}",
			expected: "bar",
		},
		{
			name:     "deeply nested block missing intermediate",
			field:    "spec.template.name",
			src:      "spec {\n  name = \"bar\"\n}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := []byte(tt.src)
			body := parseBody(t, src)
			result, err := resolveFieldValue(tt.field, body, src)

			if tt.errMsg != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestResolveTraversal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      string
		expected string
		ok       bool
	}{
		{
			name: "not a traversal",
			src:  `name = 1 + 2`,
			ok:   false,
		},
		{
			name: "var skipped",
			src:  `name = var.foo`,
			ok:   false,
		},
		{
			name: "each skipped",
			src:  `name = each.value.field`,
			ok:   false,
		},
		{
			name:     "module ref",
			src:      `name = module.foo.id`,
			expected: "foo",
			ok:       true,
		},
		{
			name:     "local ref",
			src:      `name = local.bar`,
			expected: "bar",
			ok:       true,
		},
		{
			name:     "data ref",
			src:      `name = data.aws_ami.latest.id`,
			expected: "aws_ami.latest",
			ok:       true,
		},
		{
			name:     "resource ref",
			src:      `name = aws_instance.web.id`,
			expected: "web",
			ok:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := []byte(tt.src)
			body := parseBody(t, src)
			attr := body.Attributes["name"]
			require.NotNil(t, attr)

			result, ok := resolveTraversal(attr.Expr)
			assert.Equal(t, tt.ok, ok)

			if tt.ok {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      string
		expected string
	}{
		{
			name:     "source present",
			src:      "module \"foo\" {\n  source = \"app/example\"\n}",
			expected: "app/example",
		},
		{
			name:     "source missing",
			src:      "module \"foo\" {\n  name = \"bar\"\n}",
			expected: "",
		},
		{
			name:     "source is not a string",
			src:      "module \"foo\" {\n  source = 42\n}",
			expected: "",
		},
		{
			name:     "source is a variable ref",
			src:      "module \"foo\" {\n  source = var.src\n}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := []byte(tt.src)
			body := parseBody(t, src)
			require.Len(t, body.Blocks, 1)

			assert.Equal(t, tt.expected, getSource(body.Blocks[0]))
		})
	}
}

func TestSourceLine(t *testing.T) {
	t.Parallel()

	src := []byte("first\nsecond\nthird")

	tests := []struct {
		name     string
		line     int
		expected string
	}{
		{"line 1", 1, "first"},
		{"line 2", 2, "second"},
		{"line 3", 3, "third"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, sourceLine(src, tt.line))
		})
	}
}
