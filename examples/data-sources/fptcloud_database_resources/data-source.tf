data "fptcloud_database_resources" "vpc" {
  vpc_id = "ab463af6-1147-4446-92fc-3700df39ba05"
}
 
output "database_totals" {
  value = data.fptcloud_database_resources.vpc.total
}
 
output "proxies" {
  value = {
    for c in data.fptcloud_database_resources.vpc.clusters : c.cluster_id => c.proxy.items
    if length(c.proxy.items) > 0
  }
}
