resource "google_project" "my_project" {
  project_id = "my-project"
  name       = "My Project"
}

resource "google_project" "my_other" {
  project_id = "my-other"
  name       = "My Other"
}
