package fptcloud_load_balancer_v2

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var dataSourceLoadBalancers = map[string]*schema.Schema{
	"vpc_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the VPC to list load balancers from",
	},
	"loadbalancers": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the load balancer",
				},
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the load balancer",
				},
				"description": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The description of the load balancer",
				},
				"operating_status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The operating status of the load balancer",
				},
				"provisioning_status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The provisioning status of the load balancer",
				},
				"public_ip": {
					Type:        schema.TypeList,
					Computed:    true,
					Description: "The public IP address of the load balancer",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"ip_address": {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
				},
				"private_ip": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The private IP address of the load balancer",
				},
				"network": {
					Type:        schema.TypeList,
					Computed:    true,
					Description: "The network of the load balancer",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"name": {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
				},
				"cidr": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The CIDR of the load balancer",
				},
				"size": {
					Type:        schema.TypeList,
					Computed:    true,
					Description: "The size of the load balancer",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"name": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"vip_amount": {
								Type:     schema.TypeInt,
								Computed: true,
							},
							"active_connection": {
								Type:     schema.TypeInt,
								Computed: true,
							},
							"application_throughput": {
								Type:     schema.TypeInt,
								Computed: true,
							},
						},
					},
				},
				"created_at": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The creation time of the load balancer",
				},
				"tags": {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The tags associated with the load balancer",
				},
				"egw_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the edge gateway associated with the load balancer",
				},
			},
		},
	},
}

var dataSourceLoadBalancer = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"load_balancer_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the load balancer",
	},
	"name": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The name of the load balancer",
	},
	"description": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The description of the load balancer",
	},
	"operating_status": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The operating status of the load balancer",
	},
	"provisioning_status": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The provisioning status of the load balancer",
	},
	"public_ip": {
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The public IP address of the load balancer",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"ip_address": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	},
	"private_ip": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The private IP address of the load balancer",
	},
	"network": {
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The network of the load balancer",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	},
	"cidr": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The CIDR of the load balancer",
	},
	"size": {
		Type:        schema.TypeList,
		Computed:    true,
		Description: "The size of the load balancer",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"vip_amount": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"active_connection": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"application_throughput": {
					Type:     schema.TypeInt,
					Computed: true,
				},
			},
		},
	},
	"created_at": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The creation time of the load balancer",
	},
	"tags": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Computed:    true,
		Description: "The tags associated with the load balancer",
	},
	"egw_id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The ID of the edge gateway associated with the load balancer",
	},
}

var resourceLoadBalancer = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"name": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The name of the load balancer",
	},
	"description": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The description of the load balancer",
	},
	"size": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The size ID of the load balancer",
	},
	"floating_ip": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The floating IP ID of the load balancer",
	},
	"network_id": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The network ID of the load balancer",
	},
	"vip_address": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The VIP address of the load balancer",
	},
	"cidr": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The CIDR of the load balancer",
	},
	"listener": {
		Type:        schema.TypeSet,
		Optional:    true,
		ForceNew:    true,
		Description: "The listener of the load balancer",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"description": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"protocol": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"protocol_port": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"certificate_id": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	},
	"pool": {
		Type:        schema.TypeSet,
		Optional:    true,
		ForceNew:    true,
		Description: "The default server pool of the load balancer",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"protocol": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"algorithm": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"persistence_type": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"persistence_cookie_name": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"pool_members": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"ip_address": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"network_id": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"protocol_port": {
								Type:     schema.TypeInt,
								Optional: true,
							},
							"weight": {
								Type:     schema.TypeInt,
								Optional: true,
							},
							"vm_id": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"is_external": {
								Type:     schema.TypeBool,
								Optional: true,
							},
						},
					},
				},
				"health_monitor": {
					Type:     schema.TypeSet,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"type": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"url_path": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"http_method": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"expected_codes": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"max_retries": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"max_retries_down": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"delay": {
								Type:     schema.TypeString,
								Optional: true,
							},
							"timeout": {
								Type:     schema.TypeString,
								Optional: true,
							},
						},
					},
				},
			},
		}},
	"egw_id": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The edge gateway ID of the load balancer",
	},
}

