# Rule `naming_name`

Enforces that resource, data source, and module labels match the expected name
derived from configured attribute values.

## Configuration
```hcl
plugin "naming" {
  enabled = true

  module "app/example" {
    name_format = "{group_name}_{project_name}"
  }

  resource "google_project" {
    name_format = "{project_id}"
  }
}
```

## Module Example

```hcl
module "wrong_name" {
  source       = "app/example"
  group_name   = "my-group"
  project_name = "my-project"
}
```

```text
Error: module "wrong_name" should be named "my_group_my_project" based on its attributes (naming_name)

  on template.tf line 1:
   1: module "wrong_name" {
```

The fixer will rename the label to `"my_group_my_project"`.

## Resource Example

```hcl
resource "google_project" "bad_name" {
  project_id = "my-project"
  name       = "My Project"
}
```

```text
Error: resource "bad_name" should be named "my_project" based on its attributes (naming_name)

  on template.tf line 1:
   1: resource "google_project" "bad_name" {
```

## Nested Attributes

Dot notation traverses into child blocks:

```hcl
plugin "naming" {
  resource "kubernetes_service_account" {
    name_format = "{metadata.namespace}_{metadata.name}"
  }
}
```

## Data Source Example

Data sources are configured using the same `resource` block as regular resources:

```hcl
plugin "naming" {
  resource "aws_ami" {
    name_format = "{name}"
  }
}
```

```hcl
data "aws_ami" "wrong_name" {
  name = "my-ami"
}
```

```text
Error: data "wrong_name" should be named "my_ami" based on its attributes (naming_name)

  on template.tf line 1:
   1: data "aws_ami" "wrong_name" {
```

## Reference Resolution

When an attribute is a reference rather than a string literal, the rule extracts
the meaningful identifier:

| Reference | Resolves to |
|-----------|-------------|
| `module.foo.id` | `foo` |
| `local.bar` | `bar` |
| `aws_instance.web.id` | `web` |

References to `var.*` and `each.*` cannot be statically resolved and will
produce an error. Use `{?field}` to make them optional, or add a
`//naming:` comment to provide an explicit value.

## Skipped Cases

The check is silently skipped when:

- The resource type or module source is not configured
- Any required referenced attribute is missing from the block

The check produces an error when:

- An attribute is an unresolvable expression (e.g. `var.*`, `each.*`) without
  a `//naming:` comment override and not marked optional with `{?}`
