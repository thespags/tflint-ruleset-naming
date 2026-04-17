module "should_skip" {
  source       = "app/example"
  group_name   = var.group
  project_name = "my-project"
}