var dataSourceListeners = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"load_balancer_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the load balancer that this listener belongs to",
	},
	"listeners": {
		Type:     schema.TypeSet,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the listener",
				},
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the listener",
				},
				"description": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The description of the listener",
				},
				"provisioning_status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The provisioning status of the listener",
				},
				"protocol": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The protocol of the listener",
				},
				"port": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The port of the listener",
				},
				"insert_headers": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"x_forwarded_for": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"x_forwarded_port": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"x_forwarded_proto": {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
					Description: "The headers to insert into the listener",
				},
				"default_pool": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"name": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"protocol": {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
					Description: "The default pool of the listener",
				},
				"certificate": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"name": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"expired_at": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"created_at": {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
					Description: "The certificate of the listener",
				},
				"sni_certificates": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"name": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"expired_at": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"created_at": {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
					Description: "The SNI certificates of the listener",
				},
				"hsts_max_age": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The HSTS max age of the listener",
				},
				"hsts_include_subdomains": {
					Type:        schema.TypeBool,
					Computed:    true,
					Description: "Defines whether the include sub domains directive should be added to the Strict-Transport-Security HTTP response header",
				},
				"hsts_preload": {
					Type:        schema.TypeBool,
					Computed:    true,
					Description: "Defines whether the preload directive should be added to the Strict-Transport-Security HTTP response header.",
				},
				"connection_limit": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The connection limit of the listener",
				},
				"client_data_timeout": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The client data timeout of the listener",
				},
				"member_connect_timeout": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The member connect timeout of the listener",
				},
				"member_data_timeout": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The member data timeout of the listener",
				},
				"tcp_inspect_timeout": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The TCP inspect timeout of the listener",
				},
				"alpn_protocols": {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The ALPN protocols of the listener",
				},
				"created_at": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The creation timestamp of the listener",
				},
				"allowed_cidrs": {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The allowed CIDRs of the listener",
				},
				"tags": {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The tags associated with the listener",
				},
			},
		},
	},
}

var dataSourceListener = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"listener_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the istener",
	},
	"name": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The name of the listener",
	},
	"description": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The description of the listener",
	},
	"provisioning_status": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The provisioning status of the listener",
	},
	"protocol": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The protocol of the listener",
	},
	"port": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The port of the listener",
	},
	"insert_headers": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"x_forwarded_for": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"x_forwarded_port": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"x_forwarded_proto": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Description: "The headers to insert into the listener",
	},
	"default_pool": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"protocol": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Description: "The default pool of the listener",
	},
	"certificate": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"expired_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"created_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Description: "The certificate of the listener",
	},
	"sni_certificates": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"expired_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"created_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Description: "The SNI certificates of the listener",
	},
	"hsts_max_age": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The HSTS max age of the listener",
	},
	"hsts_include_subdomains": {
		Type:        schema.TypeBool,
		Computed:    true,
		Description: "Defines whether the include sub domains directive should be added to the Strict-Transport-Security HTTP response header",
	},
	"hsts_preload": {
		Type:        schema.TypeBool,
		Computed:    true,
		Description: "Defines whether the preload directive should be added to the Strict-Transport-Security HTTP response header.",
	},
	"connection_limit": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The connection limit of the listener",
	},
	"client_data_timeout": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The client data timeout of the listener",
	},
	"member_connect_timeout": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The member connect timeout of the listener",
	},
	"member_data_timeout": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The member data timeout of the listener",
	},
	"tcp_inspect_timeout": {
		Type:        schema.TypeInt,
		Computed:    true,
		Description: "The TCP inspect timeout of the listener",
	},
	"alpn_protocols": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Computed:    true,
		Description: "The ALPN protocols of the listener",
	},
	"created_at": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The creation timestamp of the listener",
	},
	"allowed_cidrs": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Computed:    true,
		Description: "The allowed CIDRs of the listener",
	},
	"tags": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Computed:    true,
		Description: "The tags associated with the listener",
	},
}

var resourceListener = map[string]*schema.Schema{
	"vpc_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the VPC",
	},
	"load_balancer_id": {
		Type:        schema.TypeString,
		Optional:    true,
		ForceNew:    true,
		Description: "The ID of the load balancer which owns the listener",
	},
	"name": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The name of the listener",
	},
	"description": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The description of the listener",
	},
	"protocol": {
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
		Description: "The protocol of the listener",
	},
	"protocol_port": {
		Type:        schema.TypeString,
		Required:    true,
		ForceNew:    true,
		Description: "The port of the listener",
	},
	"insert_headers": {
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"x_forwarded_for": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"x_forwarded_port": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"x_forwarded_proto": {
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
		Description: "The headers to insert into the listener",
	},
	"default_pool_id": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The default pool ID of the listener",
	},
	"certificate_id": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The certificate of the listener",
	},
	"sni_certificate_ids": {
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Description: "The SNI certificate IDs of the listener",
	},
	"hsts_max_age": {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The HSTS max age of the listener",
	},
	"hsts_include_subdomains": {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "Defines whether the include sub domains directive should be added to the Strict-Transport-Security HTTP response header",
	},
	"hsts_preload": {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "Defines whether the preload directive should be added to the Strict-Transport-Security HTTP response header.",
	},
	"connection_limit": {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The connection limit of the listener",
	},
	"client_data_timeout": {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The client data timeout of the listener",
	},
	"member_connect_timeout": {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The member connect timeout of the listener",
	},
	"member_data_timeout": {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The member data timeout of the listener",
	},
	"tcp_inspect_timeout": {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The TCP inspect timeout of the listener",
	},
	"alpn_protocols": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Optional:    true,
		Description: "The ALPN protocols of the listener",
	},
	"allowed_cidrs": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Optional:    true,
		Description: "The allowed CIDRs of the listener",
	},
}

