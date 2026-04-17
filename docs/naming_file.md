# Rule `naming_file`

Enforces that files containing configured resources, data sources, or modules
are named according to the expected name derived from attribute values.

## Configuration

By default, `file_format` uses the same template as `name_format`. You can
override it:

```hcl
plugin "naming" {
  enabled = true

  module "app/example" {
    name_format = "{group_name}_{project_name}"
    file_format = "example_{group_name}_{project_name}"
  }
}
```

## Example

Given this configuration:

```hcl
module "app/example" {
  name_format = "{group_name}_{project_name}"
}
```

And this Terraform in a file called `main.tf`:

```hcl
module "my_group_my_project" {
  source       = "app/example"
  group_name   = "my-group"
  project_name = "my-project"
}
```

```text
Warning: file "main.tf" should be named "my_group_my_project.tf" (naming_file)

  on main.tf line 1:
   1: module "my_group_my_project" {
```

## No Fixer

This rule does not provide a fixer because TFLint cannot rename files. Rename
the file manually.

## Disabling

To disable file name checking while keeping name checking:

```hcl
rule "naming_file" {
  enabled = false
}
```
