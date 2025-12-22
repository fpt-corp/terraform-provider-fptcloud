package fptcloud_load_balancer_v2_test

import (
	common "terraform-provider-fptcloud/commons"
	fptcloud_load_balancer_v2 "terraform-provider-fptcloud/fptcloud/load_balancer_v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListLoadBalancerSizesSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get load balancer size successfully",
		"data": [
			{
				"id": "3acbe070-5364-4fa6-81c2-1c04167ca9da",
				"name": "Basic-1",
				"vip_amount": 1,
				"active_connection": 1000,
				"application_throughput": 5000
			},
			{
				"id": "4ce5ff27-58e9-4c0a-8978-4caec289450f",
				"name": "Basic-2",
				"vip_amount": 1,
				"active_connection": 2000,
				"application_throughput": 10000
			}
		],
		"total": 2
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/sizes": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	response, err := service.ListSizes(vpcId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 2, response.Total)
	assert.Equal(t, "Get load balancer size successfully", response.Message)

	assert.Equal(t, "3acbe070-5364-4fa6-81c2-1c04167ca9da", response.Sizes[0].Id)
	assert.Equal(t, "Basic-1", response.Sizes[0].Name)
	assert.EqualValues(t, 1, response.Sizes[0].VipAmount)
	assert.EqualValues(t, 1000, response.Sizes[0].ActiveConnection)
	assert.EqualValues(t, 5000, response.Sizes[0].ApplicationThroughput)

	assert.Equal(t, "4ce5ff27-58e9-4c0a-8978-4caec289450f", response.Sizes[1].Id)
	assert.Equal(t, "Basic-2", response.Sizes[1].Name)
	assert.EqualValues(t, 1, response.Sizes[1].VipAmount)
	assert.EqualValues(t, 2000, response.Sizes[1].ActiveConnection)
	assert.EqualValues(t, 10000, response.Sizes[1].ApplicationThroughput)
}

