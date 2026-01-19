package fptcloud_vpc

type Tenant struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type VPC struct {
	Id     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

type Response struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type GetTenantResponse struct {
	Response
	Data *Tenant `json:"data,omitempty"`
}

type FindVPCResponse struct {
	Response
	Data *VPC `json:"data,omitempty"`
}

type FindVPCParam struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type CreateVPCDTO struct {
	Name              string   `json:"name"`
	Hypervisor        string   `json:"hypervisor,omitempty"`
	Owners            []string `json:"owners"`
	ProjectIaasId     string   `json:"project_iaas_id,omitempty"`
	SubnetName        string   `json:"subnet_name,omitempty"`
	NetworkType       string   `json:"network_type,omitempty"`
	CIDR              string   `json:"cidr,omitempty"`
	GatewayIp         string   `json:"gateway_ip,omitempty"`
	StaticIpPoolFrom   string   `json:"static_ip_pool_from,omitempty"`
	StaticIpPoolTo     string   `json:"static_ip_pool_to,omitempty"`
	TagIds            []string `json:"tag_ids,omitempty"`
}

type CreateVPCResponse struct {
	Response
	Data *CreateVPCData `json:"data,omitempty"`
}

type CreateVPCData struct {
	Id string `json:"id"`
}
