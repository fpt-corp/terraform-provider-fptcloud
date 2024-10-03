resource "fptcloud_dedicated_kubernetes_engine_v1_state" "test_state" {
  id = "your-cluster-uuid"
  vpc_id = "your-vpc-id"
  is_running = true
}