func TestListLoadBalancersSuccessfully(t *testing.T) {
	mockResponse := `{
		"data": [
			{
				"id": "bzesb4e5-2752-4a78-bdd2-e67053a5x99d",
				"name": "aaaaaaa",
				"description": "",
				"operating_status": "Healthy",
				"provisioning_status": "Active",
				"public_ip": null,
				"private_ip": "169.10.2.59",
				"network": null,
				"egw_name": "R1",
				"cidr": null,
				"size": {
					"id": "8458791c-5ad0-4ccd-87a2-ca8c4f9f61d8",
					"name": "Basic-1",
					"vip_amount": 1,
					"active_connection": 1000,
					"application_throughput": 5000
				},
				"created_at": "2025-10-03T18:43:51",
				"tags": null
			}
		],
		"message": "Get load balancers successfully",
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/list?page=1&page_size=1000": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	response, err := service.ListLoadBalancers(vpcId, 1, 1000)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, response.Total)
	assert.Equal(t, "bzesb4e5-2752-4a78-bdd2-e67053a5x99d", response.LoadBalancers[0].Id)
	assert.Equal(t, "aaaaaaa", response.LoadBalancers[0].Name)
	assert.Equal(t, "Healthy", response.LoadBalancers[0].OperatingStatus)
	assert.Equal(t, "Active", response.LoadBalancers[0].ProvisioningStatus)
	assert.Equal(t, "169.10.2.59", response.LoadBalancers[0].PrivateIp)
	assert.Equal(t, "R1", response.LoadBalancers[0].EgwName)
	assert.Equal(t, "8458791c-5ad0-4ccd-87a2-ca8c4f9f61d8", response.LoadBalancers[0].Size.Id)
	assert.Equal(t, "Basic-1", response.LoadBalancers[0].Size.Name)
	assert.EqualValues(t, 1, response.LoadBalancers[0].Size.VipAmount)
	assert.EqualValues(t, 1000, response.LoadBalancers[0].Size.ActiveConnection)
	assert.EqualValues(t, 5000, response.LoadBalancers[0].Size.ApplicationThroughput)
	assert.Equal(t, "2025-10-03T18:43:51", response.LoadBalancers[0].CreatedAt)
	assert.Nil(t, response.LoadBalancers[0].Tags)
}

func TestGetLoadBalancerSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get load balancer successfully",
		"data": {
			"id": "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e",
			"name": "loadbalancer",
			"description": "",
			"operating_status": "Healthy",
			"provisioning_status": "Active",
			"public_ip": null,
			"private_ip": "192.168.20.108",
			"network": {
				"id": "df34979c-3d7b-4ae9-a84f-272da5339fe0",
				"name": "hehe-net"
			},
			"cidr": null,
			"size": {
				"id": "3acbe070-5364-4fa6-81c2-1c04167ca9da",
				"name": "Basic-1",
				"vip_amount": 1,
				"active_connection": 1000,
				"application_throughput": 5000
			},
			"created_at": "2025-10-13T06:28:29",
			"tags": [
				"LBv2"
			],
			"egw_name": "R1"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/load_balancer_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	loadBalancerId := "load_balancer_id"
	response, err := service.GetLoadBalancer(vpcId, loadBalancerId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Get load balancer successfully", response.Message)
	assert.Equal(t, "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e", response.LoadBalancer.Id)
	assert.Equal(t, "loadbalancer", response.LoadBalancer.Name)
	assert.Equal(t, "", response.LoadBalancer.Description)
	assert.Equal(t, "Healthy", response.LoadBalancer.OperatingStatus)
	assert.Equal(t, "Active", response.LoadBalancer.ProvisioningStatus)
	assert.Equal(t, "192.168.20.108", response.LoadBalancer.PrivateIp)
	assert.Equal(t, "df34979c-3d7b-4ae9-a84f-272da5339fe0", response.LoadBalancer.Network.Id)
	assert.Equal(t, "hehe-net", response.LoadBalancer.Network.Name)
	assert.Equal(t, "", response.LoadBalancer.Cidr)
	assert.Equal(t, "3acbe070-5364-4fa6-81c2-1c04167ca9da", response.LoadBalancer.Size.Id)
	assert.Equal(t, "Basic-1", response.LoadBalancer.Size.Name)
	assert.EqualValues(t, 1, response.LoadBalancer.Size.VipAmount)
	assert.EqualValues(t, 1000, response.LoadBalancer.Size.ActiveConnection)
	assert.EqualValues(t, 5000, response.LoadBalancer.Size.ApplicationThroughput)
	assert.Equal(t, "2025-10-13T06:28:29", response.LoadBalancer.CreatedAt)
	assert.Equal(t, "LBv2", response.LoadBalancer.Tags[0])
	assert.Equal(t, "R1", response.LoadBalancer.EgwName)
}

func TestCreateLoadBalancerSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Create load balancer successfully",
		"data": {
			"id": "ba0356e8-3dac-4ecb-b6d3-3581d40b580d",
			"id_on_platform": "5be1e96e-9073-4443-9832-1c823577dc52",
			"vpc_id": "dbaab857-1932-4779-88f9-e410c8b54f9a",
			"size_id": "3acbe070-5364-4fa6-81c2-1c04167ca9da",
			"network_id": "df34979c-3d7b-4ae9-a84f-272da5339fe0",
			"cidr": null,
			"egw_id": "a75aaf55-2720-4079-ad62-8dd716f32f3a",
			"ip_address_id": null,
			"virtual_ip_address": "192.168.20.143",
			"name": "loadbalancer2",
			"description": "",
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"status_message": null,
			"created_at": "2025-10-14T03:43:54",
			"updated_at": null,
			"is_deleted": false,
			"tags": []
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/create": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	createRequest := fptcloud_load_balancer_v2.LoadBalancerCreateModel{
		Name:      "loadbalancer2",
		Size:      "Basic-1",
		NetworkId: "df34979c-3d7b-4ae9-a84f-272da5339fe0",
		EgwId:     "a75aaf55-2720-4079-ad62-8dd716f32f3a",
		Listener: fptcloud_load_balancer_v2.DefaultListener{
			Name:         "Default listener",
			Protocol:     "HTTP",
			ProtocolPort: "80",
		},
		Pool: fptcloud_load_balancer_v2.InputDefaultServerPool{
			Name:        "Default server pool",
			Algorithm:   "ROUND_ROBIN",
			Protocol:    "HTTP",
			PoolMembers: nil,
			HealthMonitor: fptcloud_load_balancer_v2.InputHealthMonitor{
				Type:           "HTTP",
				Delay:          "5",
				MaxRetries:     "3",
				MaxRetriesDown: "3",
				Timeout:        "5",
				HttpMethod:     "GET",
				UrlPath:        "/",
				ExpectedCodes:  "200",
			},
		},
	}
	response, err := service.CreateLoadBalancer(vpcId, createRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Create load balancer successfully", response.Message)
	assert.Equal(t, "ba0356e8-3dac-4ecb-b6d3-3581d40b580d", response.Data.Id)
	assert.Equal(t, "5be1e96e-9073-4443-9832-1c823577dc52", response.Data.IdOnPlatform)
	assert.Equal(t, "dbaab857-1932-4779-88f9-e410c8b54f9a", response.Data.VpcId)
	assert.Equal(t, "3acbe070-5364-4fa6-81c2-1c04167ca9da", response.Data.SizeId)
	assert.Equal(t, "df34979c-3d7b-4ae9-a84f-272da5339fe0", response.Data.NetworkId)
	assert.Equal(t, "a75aaf55-2720-4079-ad62-8dd716f32f3a", response.Data.EgwId)
	assert.Equal(t, "192.168.20.143", response.Data.VirtualIpAddress)
	assert.Equal(t, "loadbalancer2", response.Data.Name)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, "2025-10-14T03:43:54", response.Data.CreatedAt)
	assert.False(t, response.Data.IsDeleted)
	assert.Empty(t, response.Data.Tags)
}

func TestUpdateLoadBalancerSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Update load balancer successfully",
		"data": {
			"id": "ba0356e8-3dac-4ecb-b6d3-3581d40b580d",
			"id_on_platform": "5be1e96e-9073-4443-9832-1c823577dc52",
			"vpc_id": "dbaab857-1932-4779-88f9-e410c8b54f9a",
			"size_id": "3acbe070-5364-4fa6-81c2-1c04167ca9da",
			"network_id": "df34979c-3d7b-4ae9-a84f-272da5339fe0",
			"cidr": null,
			"egw_id": "a75aaf55-2720-4079-ad62-8dd716f32f3a",
			"ip_address_id": null,
			"virtual_ip_address": "192.168.20.143",
			"name": "loadbalancer_updated",
			"description": "loadbalancer_updated",
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"status_message": null,
			"created_at": "2025-10-14T03:43:54",
			"updated_at": null,
			"is_deleted": false,
			"tags": []
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/load_balancer_id/update": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	loadBalancerId := "load_balancer_id"
	updateRequest := fptcloud_load_balancer_v2.LoadBalancerUpdateModel{
		Name:        "loadbalancer_updated",
		Description: "loadbalancer_updated",
	}
	response, err := service.UpdateLoadBalancer(vpcId, loadBalancerId, updateRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Update load balancer successfully", response.Message)
	assert.Equal(t, "ba0356e8-3dac-4ecb-b6d3-3581d40b580d", response.Data.Id)
	assert.Equal(t, "5be1e96e-9073-4443-9832-1c823577dc52", response.Data.IdOnPlatform)
	assert.Equal(t, "dbaab857-1932-4779-88f9-e410c8b54f9a", response.Data.VpcId)
	assert.Equal(t, "3acbe070-5364-4fa6-81c2-1c04167ca9da", response.Data.SizeId)
	assert.Equal(t, "df34979c-3d7b-4ae9-a84f-272da5339fe0", response.Data.NetworkId)
	assert.Equal(t, "a75aaf55-2720-4079-ad62-8dd716f32f3a", response.Data.EgwId)
	assert.Equal(t, "192.168.20.143", response.Data.VirtualIpAddress)
	assert.Equal(t, "loadbalancer_updated", response.Data.Name)
	assert.Equal(t, "loadbalancer_updated", response.Data.Description)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, "2025-10-14T03:43:54", response.Data.CreatedAt)
	assert.False(t, response.Data.IsDeleted)
	assert.Empty(t, response.Data.Tags)
}

func TestResizeLoadBalancerSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Resize load balancer successfully",
		"data": {
			"id": "ba0356e8-3dac-4ecb-b6d3-3581d40b580d",
			"id_on_platform": "5be1e96e-9073-4443-9832-1c823577dc52",
			"vpc_id": "dbaab857-1932-4779-88f9-e410c8b54f9a",
			"size_id": "3acbe070-5364-4fa6-81c2-1c04167ca9da",
			"network_id": "df34979c-3d7b-4ae9-a84f-272da5339fe0",
			"cidr": null,
			"egw_id": "a75aaf55-2720-4079-ad62-8dd716f32f3a",
			"ip_address_id": null,
			"virtual_ip_address": "192.168.20.143",
			"name": "loadbalancer",
			"description": "loadbalancer",
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"status_message": null,
			"created_at": "2025-10-14T03:43:54",
			"updated_at": null,
			"is_deleted": false,
			"tags": []
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/load_balancer_id/resize": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	loadBalancerId := "load_balancer_id"
	resizeRequest := fptcloud_load_balancer_v2.LoadBalancerResizeModel{
		NewSize: "Basic-2",
	}
	response, err := service.ResizeLoadBalancer(vpcId, loadBalancerId, resizeRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Resize load balancer successfully", response.Message)
	assert.Equal(t, "ba0356e8-3dac-4ecb-b6d3-3581d40b580d", response.Data.Id)
	assert.Equal(t, "5be1e96e-9073-4443-9832-1c823577dc52", response.Data.IdOnPlatform)
	assert.Equal(t, "dbaab857-1932-4779-88f9-e410c8b54f9a", response.Data.VpcId)
	assert.Equal(t, "3acbe070-5364-4fa6-81c2-1c04167ca9da", response.Data.SizeId)
	assert.Equal(t, "df34979c-3d7b-4ae9-a84f-272da5339fe0", response.Data.NetworkId)
	assert.Equal(t, "a75aaf55-2720-4079-ad62-8dd716f32f3a", response.Data.EgwId)
	assert.Equal(t, "192.168.20.143", response.Data.VirtualIpAddress)
	assert.Equal(t, "loadbalancer", response.Data.Name)
	assert.Equal(t, "loadbalancer", response.Data.Description)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, "2025-10-14T03:43:54", response.Data.CreatedAt)
	assert.False(t, response.Data.IsDeleted)
	assert.Empty(t, response.Data.Tags)
}

func TestDeleteLoadBalancerSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Delete load balancer successfully",
		"data": {}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/load_balancer_id/delete": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	loadBalancerId := "load_balancer_id"
	response, err := service.DeleteLoadBalancer(vpcId, loadBalancerId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Delete load balancer successfully", response.Message)
}

func TestListListenersSuccessfully(t *testing.T) {
	mockResponse := `{
		"data": [
			{
				"id": "cadbea0a-c33e-41c9-81a7-ac78f5e2ae52",
				"name": "Default listener",
				"description": "",
				"provisioning_status": "Active",
				"protocol": "HTTP",
				"port": "80",
				"load_balancer_id": "d37cf32d-3c01-42b5-b817-e84ca36f1aee",
				"insert_headers": {
					"X-Forwarded-For": "False",
					"X-Forwarded-Port": "False",
					"X-Forwarded-Proto": "False"
				},
				"certificate": null,
				"sni_certificates": [],
				"hsts_max_age": null,
				"hsts_include_subdomains": false,
				"hsts_preload": false,
				"connection_limit": -1,
				"client_data_timeout": 50000,
				"member_connect_timeout": 5000,
				"member_data_timeout": 50000,
				"tcp_inspect_timeout": 0,
				"alpn_protocols": null,
				"created_at": "2025-09-23T08:16:54",
				"allowed_cidrs": ["192.168.1.0/24"],
				"denied_cidrs": ["192.168.2.0/24"]
			}
		],
		"message": "Get listeners successfully",
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/load_balancer_id/listeners/list?page=1&page_size=1000": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	loadBalancerId := "load_balancer_id"
	response, err := service.ListListeners(vpcId, loadBalancerId, 1, 1000)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, response.Total)
	assert.Equal(t, "cadbea0a-c33e-41c9-81a7-ac78f5e2ae52", response.Listeners[0].Id)
	assert.Equal(t, "Default listener", response.Listeners[0].Name)
	assert.Equal(t, "Active", response.Listeners[0].ProvisioningStatus)
	assert.Equal(t, "HTTP", response.Listeners[0].Protocol)
	assert.Equal(t, "80", response.Listeners[0].Port)
	assert.Equal(t, "d37cf32d-3c01-42b5-b817-e84ca36f1aee", response.Listeners[0].LoadBalancerId)
	assert.Equal(t, "False", response.Listeners[0].InsertHeaders.XForwardedFor)
	assert.Equal(t, "False", response.Listeners[0].InsertHeaders.XForwardedPort)
	assert.Equal(t, "False", response.Listeners[0].InsertHeaders.XForwardedProto)
	assert.Equal(t, "", response.Listeners[0].Certificate.Id)
	assert.Equal(t, "", response.Listeners[0].Certificate.Name)
	assert.Equal(t, "", response.Listeners[0].Certificate.ExpiredAt)
	assert.Equal(t, "", response.Listeners[0].Certificate.CreatedAt)
	assert.Empty(t, response.Listeners[0].SniCertificates)
	assert.Equal(t, 0, response.Listeners[0].HstsMaxAge)
	assert.False(t, false, response.Listeners[0].HstsIncludeSubdomains)
	assert.False(t, false, response.Listeners[0].HstsPreload)
	assert.Equal(t, -1, response.Listeners[0].ConnectionLimit)
	assert.Equal(t, 50000, response.Listeners[0].ClientDataTimeout)
	assert.Equal(t, 5000, response.Listeners[0].MemberConnectTimeout)
	assert.Equal(t, 50000, response.Listeners[0].MemberDataTimeout)
	assert.Equal(t, 0, response.Listeners[0].TcpInspectTimeout)
	assert.Nil(t, response.Listeners[0].AlpnProtocols)
	assert.Equal(t, "2025-09-23T08:16:54", response.Listeners[0].CreatedAt)
	assert.Equal(t, []string{"192.168.1.0/24"}, response.Listeners[0].AllowedCidrs)
	assert.Equal(t, []string{"192.168.2.0/24"}, response.Listeners[0].DeniedCidrs)
}

func TestGetListenerSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get listener successfully",
		"data": {
			"id": "2d5bcd2f-c712-4276-9b8a-b1331966cfb4",
			"name": "Default listener",
			"description": "",
			"provisioning_status": "Active",
			"protocol": "HTTP",
			"port": "80",
			"load_balancer_id": "5da3bb98-e882-4556-af64-ccceeb48069d",
			"insert_headers": {
				"X-Forwarded-For": "False",
				"X-Forwarded-Port": "False",
				"X-Forwarded-Proto": "False"
			},
			"default_pool": {
				"id": "0bfed8b5-5d39-41e8-84f6-1c29f63d0614",
				"name": "Default server pool",
				"protocol": "HTTP"
			},
			"certificate": null,
			"sni_certificates": [],
			"hsts_max_age": null,
			"hsts_include_subdomains": false,
			"hsts_preload": false,
			"connection_limit": -1,
			"client_data_timeout": 50000,
			"member_connect_timeout": 5000,
			"member_data_timeout": 50000,
			"tcp_inspect_timeout": 0,
			"alpn_protocols": null,
			"created_at": "2025-09-16T04:15:40",
			"allowed_cidrs": ["192.168.1.0/24"],
			"denied_cidrs": ["192.168.2.0/24"]
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	response, err := service.GetListener(vpcId, listenerId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Get listener successfully", response.Message)
	assert.Equal(t, "2d5bcd2f-c712-4276-9b8a-b1331966cfb4", response.Listener.Id)
	assert.Equal(t, "Default listener", response.Listener.Name)
	assert.Equal(t, "", response.Listener.Description)
	assert.Equal(t, "Active", response.Listener.ProvisioningStatus)
	assert.Equal(t, "HTTP", response.Listener.Protocol)
	assert.Equal(t, "80", response.Listener.Port)
	assert.Equal(t, "5da3bb98-e882-4556-af64-ccceeb48069d", response.Listener.LoadBalancerId)
	assert.Equal(t, "False", response.Listener.InsertHeaders.XForwardedFor)
	assert.Equal(t, "False", response.Listener.InsertHeaders.XForwardedPort)
	assert.Equal(t, "False", response.Listener.InsertHeaders.XForwardedProto)
	assert.Equal(t, "0bfed8b5-5d39-41e8-84f6-1c29f63d0614", response.Listener.DefaultPool.Id)
	assert.Equal(t, "Default server pool", response.Listener.DefaultPool.Name)
	assert.Equal(t, "HTTP", response.Listener.DefaultPool.Protocol)
	assert.Equal(t, "", response.Listener.Certificate.Id)
	assert.Equal(t, "", response.Listener.Certificate.Name)
	assert.Equal(t, "", response.Listener.Certificate.ExpiredAt)
	assert.Equal(t, "", response.Listener.Certificate.CreatedAt)
	assert.Empty(t, response.Listener.SniCertificates)
	assert.Equal(t, 0, response.Listener.HstsMaxAge)
	assert.False(t, false, response.Listener.HstsIncludeSubdomains)
	assert.False(t, false, response.Listener.HstsPreload)
	assert.Equal(t, -1, response.Listener.ConnectionLimit)
	assert.Equal(t, 50000, response.Listener.ClientDataTimeout)
	assert.Equal(t, 5000, response.Listener.MemberConnectTimeout)
	assert.Equal(t, 50000, response.Listener.MemberDataTimeout)
	assert.Equal(t, 0, response.Listener.TcpInspectTimeout)
	assert.Nil(t, response.Listener.AlpnProtocols)
	assert.Equal(t, "2025-09-16T04:15:40", response.Listener.CreatedAt)
	assert.Equal(t, []string{"192.168.1.0/24"}, response.Listener.AllowedCidrs)
	assert.Equal(t, []string{"192.168.2.0/24"}, response.Listener.DeniedCidrs)
}

func TestCreateListenerSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Create listener successfully",
		"data": {
			"id": "9800aa98-df5b-4061-aa0d-f4236c27f9a2",
			"load_balancer_id": "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e",
			"name": "listener",
			"description": "",
			"certificate_id": null,
			"sni_certificate_ids": [],
			"default_pool_id": null,
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"protocol": "HTTP",
			"port": 82,
			"insert_headers": {
				"X-Forwarded-For": "False",
				"X-Forwarded-Port": "False",
				"X-Forwarded-Proto": "False"
			},
			"hsts_max_age": null,
			"hsts_include_subdomains": false,
			"hsts_preload": false,
			"connection_limit": -1,
			"client_data_timeout": 50000,
			"member_connect_timeout": 5000,
			"member_data_timeout": 50000,
			"tcp_inspect_timeout": 0,
			"allowed_cidrs": ["192.168.1.0/24"],
			"denied_cidrs": ["192.168.2.0/24"],
			"alpn_protocols": null,
			"created_at": "2025-10-14T03:59:06",
			"updated_at": null,
			"tags": []
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/load_balancer_id/listeners/create": mockResponse,
	})
	defer server.Close()

	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	loadBalancerId := "load_balancer_id"
	createRequest := fptcloud_load_balancer_v2.ListenerCreateModel{
		Name:                 "listener",
		Description:          "",
		Protocol:             "HTTP",
		ProtocolPort:         "82",
		DefaultPoolId:        "",
		CertificateId:        "",
		SniCertificateIds:    []string{},
		ConnectionLimit:      -1,
		ClientDataTimeout:    50000,
		MemberConnectTimeout: 5000,
		MemberDataTimeout:    50000,
		TcpInspectTimeout:    0,
		InsertHeaders: map[string]bool{
			"X-Forwarded-For":   false,
			"X-Forwarded-Port":  false,
			"X-Forwarded-Proto": false,
		},
		HstsMaxAge:            0,
		HstsIncludeSubdomains: false,
		HstsPreload:           false,
		AllowedCidrs:          []string{"192.168.1.0/24"},
		DeniedCidrs:           []string{"192.168.2.0/24"},
		AlpnProtocols:         []string{},
	}

	response, err := service.CreateListener(vpcId, loadBalancerId, createRequest)

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Create listener successfully", response.Message)
	assert.Equal(t, "9800aa98-df5b-4061-aa0d-f4236c27f9a2", response.Data.Id)
	assert.Equal(t, "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e", response.Data.LoadBalancerId)
	assert.Equal(t, "listener", response.Data.Name)
	assert.Equal(t, "", response.Data.Description)
	assert.Equal(t, "", response.Data.CertificateId)
	assert.Empty(t, response.Data.SniCertificateIds)
	assert.Equal(t, "", response.Data.DefaultPoolId)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, "HTTP", response.Data.Protocol)
	assert.Equal(t, 82, response.Data.Port)
	assert.Equal(t, map[string]string{
		"X-Forwarded-For":   "False",
		"X-Forwarded-Port":  "False",
		"X-Forwarded-Proto": "False",
	}, response.Data.InsertHeaders)
	assert.EqualValues(t, 0, response.Data.HstsMaxAge)
	assert.False(t, response.Data.HstsIncludeSubdomains)
	assert.False(t, response.Data.HstsPreload)
	assert.Equal(t, -1, response.Data.ConnectionLimit)
	assert.Equal(t, 50000, response.Data.ClientDataTimeout)
	assert.Equal(t, 5000, response.Data.MemberConnectionTimeout)
	assert.Equal(t, 50000, response.Data.MemberDataTimeout)
	assert.Equal(t, 0, response.Data.TcpInspectTimeout)
	assert.Equal(t, []string{"192.168.1.0/24"}, response.Data.AllowedCidrs)
	assert.Equal(t, []string{"192.168.2.0/24"}, response.Data.DeniedCidrs)
	assert.Empty(t, response.Data.AlpnProtocols)
	assert.Equal(t, "2025-10-14T03:59:06", response.Data.CreatedAt)
	assert.Equal(t, "", response.Data.UpdatedAt)
	assert.Empty(t, response.Data.Tags)
}

func TestUpdateListenerSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Update listener successfully",
		"data": {
			"id": "9800aa98-df5b-4061-aa0d-f4236c27f9a2",
			"load_balancer_id": "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e",
			"name": "listener_updated",
			"description": "listener_updated",
			"certificate_id": null,
			"sni_certificate_ids": [],
			"default_pool_id": null,
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"protocol": "HTTP",
			"port": 82,
			"insert_headers": {
				"X-Forwarded-For": "False",
				"X-Forwarded-Port": "False",
				"X-Forwarded-Proto": "False"
			},
			"hsts_max_age": null,
			"hsts_include_subdomains": false,
			"hsts_preload": false,
			"connection_limit": -1,
			"client_data_timeout": 50000,
			"member_connect_timeout": 5000,
			"member_data_timeout": 50000,
			"tcp_inspect_timeout": 0,
			"allowed_cidrs": ["192.168.1.0/24"],
			"denied_cidrs": ["192.168.2.0/24"],
			"alpn_protocols": null,
			"created_at": "2025-10-14T03:59:06",
			"updated_at": null,
			"tags": []
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/update": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	updateRequest := fptcloud_load_balancer_v2.ListenerUpdateModel{
		Name:                 "listener_updated",
		Description:          "listener_updated",
		DefaultPoolId:        "",
		CertificateId:        "",
		SniCertificateIds:    []string{},
		ConnectionLimit:      -1,
		ClientDataTimeout:    50000,
		MemberConnectTimeout: 5000,
		MemberDataTimeout:    50000,
		TcpInspectTimeout:    0,
		InsertHeaders: map[string]bool{
			"X-Forwarded-For":   false,
			"X-Forwarded-Port":  false,
			"X-Forwarded-Proto": false,
		},
		HstsMaxAge:            0,
		HstsIncludeSubdomains: false,
		HstsPreload:           false,
		AllowedCidrs:          []string{"192.168.1.0/24"},
		DeniedCidrs:           []string{"192.168.2.0/24"},
		AlpnProtocols:         []string{},
	}

	response, err := service.UpdateListener(vpcId, listenerId, updateRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Update listener successfully", response.Message)
	assert.Equal(t, "9800aa98-df5b-4061-aa0d-f4236c27f9a2", response.Data.Id)
	assert.Equal(t, "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e", response.Data.LoadBalancerId)
	assert.Equal(t, "listener_updated", response.Data.Name)
	assert.Equal(t, "listener_updated", response.Data.Description)
	assert.Equal(t, "", response.Data.CertificateId)
	assert.Empty(t, response.Data.SniCertificateIds)
	assert.Equal(t, "", response.Data.DefaultPoolId)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, "HTTP", response.Data.Protocol)
	assert.Equal(t, 82, response.Data.Port)
	assert.Equal(t, map[string]string{
		"X-Forwarded-For":   "False",
		"X-Forwarded-Port":  "False",
		"X-Forwarded-Proto": "False",
	}, response.Data.InsertHeaders)
	assert.EqualValues(t, 0, response.Data.HstsMaxAge)
	assert.False(t, response.Data.HstsIncludeSubdomains)
	assert.False(t, response.Data.HstsPreload)
	assert.Equal(t, -1, response.Data.ConnectionLimit)
	assert.Equal(t, 50000, response.Data.ClientDataTimeout)
	assert.Equal(t, 5000, response.Data.MemberConnectionTimeout)
	assert.Equal(t, 50000, response.Data.MemberDataTimeout)
	assert.Equal(t, 0, response.Data.TcpInspectTimeout)
	assert.Equal(t, []string{"192.168.1.0/24"}, response.Data.AllowedCidrs)
	assert.Equal(t, []string{"192.168.2.0/24"}, response.Data.DeniedCidrs)
	assert.Empty(t, response.Data.AlpnProtocols)
	assert.Equal(t, "2025-10-14T03:59:06", response.Data.CreatedAt)
	assert.Equal(t, "", response.Data.UpdatedAt)
	assert.Empty(t, response.Data.Tags)
}

func TestDeleteListenerSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Delete listener successfully",
		"data": {}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/delete": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	response, err := service.DeleteListener(vpcId, listenerId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Delete listener successfully", response.Message)
}

func TestListPoolsSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get pools successfully",
		"data": [
			{
				"id": "1f6d8c2c-ca7b-4ca4-ae62-c0b9033fbf8d",
				"name": "Default server pool",
				"description": "",
				"load_balancer_id": "a2e8b4e5-2752-4678-bdd2-e67053a5999d",
				"operating_status": "Healthy",
				"provisioning_status": "Active",
				"protocol": "HTTP",
				"algorithm": "ROUND_ROBIN",
				"health_monitor": {
					"type": "HTTP",
					"delay": 5,
					"timeout": 5,
					"max_retries": 3,
					"max_retries_down": 3,
					"http_method": "GET",
					"url_path": "/",
					"expected_codes": "200"
				},
				"members": [],
				"persistence_type": null,
				"persistence_cookie_name": null,
				"alpn_protocols": null,
				"tls_enabled": false,
				"created_at": "2025-10-03T18:43:51"
			}
		],
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/load_balancer_id/pools/list?page=1&page_size=1000": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	loadBalancerId := "load_balancer_id"
	response, err := service.ListPools(vpcId, loadBalancerId, 1, 1000)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, response.Total)
	assert.Equal(t, "1f6d8c2c-ca7b-4ca4-ae62-c0b9033fbf8d", response.Pools[0].Id)
	assert.Equal(t, "Default server pool", response.Pools[0].Name)
	assert.Equal(t, "", response.Pools[0].Description)
	assert.Equal(t, "a2e8b4e5-2752-4678-bdd2-e67053a5999d", response.Pools[0].LoadBalancerId)
	assert.Equal(t, "Healthy", response.Pools[0].OperatingStatus)
	assert.Equal(t, "Active", response.Pools[0].ProvisioningStatus)
	assert.Equal(t, "HTTP", response.Pools[0].Protocol)
	assert.Equal(t, "ROUND_ROBIN", response.Pools[0].Algorithm)
	assert.Equal(t, "HTTP", response.Pools[0].HealthMonitor.Type)
	assert.Equal(t, 5, response.Pools[0].HealthMonitor.Delay)
	assert.Equal(t, 5, response.Pools[0].HealthMonitor.Timeout)
	assert.Equal(t, 3, response.Pools[0].HealthMonitor.MaxRetries)
	assert.Equal(t, 3, response.Pools[0].HealthMonitor.MaxRetriesDown)
	assert.Equal(t, "GET", response.Pools[0].HealthMonitor.HttpMethod)
	assert.Equal(t, "/", response.Pools[0].HealthMonitor.UrlPath)
	assert.Equal(t, "200", response.Pools[0].HealthMonitor.ExpectedCodes)
	assert.Empty(t, response.Pools[0].Members)
	assert.Equal(t, "", response.Pools[0].PersistenceType)
	assert.Equal(t, "", response.Pools[0].PersistenceCookieName)
	assert.Nil(t, response.Pools[0].AlpnProtocols)
	assert.False(t, response.Pools[0].TlsEnabled)
	assert.Equal(t, "2025-10-03T18:43:51", response.Pools[0].CreatedAt)
}

func TestGetPoolSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get server pool successfully",
		"data": {
			"id": "0bfed8b5-5d39-41e8-84f6-1c29f63d0614",
			"name": "Default server pool",
			"description": "",
			"load_balancer_id": "5da3bb98-e882-4556-af64-ccceeb48069d",
			"operating_status": "Healthy",
			"provisioning_status": "Active",
			"protocol": "HTTP",
			"algorithm": "ROUND_ROBIN",
			"health_monitor": {
				"type": "HTTP",
				"delay": 5,
				"timeout": 5,
				"max_retries": 3,
				"max_retries_down": 3,
				"http_method": "GET",
				"url_path": "/",
				"expected_codes": "200"
			},
			"members": [],
			"persistence_type": null,
			"persistence_cookie_name": null,
			"alpn_protocols": null,
			"tls_enabled": false,
			"created_at": "2025-09-16T04:15:40"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/pools/pool_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	poolId := "pool_id"
	response, err := service.GetPool(vpcId, poolId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Get server pool successfully", response.Message)
	assert.Equal(t, "0bfed8b5-5d39-41e8-84f6-1c29f63d0614", response.Pool.Id)
	assert.Equal(t, "Default server pool", response.Pool.Name)
	assert.Equal(t, "", response.Pool.Description)
	assert.Equal(t, "5da3bb98-e882-4556-af64-ccceeb48069d", response.Pool.LoadBalancerId)
	assert.Equal(t, "Healthy", response.Pool.OperatingStatus)
	assert.Equal(t, "Active", response.Pool.ProvisioningStatus)
	assert.Equal(t, "HTTP", response.Pool.Protocol)
	assert.Equal(t, "ROUND_ROBIN", response.Pool.Algorithm)
	assert.Equal(t, "HTTP", response.Pool.HealthMonitor.Type)
	assert.Equal(t, 5, response.Pool.HealthMonitor.Delay)
	assert.Equal(t, 5, response.Pool.HealthMonitor.Timeout)
	assert.Equal(t, 3, response.Pool.HealthMonitor.MaxRetries)
	assert.Equal(t, 3, response.Pool.HealthMonitor.MaxRetriesDown)
	assert.Equal(t, "GET", response.Pool.HealthMonitor.HttpMethod)
	assert.Equal(t, "/", response.Pool.HealthMonitor.UrlPath)
	assert.Equal(t, "200", response.Pool.HealthMonitor.ExpectedCodes)
	assert.Empty(t, response.Pool.Members)
	assert.Equal(t, "", response.Pool.PersistenceType)
	assert.Equal(t, "", response.Pool.PersistenceCookieName)
	assert.Nil(t, response.Pool.AlpnProtocols)
	assert.False(t, response.Pool.TlsEnabled)
	assert.Equal(t, "2025-09-16T04:15:40", response.Pool.CreatedAt)
}

func TestCreatePoolSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Create pool successfully",
		"data": {
			"id": "d8a43897-f7cf-4c8c-8b0c-42c76f7e2d55",
			"load_balancer_id": "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e",
			"name": "Default server pool",
			"description": "",
			"protocol": "HTTP",
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"algorithm": "ROUND_ROBIN",
			"persistence_type": "",
			"persistence_cookie_name": "",
			"alpn_protocols": [],
			"tls_enabled": false,
			"created_at": "2025-10-14T04:05:10",
			"updated_at": null,
			"tags": [],
			"health_monitor": {
				"id": "6d5df57e-897f-4b51-a0a4-4029b4cb747b",
				"pool_id": "d8a43897-f7cf-4c8c-8b0c-42c76f7e2d55",
				"protocol": "HTTP",
				"http_method": "GET",
				"expected_codes": "200",
				"url_path": "/",
				"max_retries": 3,
				"max_retries_down": 3,
				"delay": 5,
				"timeout": 5
			},
			"members": []
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/load_balancer_id/pools/create": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	loadBalancerId := "load_balancer_id"
	createRequest := fptcloud_load_balancer_v2.PoolCreateModel{
		Name:                  "Default server pool",
		Description:           "",
		Algorithm:             "ROUND_ROBIN",
		Protocol:              "HTTP",
		PersistenceType:       "",
		PersistenceCookieName: "",
		PoolMembers:           []fptcloud_load_balancer_v2.InputPoolMember{},
		HealthMonitor: fptcloud_load_balancer_v2.InputHealthMonitor{
			Type:           "HTTP",
			Delay:          "5",
			MaxRetries:     "3",
			MaxRetriesDown: "3",
			Timeout:        "5",
			HttpMethod:     "GET",
			UrlPath:        "/",
			ExpectedCodes:  "200",
		},
		AlpnProtocols: []string{},
		TlsEnabled:    false,
	}
	response, err := service.CreatePool(vpcId, loadBalancerId, createRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Create pool successfully", response.Message)
	assert.Equal(t, "d8a43897-f7cf-4c8c-8b0c-42c76f7e2d55", response.Data.Id)
	assert.Equal(t, "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e", response.Data.LoadBalancerId)
	assert.Equal(t, "Default server pool", response.Data.Name)
	assert.Equal(t, "HTTP", response.Data.Protocol)
	assert.Equal(t, "ROUND_ROBIN", response.Data.Algorith)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, "6d5df57e-897f-4b51-a0a4-4029b4cb747b", response.Data.HealthMonitor.Id)
	assert.Equal(t, "HTTP", response.Data.HealthMonitor.Protocol)
	assert.Equal(t, "GET", response.Data.HealthMonitor.HttpMethod)
	assert.Equal(t, "200", response.Data.HealthMonitor.ExpectedCodes)
	assert.Equal(t, "/", response.Data.HealthMonitor.UrlPath)
	assert.Equal(t, 3, response.Data.HealthMonitor.MaxRetries)
	assert.Equal(t, 3, response.Data.HealthMonitor.MaxRetriesDown)
	assert.Equal(t, 5, response.Data.HealthMonitor.Delay)
	assert.Equal(t, 5, response.Data.HealthMonitor.Timeout)
	assert.Empty(t, response.Data.Members)
}

func TestUpdatePoolSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Update pool successfully",
		"data": {
			"id": "d8a43897-f7cf-4c8c-8b0c-42c76f7e2d55",
			"load_balancer_id": "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e",
			"name": "pool_updated",
			"description": "pool_updated",
			"protocol": "HTTP",
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"algorithm": "ROUND_ROBIN",
			"persistence_type": "",
			"persistence_cookie_name": "",
			"alpn_protocols": [],
			"tls_enabled": false,
			"created_at": "2025-10-14T04:05:10",
			"updated_at": null,
			"tags": [],
			"health_monitor": {
				"id": "6d5df57e-897f-4b51-a0a4-4029b4cb747b",
				"pool_id": "d8a43897-f7cf-4c8c-8b0c-42c76f7e2d55",
				"protocol": "HTTP",
				"http_method": "GET",
				"expected_codes": "200",
				"url_path": "/",
				"max_retries": 3,
				"max_retries_down": 3,
				"delay": 5,
				"timeout": 5
			},
			"members": []
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/pools/pool_id/update": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	poolId := "pool_id"
	updateRequest := fptcloud_load_balancer_v2.PoolUpdateModel{
		Name:                  "Default server pool",
		Description:           "",
		Algorithm:             "ROUND_ROBIN",
		PersistenceType:       "",
		PersistenceCookieName: "",
		PoolMembers:           []fptcloud_load_balancer_v2.InputPoolMember{},
		HealthMonitor: fptcloud_load_balancer_v2.InputHealthMonitor{
			Type:           "HTTP",
			Delay:          "5",
			MaxRetries:     "3",
			MaxRetriesDown: "3",
			Timeout:        "5",
			HttpMethod:     "GET",
			UrlPath:        "/",
			ExpectedCodes:  "200",
		},
		AlpnProtocols: []string{},
		TlsEnabled:    false,
	}
	response, err := service.UpdatePool(vpcId, poolId, updateRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Update pool successfully", response.Message)
	assert.Equal(t, "d8a43897-f7cf-4c8c-8b0c-42c76f7e2d55", response.Data.Id)
	assert.Equal(t, "5181ab3b-49ee-45df-88d6-f5f4f3c0c45e", response.Data.LoadBalancerId)
	assert.Equal(t, "pool_updated", response.Data.Name)
	assert.Equal(t, "pool_updated", response.Data.Description)
	assert.Equal(t, "HTTP", response.Data.Protocol)
	assert.Equal(t, "ROUND_ROBIN", response.Data.Algorith)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, "6d5df57e-897f-4b51-a0a4-4029b4cb747b", response.Data.HealthMonitor.Id)
	assert.Equal(t, "HTTP", response.Data.HealthMonitor.Protocol)
	assert.Equal(t, "GET", response.Data.HealthMonitor.HttpMethod)
	assert.Equal(t, "200", response.Data.HealthMonitor.ExpectedCodes)
	assert.Equal(t, "/", response.Data.HealthMonitor.UrlPath)
	assert.Equal(t, 3, response.Data.HealthMonitor.MaxRetries)
	assert.Equal(t, 3, response.Data.HealthMonitor.MaxRetriesDown)
	assert.Equal(t, 5, response.Data.HealthMonitor.Delay)
	assert.Equal(t, 5, response.Data.HealthMonitor.Timeout)
	assert.Empty(t, response.Data.Members)
}

func TestDeletePoolSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Delete pool successfully",
		"data": {}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/pools/pool_id/delete": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	poolId := "pool_id"
	response, err := service.DeletePool(vpcId, poolId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Delete pool successfully", response.Message)
}

func TestListCertificatesSuccessfully(t *testing.T) {
	mockResponse := `{
		"data": [
			{
				"id": "22e80ac7-4b0e-41f2-85f2-0b54670a6fd9",
				"name": "cert1",
				"expired_at": "2026-03-12T10:42:57",
				"created_at": "2025-07-31T04:30:31"
			}
		],
		"message": "Get SSL Certificates successfully",
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/certificates?page=1&page_size=1000": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	response, err := service.ListCertificates(vpcId, 1, 1000)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, response.Total)
	assert.Equal(t, "22e80ac7-4b0e-41f2-85f2-0b54670a6fd9", response.Certificates[0].Id)
	assert.Equal(t, "cert1", response.Certificates[0].Name)
	assert.Equal(t, "2026-03-12T10:42:57", response.Certificates[0].ExpiredAt)
	assert.Equal(t, "2025-07-31T04:30:31", response.Certificates[0].CreatedAt)
}

func TestGetCertificateSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get SSL Certificate successfully",
		"data": {
			"id": "5475dd11-22d2-41f3-afcb-fbf4b40566c1",
			"name": "gwerg",
			"expired_at": "2026-03-12T10:42:57",
			"created_at": "2025-08-06T03:44:41"
		},
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/certificates/certificate_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	certificateId := "certificate_id"
	response, err := service.GetCertificate(vpcId, certificateId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Get SSL Certificate successfully", response.Message)
	assert.Equal(t, "5475dd11-22d2-41f3-afcb-fbf4b40566c1", response.Certificate.Id)
	assert.Equal(t, "gwerg", response.Certificate.Name)
	assert.Equal(t, "2026-03-12T10:42:57", response.Certificate.ExpiredAt)
	assert.Equal(t, "2025-08-06T03:44:41", response.Certificate.CreatedAt)
}

func TestCreateCertificateSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Create SSL Certificate successfully",
		"data": {
			"id": "4184fed3-3d38-4b3c-9e01-88fd4df8f2fa",
			"secret_ref": "hehe",
			"vpc_id": "dbaab857-1932-4779-88f9-e410c8b54f9a",
			"name": "cert",
			"created_at": "2025-10-14T04:19:40",
			"expired_at": "2026-03-12T10:42:57"
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/certificates/create": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	createRequest := fptcloud_load_balancer_v2.CertificateCreateModel{
		Name:        "cert",
		Certificate: "-----BEGIN CERTIFICATE-----fake-cert-----END CERTIFICATE-----",
		PrivateKey:  "-----BEGIN PRIVATE KEY-----fake-key-----END PRIVATE KEY-----",
		CertChain:   "-----BEGIN CERTIFICATE-----fake-chain-----END CERTIFICATE-----",
	}
	response, err := service.CreateCertificate(vpcId, createRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Create SSL Certificate successfully", response.Message)
	assert.Equal(t, "4184fed3-3d38-4b3c-9e01-88fd4df8f2fa", response.Data.Id)
	assert.Equal(t, "hehe", response.Data.SecretRef)
	assert.Equal(t, "dbaab857-1932-4779-88f9-e410c8b54f9a", response.Data.VpcId)
	assert.Equal(t, "cert", response.Data.Name)
	assert.Equal(t, "2025-10-14T04:19:40", response.Data.CreatedAt)
	assert.Equal(t, "2026-03-12T10:42:57", response.Data.ExpiredAt)
}

func TestDeleteCertificateSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Delete SSL Certificate successfully",
		"data": {}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/certificates/certificate_id/delete": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	certificateId := "certificate_id"
	response, err := service.DeleteCertificate(vpcId, certificateId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Delete SSL Certificate successfully", response.Message)
}

func TestListL7PoliciesSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get L7 Policies successfully",
		"data": [
			{
				"id": "4f4b3999-27dc-423d-8901-f198a991b180",
				"name": "policy",
				"action": "REDIRECT_TO_POOL",
				"provisioning_status": "Active",
				"redirect_pool": {
					"id": "1f6d8c2c-ca7b-4ca4-ae62-c0b9033fbf8d",
					"name": "Default server pool",
					"protocol": "HTTP"
				},
				"redirect_url": null,
				"redirect_prefix": null,
				"redirect_http_code": null,
				"position": "1",
				"created_at": "2025-10-08 04:51:38",
				"updated_at": "None"
			}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	response, err := service.ListL7Policies(vpcId, listenerId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, len(response.L7Policies))
	assert.Equal(t, "4f4b3999-27dc-423d-8901-f198a991b180", response.L7Policies[0].Id)
	assert.Equal(t, "policy", response.L7Policies[0].Name)
	assert.Equal(t, "REDIRECT_TO_POOL", response.L7Policies[0].Action)
	assert.Equal(t, "Active", response.L7Policies[0].ProvisioningStatus)
	assert.Equal(t, "1f6d8c2c-ca7b-4ca4-ae62-c0b9033fbf8d", response.L7Policies[0].RedirectPool.Id)
	assert.Equal(t, "Default server pool", response.L7Policies[0].RedirectPool.Name)
	assert.Equal(t, "HTTP", response.L7Policies[0].RedirectPool.Protocol)
	assert.Equal(t, "", response.L7Policies[0].RedirectUrl)
	assert.Equal(t, "", response.L7Policies[0].RedirectPrefix)
	assert.Equal(t, 0, response.L7Policies[0].RedirectHttpCode)
	assert.Equal(t, "1", response.L7Policies[0].Position)
	assert.Equal(t, "2025-10-08 04:51:38", response.L7Policies[0].CreatedAt)
	assert.Equal(t, "None", response.L7Policies[0].UpdatedAt)
}

func TestGetL7PolicySuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get L7 Policy successfully",
		"data": {
			"id": "fe065467-3ee3-4f93-85c6-1d59c32858ec",
			"name": "policy",
			"action": "REDIRECT_TO_POOL",
			"provisioning_status": "Active",
			"redirect_pool": {
				"id": "0bfed8b5-5d39-41e8-84f6-1c29f63d0614",
				"name": "Default server pool",
				"protocol": "HTTP"
			},
			"redirect_url": null,
			"redirect_prefix": null,
			"redirect_http_code": null,
			"position": "1",
			"created_at": "2025-10-06 07:21:22",
			"updated_at": "2025-10-08 07:19:18"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies/l7_policy_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	l7PolicyId := "l7_policy_id"
	response, err := service.GetL7Policy(vpcId, listenerId, l7PolicyId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Get L7 Policy successfully", response.Message)
	assert.Equal(t, "fe065467-3ee3-4f93-85c6-1d59c32858ec", response.L7Policy.Id)
	assert.Equal(t, "policy", response.L7Policy.Name)
	assert.Equal(t, "REDIRECT_TO_POOL", response.L7Policy.Action)
	assert.Equal(t, "Active", response.L7Policy.ProvisioningStatus)
	assert.Equal(t, "0bfed8b5-5d39-41e8-84f6-1c29f63d0614", response.L7Policy.RedirectPool.Id)
	assert.Equal(t, "Default server pool", response.L7Policy.RedirectPool.Name)
	assert.Equal(t, "HTTP", response.L7Policy.RedirectPool.Protocol)
	assert.Equal(t, "", response.L7Policy.RedirectUrl)
	assert.Equal(t, "", response.L7Policy.RedirectPrefix)
	assert.Equal(t, 0, response.L7Policy.RedirectHttpCode)
	assert.Equal(t, "1", response.L7Policy.Position)
	assert.Equal(t, "2025-10-06 07:21:22", response.L7Policy.CreatedAt)
	assert.Equal(t, "2025-10-08 07:19:18", response.L7Policy.UpdatedAt)
}

func TestCreateL7PolicySuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Create L7 Policy successfully",
		"data": {
			"id": "fe065467-3ee3-4f93-85c6-1d59c32858ec",
			"listener_id": "9800aa98-df5b-4061-aa0d-f4236c27f9a2",
			"name": "policy",
			"action": "REDIRECT_TO_POOL",
			"redirect_pool_id": "0bfed8b5-5d39-41e8-84f6-1c29f63d0614",
			"redirect_url": "",
			"redirect_prefix": "",
			"redirect_http_code": 0,
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"position": 1,
			"created_at": "2025-10-14T04:33:15",
			"updated_at": null
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies/create": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	createRequest := fptcloud_load_balancer_v2.L7PolicyInput{
		Name:             "policy",
		Action:           "REDIRECT_TO_POOL",
		RedirectPool:     "0bfed8b5-5d39-41e8-84f6-1c29f63d0614",
		RedirectUrl:      "",
		RedirectPrefix:   "",
		RedirectHttpCode: 0,
		Position:         1,
	}
	response, err := service.CreateL7Policy(vpcId, listenerId, createRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Create L7 Policy successfully", response.Message)
	assert.Equal(t, "fe065467-3ee3-4f93-85c6-1d59c32858ec", response.Data.Id)
	assert.Equal(t, "9800aa98-df5b-4061-aa0d-f4236c27f9a2", response.Data.ListenerId)
	assert.Equal(t, "policy", response.Data.Name)
	assert.Equal(t, "REDIRECT_TO_POOL", response.Data.Action)
	assert.Equal(t, "0bfed8b5-5d39-41e8-84f6-1c29f63d0614", response.Data.RedirectPoolId)
	assert.Equal(t, "", response.Data.RedirectUrl)
	assert.Equal(t, "", response.Data.RedirectPrefix)
	assert.Equal(t, 0, response.Data.RedirectHttpCode)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, 1, response.Data.Position)
	assert.Equal(t, "2025-10-14T04:33:15", response.Data.CreatedAt)
	assert.Equal(t, "", response.Data.UpdatedAt)
}

func TestUpdateL7PolicySuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Update L7 Policy successfully",
		"data": {
			"id": "fe065467-3ee3-4f93-85c6-1d59c32858ec",
			"listener_id": "9800aa98-df5b-4061-aa0d-f4236c27f9a2",
			"name": "policy",
			"action": "REDIRECT_TO_POOL",
			"redirect_pool_id": "0bfed8b5-5d39-41e8-84f6-1c29f63d0614",
			"redirect_url": "",
			"redirect_prefix": "",
			"redirect_http_code": 0,
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"position": 1,
			"created_at": "2025-10-14T04:33:15",
			"updated_at": null
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies/l7_policy_id/update": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	l7PolicyId := "l7_policy_id"
	updateRequest := fptcloud_load_balancer_v2.L7PolicyInput{
		Name:             "policy",
		Action:           "REDIRECT_TO_POOL",
		RedirectPool:     "0bfed8b5-5d39-41e8-84f6-1c29f63d0614",
		RedirectUrl:      "",
		RedirectPrefix:   "",
		RedirectHttpCode: 0,
		Position:         1,
	}
	response, err := service.UpdateL7Policy(vpcId, listenerId, l7PolicyId, updateRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Update L7 Policy successfully", response.Message)
	assert.Equal(t, "fe065467-3ee3-4f93-85c6-1d59c32858ec", response.Data.Id)
	assert.Equal(t, "9800aa98-df5b-4061-aa0d-f4236c27f9a2", response.Data.ListenerId)
	assert.Equal(t, "policy", response.Data.Name)
	assert.Equal(t, "REDIRECT_TO_POOL", response.Data.Action)
	assert.Equal(t, "0bfed8b5-5d39-41e8-84f6-1c29f63d0614", response.Data.RedirectPoolId)
	assert.Equal(t, "", response.Data.RedirectUrl)
	assert.Equal(t, "", response.Data.RedirectPrefix)
	assert.Equal(t, 0, response.Data.RedirectHttpCode)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, 1, response.Data.Position)
	assert.Equal(t, "2025-10-14T04:33:15", response.Data.CreatedAt)
	assert.Equal(t, "", response.Data.UpdatedAt)
}

func TestDeleteL7PolicySuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Delete L7 Policy successfully",
		"data": {}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies/l7_policy_id/delete": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	l7PolicyId := "l7_policy_id"
	response, err := service.DeleteL7Policy(vpcId, listenerId, l7PolicyId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Delete L7 Policy successfully", response.Message)
}

func TestListL7RulesSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get L7 Rules successfully",
		"data": {		
			"l7_policy_name": "policy",
			"rules": [
				{
					"id": "72000299-137c-4c7c-b48d-802f48f494a2",
					"l7_policy_id": "fe065467-3ee3-4f93-85c6-1d59c32858ec",
					"type": "HOST_NAME",
					"compare_type": "REGEX",
					"key": null,
					"value": "1",
					"invert": false,
					"operating_status": "Healthy",
					"provisioning_status": "Active"
				}
			]
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies/l7_policy_id/rules": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	l7PolicyId := "l7_policy_id"
	response, err := service.ListL7Rules(vpcId, listenerId, l7PolicyId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "policy", response.Data.L7PolicyName)
	assert.Equal(t, 1, len(response.Data.L7Rules))
	assert.Equal(t, "72000299-137c-4c7c-b48d-802f48f494a2", response.Data.L7Rules[0].Id)
	assert.Equal(t, "fe065467-3ee3-4f93-85c6-1d59c32858ec", response.Data.L7Rules[0].L7PolicyId)
	assert.Equal(t, "HOST_NAME", response.Data.L7Rules[0].Type)
	assert.Equal(t, "REGEX", response.Data.L7Rules[0].CompareType)
	assert.Equal(t, "", response.Data.L7Rules[0].Key)
	assert.Equal(t, "1", response.Data.L7Rules[0].Value)
	assert.False(t, response.Data.L7Rules[0].Invert)
	assert.Equal(t, "Healthy", response.Data.L7Rules[0].OperatingStatus)
	assert.Equal(t, "Active", response.Data.L7Rules[0].ProvisioningStatus)
}

func TestGetL7RuleSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Get L7 Rule successfully",
		"data": {		
			"id": "72000299-137c-4c7c-b48d-802f48f494a2",
			"l7_policy_id": "fe065467-3ee3-4f93-85c6-1d59c32858ec",
			"type": "HOST_NAME",
			"compare_type": "REGEX",
			"key": null,
			"value": "1",
			"invert": false,
			"operating_status": "Healthy",
			"provisioning_status": "Active"
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies/l7_policy_id/rules/l7_rule_id": mockResponse,
	})
	defer server.Close()

	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	l7PolicyId := "l7_policy_id"
	l7RuleId := "l7_rule_id"

	response, err := service.GetL7Rule(vpcId, listenerId, l7PolicyId, l7RuleId)

	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Get L7 Rule successfully", response.Message)
	assert.Equal(t, "72000299-137c-4c7c-b48d-802f48f494a2", response.L7Rule.Id)
	assert.Equal(t, "fe065467-3ee3-4f93-85c6-1d59c32858ec", response.L7Rule.L7PolicyId)
	assert.Equal(t, "HOST_NAME", response.L7Rule.Type)
	assert.Equal(t, "REGEX", response.L7Rule.CompareType)
	assert.Equal(t, "", response.L7Rule.Key)
	assert.Equal(t, "1", response.L7Rule.Value)
	assert.Equal(t, false, response.L7Rule.Invert)
	assert.Equal(t, "Healthy", response.L7Rule.OperatingStatus)
	assert.Equal(t, "Active", response.L7Rule.ProvisioningStatus)
}

func TestCreateL7RuleSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Create L7 Rule successfully",
		"data": {
			"id": "72000299-137c-4c7c-b48d-802f48f494a2",
			"l7_policy_id": "fe065467-3ee3-4f93-85c6-1d59c32858ec",
			"type": "HOST_NAME",
			"compare_type": "REGEX",
			"key": "",
			"value": "1",
			"invert": false,
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"created_at": "2025-10-14T04:38:25",
			"updated_at": null
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies/l7_policy_id/rules/create": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	l7PolicyId := "l7_policy_id"
	createRequest := fptcloud_load_balancer_v2.L7RuleInput{
		Type:        "HOST_NAME",
		CompareType: "REGEX",
		Key:         "",
		Value:       "1",
		Invert:      false,
	}
	response, err := service.CreateL7Rule(vpcId, listenerId, l7PolicyId, createRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Create L7 Rule successfully", response.Message)
	assert.Equal(t, "72000299-137c-4c7c-b48d-802f48f494a2", response.Data.Id)
	assert.Equal(t, "fe065467-3ee3-4f93-85c6-1d59c32858ec", response.Data.L7PolicyId)
	assert.Equal(t, "HOST_NAME", response.Data.Type)
	assert.Equal(t, "REGEX", response.Data.CompareType)
	assert.Equal(t, "", response.Data.Key)
	assert.Equal(t, "1", response.Data.Value)
	assert.False(t, response.Data.Invert)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, "2025-10-14T04:38:25", response.Data.CreatedAt)
	assert.Equal(t, "", response.Data.UpdatedAt)
}

func TestUpdateL7RuleSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Update L7 Rule successfully",
		"data": {
			"id": "72000299-137c-4c7c-b48d-802f48f494a2",
			"l7_policy_id": "fe065467-3ee3-4f93-85c6-1d59c32858ec",
			"type": "HOST_NAME",
			"compare_type": "REGEX",
			"key": "",
			"value": "1",
			"invert": false,
			"operating_status": "OFFLINE",
			"provisioning_status": "PENDING_CREATE",
			"created_at": "2025-10-14T04:38:25",
			"updated_at": null
		}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies/l7_policy_id/rules/l7_rule_id/update": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	l7PolicyId := "l7_policy_id"
	l7RuleId := "l7_rule_id"
	updateRequest := fptcloud_load_balancer_v2.L7RuleInput{
		Type:        "HOST_NAME",
		CompareType: "REGEX",
		Key:         "",
		Value:       "1",
		Invert:      false,
	}
	response, err := service.UpdateL7Rule(vpcId, listenerId, l7PolicyId, l7RuleId, updateRequest)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Update L7 Rule successfully", response.Message)
	assert.Equal(t, "72000299-137c-4c7c-b48d-802f48f494a2", response.Data.Id)
	assert.Equal(t, "fe065467-3ee3-4f93-85c6-1d59c32858ec", response.Data.L7PolicyId)
	assert.Equal(t, "HOST_NAME", response.Data.Type)
	assert.Equal(t, "REGEX", response.Data.CompareType)
	assert.Equal(t, "", response.Data.Key)
	assert.Equal(t, "1", response.Data.Value)
	assert.False(t, response.Data.Invert)
	assert.Equal(t, "OFFLINE", response.Data.OperatingStatus)
	assert.Equal(t, "PENDING_CREATE", response.Data.ProvisioningStatus)
	assert.Equal(t, "2025-10-14T04:38:25", response.Data.CreatedAt)
	assert.Equal(t, "", response.Data.UpdatedAt)
}

func TestDeleteL7RuleSuccessfully(t *testing.T) {
	mockResponse := `{
		"message": "Delete L7 Rule successfully",
		"data": {}
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vmware/vpc/vpc_id/load_balancer_v2/listeners/listener_id/l7policies/l7_policy_id/rules/l7_rule_id/delete": mockResponse,
	})
	defer server.Close()
	service := fptcloud_load_balancer_v2.NewLoadBalancerV2Service(mockClient)
	vpcId := "vpc_id"
	listenerId := "listener_id"
	l7PolicyId := "l7_policy_id"
	l7RuleId := "l7_rule_id"
	response, err := service.DeleteL7Rule(vpcId, listenerId, l7PolicyId, l7RuleId)
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Delete L7 Rule successfully", response.Message)
}
