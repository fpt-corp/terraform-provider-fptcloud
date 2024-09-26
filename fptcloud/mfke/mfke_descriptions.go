package fptcloud_mfke

var descriptions = map[string]string{
	"vpc_id":             "VPC ID",
	"cluster_name":       "Cluster name",
	"k8s_version":        "Kubernetes version",
	"purpose":            "Cluster purpose",
	"pod_network":        "Pod network (subnet ID)",
	"pod_prefix":         "Pod network (prefix)",
	"service_network":    "Service network (subnet ID)",
	"service_prefix":     "Service prefix (prefix)",
	"range_ip_lb_start":  "IP start for range of LB",
	"range_ip_lb_end":    "IP stop for range of LB",
	"load_balancer_type": "Load balancer type",

	"k8s_max_pod":         "Max pods per node",
	"network_node_prefix": "Network node prefix",

	"name":                  "Pool name",
	"storage_profile":       "Pool storage profile",
	"worker_type":           "Worker flavor ID",
	"network_name":          "Subnet name",
	"network_id":            "Subnet ID",
	"worker_disk_size":      "Worker disk size",
	"scale_min":             "Minimum number of nodes for autoscaling",
	"scale_max":             "Maximum number of nodes for autoscaling",
	"auto_scale":            "Whether to enable autoscaling",
	"is_enable_auto_repair": "Whether to enable auto-repair",
}