var dataSourcePools = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"load_balancer_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"pools": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the pool",
				},
				"name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the pool",
				},
				"description": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The description of the pool",
				},
				"load_balancer_id": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the load balancer which owns the pool",
				},
				"operating_status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The operating status of the pool",
				},
				"provisioning_status": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The provisioning status of the pool",
				},
				"protocol": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The protocol of the pool",
				},
				"algorithm": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The algorithm of the pool",
				},
				"health_monitor": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"type": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"delay": {
								Type:     schema.TypeInt,
								Computed: true,
							},
							"timeout": {
								Type:     schema.TypeInt,
								Computed: true,
							},
							"max_retries": {
								Type:     schema.TypeInt,
								Computed: true,
							},
							"max_retries_down": {
								Type:     schema.TypeInt,
								Computed: true,
							},
							"http_method": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"url_path": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"expected_codes": {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
					Description: "The health monitor of the pool",
				},
				"members": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"vm_id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"vm_name": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"ip_address": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"network": {
								Type:     schema.TypeList,
								Computed: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"id": {
											Type:     schema.TypeString,
											Computed: true,
										},
										"name": {
											Type:     schema.TypeString,
											Computed: true,
										},
									},
								},
							},
							"port": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"weight": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"operating_status": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"provisioning_status": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"created_at": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"is_external": {
								Type:     schema.TypeBool,
								Computed: true,
							},
						},
					},
					Description: "The members of the pool",
				},
				"persistence_type": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The persistence type of the pool",
				},
				"persistence_cookie_name": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The name of the cookie used for persistence",
				},
				"alpn_protocols": {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The ALPN protocols of the pool",
				},
				"tls_enabled": {
					Type:        schema.TypeBool,
					Computed:    true,
					Description: "Whether TLS is enabled for the pool",
				},
				"created_at": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The creation time of the load balancer",
				},
				"tags": {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The tags associated with the load balancer",
				},
			},
		},
	},
}

var dataSourcePool = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"pool_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The ID of the pool",
	},
	"name": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The name of the pool",
	},
	"description": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The description of the pool",
	},
	"load_balancer_id": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The ID of the load balancer which owns the pool",
	},
	"operating_status": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The operating status of the pool",
	},
	"provisioning_status": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The provisioning status of the pool",
	},
	"protocol": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The protocol of the pool",
	},
	"algorithm": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The algorithm of the pool",
	},
	"health_monitor": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"delay": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"timeout": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"max_retries": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"max_retries_down": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"http_method": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"url_path": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"expected_codes": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Description: "The health monitor of the pool",
	},
	"members": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"vm_id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"vm_name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"ip_address": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"network": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"name": {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
				},
				"port": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"weight": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"operating_status": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"provisioning_status": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"created_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"is_external": {
					Type:     schema.TypeBool,
					Computed: true,
				},
			},
		},
		Description: "The members of the pool",
	},
	"persistence_type": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The persistence type of the pool",
	},
	"persistence_cookie_name": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The name of the cookie used for persistence",
	},
	"alpn_protocols": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Computed:    true,
		Description: "The ALPN protocols of the pool",
	},
	"tls_enabled": {
		Type:        schema.TypeBool,
		Computed:    true,
		Description: "Whether TLS is enabled for the pool",
	},
	"created_at": {
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The creation time of the load balancer",
	},
	"tags": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Computed:    true,
		Description: "The tags associated with the load balancer",
	},
}

