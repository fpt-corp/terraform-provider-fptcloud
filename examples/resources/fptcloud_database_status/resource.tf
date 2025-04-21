resource "fptcloud_database_status" "example_status" {
  id = fptcloud_database.example.id
  status = "running | stopped"
}
