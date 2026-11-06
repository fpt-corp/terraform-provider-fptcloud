# Restart the proxy. Change `triggers` to restart it again.
resource "fptcloud_database_proxy_restart" "db" {
  proxy_id = fptcloud_database_proxy.db.id
  vpc_id   = var.vpc_id

  triggers = {
    run = "2026-11-05"
  }
}