var resourcePool = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"load_balancer_id": {
		Type:        schema.TypeString,
		Optional:    true,
		ForceNew:    true,
		Description: "The ID of the load balancer which owns the pool",
	},
	"name": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The name of the pool",
	},
	"description": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The description of the pool",
	},
	"protocol": {
		Type:        schema.TypeString,
		ForceNew:    true,
		Required:    true,
		Description: "The protocol of the pool",
	},
	"algorithm": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The algorithm of the pool",
	},
	"health_monitor": {
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"type": {
					Type:     schema.TypeString,
					Required: true,
				},
				"delay": {
					Type:     schema.TypeString,
					Required: true,
				},
				"timeout": {
					Type:     schema.TypeString,
					Required: true,
				},
				"max_retries": {
					Type:     schema.TypeString,
					Required: true,
				},
				"max_retries_down": {
					Type:     schema.TypeString,
					Required: true,
				},
				"http_method": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"url_path": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"expected_codes": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
		Description: "The health monitor of the pool",
	},
	"pool_members": {
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"vm_id": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"name": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"ip_address": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"network_id": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"protocol_port": {
					Type:     schema.TypeInt,
					Required: true,
				},
				"weight": {
					Type:     schema.TypeInt,
					Required: true,
				},
				"is_external": {
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
		Description: "The members of the pool",
	},
	"persistence_type": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The persistence type of the pool",
	},
	"persistence_cookie_name": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The name of the cookie used for persistence",
	},
	"alpn_protocols": {
		Type: schema.TypeList,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
		Optional:    true,
		Description: "The ALPN protocols of the pool",
	},
	"tls_enabled": {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "Whether TLS is enabled for the pool",
	},
}

var dataSourceCertificates = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"certificates": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"certificate_id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"created_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"expired_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	},
}

var dataSourceCertificate = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"certificate_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"name": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"created_at": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"expired_at": {
		Type:     schema.TypeString,
		Computed: true,
	},
}

var resourceCertificate = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		ForceNew: true,
		Required: true,
	},
	"name": {
		Type:     schema.TypeString,
		ForceNew: true,
		Required: true,
	},
	"certificate": {
		Type:     schema.TypeString,
		ForceNew: true,
		Optional: true,
	},
	"private_key": {
		Type:      schema.TypeString,
		ForceNew:  true,
		Optional:  true,
		Sensitive: true,
	},
	"cert_chain": {
		Type:     schema.TypeString,
		ForceNew: true,
		Optional: true,
	},
}

var dataSourceL7Policies = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"listener_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"l7policies": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"action": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"provisioning_status": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"redirect_pool": {
					Type:     schema.TypeList,
					Computed: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"id": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"name": {
								Type:     schema.TypeString,
								Computed: true,
							},
							"protocol": {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
				},
				"redirect_url": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"redirect_prefix": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"redirect_http_code": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"position": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"created_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"updated_at": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	},
}

var dataSourceL7Policy = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"listener_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"l7_policy_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"name": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"action": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"provisioning_status": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"redirect_pool": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"protocol": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	},
	"redirect_url": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"redirect_prefix": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"redirect_http_code": {
		Type:     schema.TypeInt,
		Computed: true,
	},
	"position": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"created_at": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"updated_at": {
		Type:     schema.TypeString,
		Computed: true,
	},
}

var resourceL7Policy = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"listener_id": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"name": {
		Type:     schema.TypeString,
		Required: true,
	},
	"action": {
		Type:     schema.TypeString,
		Required: true,
	},
	"redirect_pool": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"redirect_url": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"redirect_prefix": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"redirect_http_code": {
		Type:     schema.TypeInt,
		Optional: true,
	},
	"position": {
		Type:     schema.TypeInt,
		Required: true,
	},
}

var dataSourceL7Rules = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"listener_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"l7_policy_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"l7rules": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"type": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"compare_type": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"key": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"value": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"invert": {
					Type:     schema.TypeBool,
					Computed: true,
				},
				"operating_status": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"provisioning_status": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	},
}

var dataSourceL7Rule = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"listener_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"l7_policy_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"l7_rule_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"type": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"compare_type": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"key": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"value": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"invert": {
		Type:     schema.TypeBool,
		Computed: true,
	},
	"operating_status": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"provisioning_status": {
		Type:     schema.TypeString,
		Computed: true,
	},
}

var resourceL7Rule = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"listener_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"l7_policy_id": {
		Type:     schema.TypeString,
		Required: true,
		ForceNew: true,
	},
	"type": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"compare_type": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"key": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"value": {
		Type:     schema.TypeString,
		Optional: true,
	},
	"invert": {
		Type:     schema.TypeBool,
		Optional: true,
	},
}

var dataSourceSizes = map[string]*schema.Schema{
	"vpc_id": {
		Type:     schema.TypeString,
		Required: true,
	},
	"sizes": {
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"id": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"vip_amount": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"active_connection": {
					Type:        schema.TypeInt,
					Description: "The number of client connections processed concurrently",
					Computed:    true,
				},
				"application_throughput": {
					Type:        schema.TypeInt,
					Description: "The number of application requests handled per second",
					Computed:    true,
				},
			},
		},
	},
}
