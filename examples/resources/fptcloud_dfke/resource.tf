resource "fptcloud_dedicated_kubernetes_engine_v1" "test" {
  cluster_name        = "terraform-test-3"
  k8s_version = "v1.25.6"
  #   master_type = data.fptcloud_flavor_v1.master.id
  master_type         = "c89d97cd-c9cb-4d70-a0c1-01f190ea1b02"
  master_count        = 1
  master_disk_size    = 76
  #   worker_type = data.fptcloud_flavor_v1.worker.id
  #   worker_type = "5ca3036e-85d6-497f-a37b-076aa8b9adde"
  worker_type         = "c89d97cd-c9cb-4d70-a0c1-01f190ea1b02"
  worker_disk_size = 103
  #   network_id = data.fptcloud_subnet_v1.xplat_network.id
  network_id          = "urn:vcloud:network:11980234-8474-4e2e-8925-8087177a43ca"
  lb_size             = "standard"
  pod_network         = "10.244.0.0/16"
  service_network     = "172.30.0.0/16"
  network_node_prefix = 23
  max_pod_per_node    = 110
  nfs_status          = ""
  nfs_disk_size       = 100
  storage_policy      = "Premium-SSD-4000"
  edge_id             = "4d4bfe05-af32-4354-b20a-de814c8b3713"
  scale_min           = 1
  scale_max           = 1
  node_dns            = "1.1.1.1"
  ip_public_firewall  = ""
  ip_private_firewall = ""
  vpc_id              = "188af427-269b-418a-90bb-0cb27afc6c1e"
  region_id           = "saigon-vn"
}