![GitHub tag](https://img.shields.io/github/v/tag/thespags/tflint-ruleset-naming)
![Build](https://img.shields.io/github/actions/workflow/status/thespags/tflint-ruleset-naming/ci.yml)
![Go Version](https://img.shields.io/github/go-mod/go-version/thespags/tflint-ruleset-naming)
![License](https://img.shields.io/github/license/thespags/tflint-ruleset-naming)
[![Go Report Card](https://goreportcard.com/badge/github.com/thespags/tflint-ruleset-naming)](https://goreportcard.com/report/github.com/thespags/tflint-ruleset-naming)
[![codecov](https://codecov.io/gh/thespags/tflint-ruleset-sort/branch/main/graph/badge.svg)](https://codecov.io/gh/thespags/tflint-ruleset-sort)

# TFLint Ruleset Naming

TFLint ruleset plugin that enforces Terraform resource, data source, and module **names** (and optionally **file names**) based on attribute values within the block.

For example, a module with `group_name = "my-group"` and `project_name = "my-project"` can be required to be named `my_group_my_project`.

## Rules

| Rule | Description | Fixer |
|------|-------------|-------|
| [`naming_name`](docs/naming_name.md) | Block label must match a name derived from its attributes | Yes |
| [`naming_file`](docs/naming_file.md) | File must be named after the derived name | No |

## Installation

You can install the plugin with `tflint --init`. Declare a config in
`.tflint.hcl` as follows:

```hcl
plugin "naming" {
  enabled = true

  version = "0.0.1"
  source  = "github.com/thespags/tflint-ruleset-naming"
}
```

## Configuration

Define naming rules per module source or resource type. Data sources share the
same `resource` configuration. The `name_format` uses `{attribute}` interpolation
to build the expected name from field values.

### Module example

```hcl
plugin "naming" {
  enabled = true

  module "app/example" {
    name_format = "{group_name}_{project_name}"
  }
}
```

Given this Terraform:

```hcl
module "wrong_name" {
  source       = "app/example"
  group_name   = "my-group"
  project_name = "my-project"
}
```

The linter will report:

```text
Error: module "wrong_name" should be named "my_group_my_project" based on its attributes (naming_name)
```

And the fixer will rename the module label automatically.

### Resource example

```hcl
plugin "naming" {
  enabled = true

  resource "google_project" {
    name_format = "{project_id}"
  }
}
```

### Nested attributes

Dot notation is supported for attributes inside nested blocks:

```hcl
plugin "naming" {
  enabled = true

  resource "kubernetes_service_account" {
    name_format = "{metadata.namespace}_{metadata.name}"
  }
}
```

### File naming

By default, `file_format` inherits from `name_format`. The `naming_file` rule
checks that the `.tf` file is named `{resolved_file_format}.tf`.

To use a different file naming pattern:

```hcl
plugin "naming" {
  enabled = true

  module "app/example" {
    name_format = "{group_name}_{project_name}"
    file_format = "example_{group_name}_{project_name}"
  }
}
```

To disable file name checking, disable the `naming_file` rule in your `.tflint.hcl`:

```hcl
rule "naming_file" {
  enabled = false
}
```

## Format string syntax

Format strings use `{field}` interpolation:

- `{field_name}` — reads the attribute value from the block
- `{block.field}` — reads a nested attribute (traverses into a child block)
- `{?field_name}` — optional field, omitted if the attribute is missing or unresolvable
- Literal text between interpolations is preserved as-is

### Optional fields

Prefix a field with `?` to make it optional. If the attribute is missing or
unresolvable, the field is omitted and surrounding separators are cleaned up:

```hcl
module "./modules/group" {
  name_format = "{?parent_id}_{path}"
}
```

- With `parent_id = module.foo.id` and `path = "bar"` → `foo_bar`
- Without `parent_id` → `bar`

### Reference resolution

When an attribute is a reference rather than a string literal, the rule extracts
the meaningful identifier:

| Reference | Resolves to |
|-----------|-------------|
| `module.foo.id` | `foo` |
| `local.bar` | `bar` |
| `aws_instance.web.id` | `web` |

References to `var.*` and `each.*` cannot be statically resolved and will
produce an error (use `{?field}` to make them optional, or use a
`//naming:` comment).

### Naming comment override

When an attribute uses a dynamic expression (e.g. `each.value`), you can
provide an explicit value with a `//naming:` comment:

```hcl
module "foo_bar_engineers" {
  source    = "./modules/group"
  name      = each.value.username //naming:engineers
  parent_id = module.foo_bar.id
}
```

### Value sanitization

Interpolated field values are sanitized for use as Terraform identifiers:

1. Non-alphanumeric characters (`-`, `/`, `.`, etc.) are replaced with `_`
2. Consecutive underscores are collapsed
3. Leading and trailing underscores are trimmed
4. The result is lowercased

For example, `"my-group/sub"` becomes `my_group_sub`.

### Skipped checks

The rule is silently skipped when:

- The resource type or module source has no configuration
- Any required referenced attribute is missing from the block

The rule produces an error when:

- An attribute is an unresolvable expression (e.g. `var.*`, `each.*`) without
  a `//naming:` comment override and not marked optional with `{?}`

## Building the plugin

Clone the repository locally and run the following command:

With mise,
```bash
mise install
```

Build the plugin with:

```bash
go build ./...
```

You can install the built plugin with the following:

```bash
mise run install
```

You can run the built plugin like the following:

```bash
cat << EOF > .tflint.hcl
config {
  plugin_dir = "~/.tflint.d/plugins"
}

plugin "naming" {
  enabled = true
}
EOF

tflint
```
