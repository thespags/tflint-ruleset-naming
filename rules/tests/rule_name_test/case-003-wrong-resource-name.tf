resource "google_project" "bad_name" {
  project_id = "my-project"
  name       = "My Project"
}
