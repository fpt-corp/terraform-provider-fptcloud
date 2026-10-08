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

// SecurityGroupListItem is one row of the paginated /security-groups endpoint.
// The list shape differs from the singular Find response: `firewall_type`
// carries what Find returns as the security group type (ACL/DFW), `type` is the
// group kind (IP_SET/LBV2), `ip_addresses` holds what Find returns as apply_to,
// tags are objects instead of tag_ids, and rules are always returned empty.
type SecurityGroupListItem struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	DisplayName  *string                 `json:"display_name"`
	CreatedAt    string                  `json:"created_at"`
	UpdatedAt    string                  `json:"updated_at"`
	Status       string                  `json:"status"`
	FirewallType string                  `json:"firewall_type"`
	IpAddresses  []string                `json:"ip_addresses"`
	Instances    []SecurityGroupInstance `json:"instances"`
	Tags         []SecurityGroupTag      `json:"tags"`
}

// SecurityGroupInstance is one member of a security group: an instance, a load
// balancer, or (type "other") an address that matches neither.
type SecurityGroupInstance struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	IpAddress   string  `json:"ip_address"`
	Ipv6Address *string `json:"ipv6_address"`
	Type        string  `json:"type"`
}

// SecurityGroupTag is a tag attached to a security group as returned by the list endpoint.
// Its `id` is the tag id, the same value Find returns in tag_ids.
type SecurityGroupTag struct {
	ID string `json:"id"`
}

// SecurityGroupListDTO holds the parameters of a paginated security group list request.
type SecurityGroupListDTO struct {
	VpcId    string
	PageSize int
}

// ListSecurityGroupsResponse is the paginated list response of the security-groups endpoint.
type ListSecurityGroupsResponse struct {
	Data  []SecurityGroupListItem `json:"data"`
	Total int                     `json:"total"`
}
