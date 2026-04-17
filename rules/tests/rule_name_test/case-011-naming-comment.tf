module "engineers" {
  source = "app/example"
  name   = each.value.username // naming: engineers
}
