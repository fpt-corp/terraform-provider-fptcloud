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
