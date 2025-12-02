package fptcloud_security_group

type FindSecurityGroupDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	VpcId string `json:"vpc_id"`
}

type SecurityGroupRule struct {
	ID          string `json:"id"`
	Direction   string `json:"direction"`
	Action      string `json:"action"`
	Protocol    string `json:"protocol"`
	PortRange   string `json:"port_range"`
	Sources     string `json:"sources"`
	IpType      string `json:"ip_type"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// SecurityGroup represents a security group model
type SecurityGroup struct {
	VpcId         string              `json:"vpc_id"`
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	EdgeGatewayId string              `json:"edge_gateway_id"`
	Type          string              `json:"firewall_type"`
	ApplyTo       []string            `json:"apply_to"`
	Rules         []SecurityGroupRule `json:"rules"`
	CreatedAt     string              `json:"created_at"`
	Status        string              `json:"status"`
	TagIds        []string            `json:"tag_ids,omitempty"`
}

type CreatedSecurityGroupDTO struct {
	VpcId    string   `json:"vpc_id"`
	Name     string   `json:"name"`
	SubnetId string   `json:"subnet_id"`
	Type     string   `json:"type"`
	ApplyTo  []string `json:"apply_to"`
	TagIds   []string `json:"tag_ids,omitempty"`
}

type FindSecurityGroupResponse struct {
	Data SecurityGroup `json:"data"`
}
