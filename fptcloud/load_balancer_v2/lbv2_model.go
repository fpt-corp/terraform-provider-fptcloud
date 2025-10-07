package fptcloud_load_balancer_v2

type CommonResponse struct {
	Data    bool   `json:"data"`
	Message string `json:"message,omitempty"`
}

type LoadBalancer struct {
	Id                 string   `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	OperatingStatus    string   `json:"operating_status"`
	ProvisioningStatus string   `json:"provisioning_status"`
	PrivateIp          string   `json:"private_ip"`
	Cidr               string   `json:"cidr"`
	CreatedAt          string   `json:"created_at"`
	Tags               []string `json:"tags"`
	EgwId              string   `json:"egw_id"`
	Size               struct {
		Id                    string `json:"id"`
		Name                  string `json:"name"`
		VipAmount             int32  `json:"vip_amount"`
		ActiveConnection      int32  `json:"active_connection"`
		ApplicationThroughput int32  `json:"application_throughput"`
	} `json:"size"`
	PublicIp struct {
		Id        string `json:"id"`
		IpAddress string `json:"ip_address"`
	} `json:"public_ip"`
	Network struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}
}

type LoadBalancerResponse struct {
	Data struct {
		Id                 string   `json:"id"`
		IdOnPlatform       string   `json:"id_on_platform"`
		VpcId              string   `json:"vpc_id"`
		SizeId             string   `json:"size_id"`
		NetworkId          string   `json:"network_id"`
		Cidr               string   `json:"cidr"`
		EgwId              string   `json:"egw_id"`
		IpAddressId        string   `json:"ip_address_id"`
		VirtualIpAddress   string   `json:"virtual_ip_address"`
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		OperatingStatus    string   `json:"operating_status"`
		ProvisioningStatus string   `json:"provisioning_status"`
		StatusMessage      string   `json:"status_message"`
		CreatedAt          string   `json:"created_at"`
		UpdatedAt          string   `json:"updated_at"`
		IsDeleted          bool     `json:"is_deleted"`
		Tags               []string `json:"tags"`
	} `json:"data"`
	Message string `json:"message"`
}
type LoadBalancerDetailResponse struct {
	LoadBalancer LoadBalancer `json:"data"`
	Message      string       `json:"message"`
}

type LoadBalancerListResponse struct {
	LoadBalancers []LoadBalancer `json:"data"`
	Total         int            `json:"total"`
	Message       string         `json:"message"`
}

type DefaultListener struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Protocol      string `json:"protocol"`
	ProtocolPort  string `json:"protocol_port"`
	CertificateId string `json:"certificate_id"`
}

type HealthMonitor struct {
	Type           string `json:"type"`
	Delay          int    `json:"delay"`
	MaxRetries     int    `json:"max_retries"`
	MaxRetriesDown int    `json:"max_retries_down"`
	Timeout        int    `json:"timeout"`
	HttpMethod     string `json:"http_method"`
	UrlPath        string `json:"url_path"`
	ExpectedCodes  string `json:"expected_codes"`
}

type PoolMember struct {
	Id        string `json:"id"`
	VmId      string `json:"vm_id"`
	VmName    string `json:"vm_name"`
	IpAddress string `json:"ip_address"`
	Network   struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"network"`
	Port               string `json:"port"`
	Weight             string `json:"weight"`
	OperatingStatus    string `json:"operating_status"`
	ProvisioningStatus string `json:"provisioning_status"`
	CreatedAt          string `json:"created_at"`
	IsExternal         bool   `json:"is_external"`
}

type InputPoolMember struct {
	VmId         string `json:"vm_id"`
	IpAddress    string `json:"ip_address"`
	NetworkId    string `json:"network_id"`
	ProtocolPort int    `json:"protocol_port"`
	Weight       int    `json:"weight"`
	Name         string `json:"name"`
	IsExternal   bool   `json:"is_external"`
}

type InputHealthMonitor struct {
	Type           string `json:"type"`
	Delay          string `json:"delay"`
	MaxRetries     string `json:"max_retries"`
	MaxRetriesDown string `json:"max_retries_down"`
	Timeout        string `json:"timeout"`
	HttpMethod     string `json:"http_method"`
	UrlPath        string `json:"url_path"`
	ExpectedCodes  string `json:"expected_codes"`
}

type InputDefaultServerPool struct {
	Id                    string             `json:"id"`
	Name                  string             `json:"name"`
	Algorithm             string             `json:"algorithm"`
	Protocol              string             `json:"protocol"`
	PersistenceType       string             `json:"persistence_type"`
	PersistenceCookieName string             `json:"persistence_cookie_name"`
	PoolMembers           []InputPoolMember  `json:"pool_members"`
	HealthMonitor         InputHealthMonitor `json:"health_monitor"`
}

type LoadBalancerCreateModel struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Size        string                 `json:"size"`
	FloatingIp  string                 `json:"floating_ip"`
	NetworkId   string                 `json:"network_id"`
	VipAddress  string                 `json:"vip_address"`
	Cidr        string                 `json:"cidr"`
	Listener    DefaultListener        `json:"listener"`
	Pool        InputDefaultServerPool `json:"pool"`
	EgwId       string                 `json:"egw_id"`
}

type LoadBalancerUpdateModel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	FloatingIp  string `json:"floating_ip"`
}

type LoadBalancerResizeModel struct {
	NewSize string `json:"new_size"`
}

// Certificate model
type Certificate struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	ExpiredAt string `json:"expired_at"`
	CreatedAt string `json:"created_at"`
}

type CertificateResponse struct {
	Data struct {
		Id        string `json:"id"`
		SecretRef string `json:"secret_ref"`
		VpcId     string `json:"vpc_id"`
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
		ExpiredAt string `json:"expired_at"`
	} `json:"data"`
	Message string `json:"message"`
}

type CertificateDetailResponse struct {
	Certificate Certificate `json:"data"`
	Message     string      `json:"message"`
}

type CertificateListResponse struct {
	Certificates []Certificate `json:"data"`
	Total        int           `json:"total"`
	Message      string        `json:"message"`
}

type CertificateCreateModel struct {
	Name        string `json:"name"`
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
	CertChain   string `json:"cert_chain"`
}

// Listener models
type Listener struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	ProvisioningStatus string `json:"provisioning_status"`
	Protocol           string `json:"protocol"`
	Port               string `json:"port"`
	LoadBalancerId     string `json:"load_balancer_id"`
	InsertHeaders      struct {
		XForwardedFor   string `json:"X-Forwarded-For"`
		XForwardedPort  string `json:"X-Forwarded-Port"`
		XForwardedProto string `json:"X-Forwarded-Proto"`
	} `json:"insert_headers"`
	DefaultPool           InputDefaultServerPool `json:"default_pool"`
	Certificate           Certificate            `json:"certificate"`
	SniCertificates       []Certificate          `json:"sni_certificates"`
	HstsMaxAge            int                    `json:"hsts_max_age"`
	HstsIncludeSubdomains bool                   `json:"hsts_include_subdomains"`
	HstsPreload           bool                   `json:"hsts_preload"`
	ConnectionLimit       int                    `json:"connection_limit"`
	ClientDataTimeout     int                    `json:"client_data_timeout"`
	MemberConnectTimeout  int                    `json:"member_connect_timeout"`
	MemberDataTimeout     int                    `json:"member_data_timeout"`
	TcpInspectTimeout     int                    `json:"tcp_inspect_timeout"`
	AlpnProtocols         []string               `json:"alpn_protocols"`
	CreatedAt             string                 `json:"created_at"`
	AllowedCidrs          []string               `json:"allowed_cidrs"`
	Tags                  []string               `json:"tags"`
}

type ListenerDetailResponse struct {
	Listener Listener `json:"data"`
	Message  string   `json:"message"`
}

type ListenerResponse struct {
	Data struct {
		Id                      string            `json:"id"`
		LoadBalancerId          string            `json:"load_balancer_id"`
		Name                    string            `json:"name"`
		Description             string            `json:"description"`
		CertificateId           string            `json:"certificate_id"`
		SniCertificateIds       []string          `json:"sni_certificate_ids"`
		DefaultPoolId           string            `json:"default_pool_id"`
		OperatingStatus         string            `json:"operating_status"`
		ProvisioningStatus      string            `json:"provisioning_status"`
		Protocol                string            `json:"protocol"`
		Port                    int               `json:"port"`
		InsertHeaders           map[string]string `json:"insert_headers"`
		HstsMaxAge              int               `json:"hsts_max_age"`
		HstsIncludeSubdomains   bool              `json:"hsts_include_subdomains"`
		HstsPreload             bool              `json:"hsts_preload"`
		ConnectionLimit         int               `json:"connection_limit"`
		ClientDataTimeout       int               `json:"client_data_timeout"`
		MemberConnectionTimeout int               `json:"member_connect_timeout"`
		MemberDataTimeout       int               `json:"member_data_timeout"`
		TcpInspectTimeout       int               `json:"tcp_inspect_timeout"`
		AllowedCidrs            []string          `json:"allowed_cidrs"`
		AlpnProtocols           []string          `json:"alpn_protocols"`
		CreatedAt               string            `json:"created_at"`
		UpdatedAt               string            `json:"updated_at"`
		Tags                    []string          `json:"tags"`
	} `json:"data"`
	Message string `json:"message"`
}

type ListenerListResponse struct {
	Listeners []Listener `json:"data"`
	Total     int        `json:"total"`
	Message   string     `json:"message"`
}

type ListenerCreateModel struct {
	Name                  string          `json:"name"`
	Description           string          `json:"description"`
	Protocol              string          `json:"protocol"`
	ProtocolPort          string          `json:"protocol_port"`
	DefaultPoolId         string          `json:"default_pool_id"`
	CertificateId         string          `json:"certificate_id"`
	SniCertificateIds     []string        `json:"sni_certificate_ids"`
	ConnectionLimit       int             `json:"connection_limit"`
	ClientDataTimeout     int             `json:"client_data_timeout"`
	MemberConnectTimeout  int             `json:"member_connect_timeout"`
	MemberDataTimeout     int             `json:"member_data_timeout"`
	TcpInspectTimeout     int             `json:"tcp_inspect_timeout"`
	InsertHeaders         map[string]bool `json:"insert_headers"`
	HstsMaxAge            int             `json:"hsts_max_age"`
	HstsIncludeSubdomains bool            `json:"hsts_include_subdomains"`
	HstsPreload           bool            `json:"hsts_preload"`
	AllowedCidrs          []string        `json:"allowed_cidrs"`
	AlpnProtocols         []string        `json:"alpn_protocols"`
}

type ListenerUpdateModel struct {
	Name                  string          `json:"name"`
	Description           string          `json:"description"`
	DefaultPoolId         string          `json:"default_pool_id"`
	CertificateId         string          `json:"certificate_id"`
	SniCertificateIds     []string        `json:"sni_certificate_ids"`
	ConnectionLimit       int             `json:"connection_limit"`
	ClientDataTimeout     int             `json:"client_data_timeout"`
	MemberConnectTimeout  int             `json:"member_connect_timeout"`
	MemberDataTimeout     int             `json:"member_data_timeout"`
	TcpInspectTimeout     int             `json:"tcp_inspect_timeout"`
	InsertHeaders         map[string]bool `json:"insert_headers"`
	HstsMaxAge            int             `json:"hsts_max_age"`
	HstsIncludeSubdomains bool            `json:"hsts_include_subdomains"`
	HstsPreload           bool            `json:"hsts_preload"`
	AllowedCidrs          []string        `json:"allowed_cidrs"`
	AlpnProtocols         []string        `json:"alpn_protocols"`
}

// Pool
type Pool struct {
	Id                    string        `json:"id"`
	Name                  string        `json:"name"`
	Description           string        `json:"description"`
	LoadBalancerId        string        `json:"load_balancer_id"`
	OperatingStatus       string        `json:"operating_status"`
	ProvisioningStatus    string        `json:"provisioning_status"`
	Protocol              string        `json:"protocol"`
	Algorithm             string        `json:"algorithm"`
	HealthMonitor         HealthMonitor `json:"health_monitor"`
	Members               []PoolMember  `json:"members"`
	PersistenceType       string        `json:"persistence_type"`
	PersistenceCookieName string        `json:"persistence_cookie_name"`
	AlpnProtocols         []string      `json:"alpn_protocols"`
	TlsEnabled            bool          `json:"tls_enabled"`
	CreatedAt             string        `json:"created_at"`
	Tags                  []string      `json:"tags"`
}

type PoolResponse struct {
	Data struct {
		Id                    string   `json:"id"`
		LoadBalancerId        string   `json:"load_balancer_id"`
		Name                  string   `json:"name"`
		Description           string   `json:"description"`
		Protocol              string   `json:"protocol"`
		OperatingStatus       string   `json:"operating_status"`
		ProvisioningStatus    string   `json:"provisioning_status"`
		Algorith              string   `json:"algorithm"`
		PersistenceType       string   `json:"persistence_type"`
		PersistenceCookieName string   `json:"persistence_cookie_name"`
		AlpnProtocols         []string `json:"alpn_protocols"`
		TlsEnabled            bool     `json:"tls_enabled"`
		CreatedAt             string   `json:"created_at"`
		UpdatedAt             string   `json:"updated_at"`
		Tags                  []string `json:"tags"`
		HealthMonitor         struct {
			Id             string `json:"id"`
			PoolId         string `json:"pool_id"`
			Protocol       string `json:"protocol"`
			HttpMethod     string `json:"http_method"`
			ExpectedCodes  string `json:"expected_codes"`
			UrlPath        string `json:"url_path"`
			MaxRetries     int    `json:"max_retries"`
			MaxRetriesDown int    `json:"max_retries_down"`
			Delay          int    `json:"delay"`
			Timeout        int    `json:"timeout"`
		} `json:"health_monitor"`
		Members []struct {
			Id                 string `json:"id"`
			IdOnPlatform       string `json:"id_on_platform"`
			PoolId             string `json:"pool_id"`
			NetworkId          string `json:"network_id"`
			VmId               string `json:"vm_id"`
			Name               string `json:"name"`
			IsExternal         bool   `json:"is_external"`
			TargetIpAddress    string `json:"target_ip_address"`
			TargetPort         int    `json:"target_port"`
			Weight             int    `json:"weight"`
			OperatingStatus    string `json:"operating_status"`
			ProvisioningStatus string `json:"provisioning_status"`
			CreatedAt          string `json:"created_at"`
			UpdatedAt          string `json:"updated_at"`
			IsDeleted          bool   `json:"is_deleted"`
		} `json:"members"`
	} `json:"data"`
	Message string `json:"message"`
}

type PoolDetailResponse struct {
	Pool    Pool   `json:"data"`
	Message string `json:"message"`
}

type PoolListResponse struct {
	Pools   []Pool `json:"data"`
	Total   int    `json:"total"`
	Message string `json:"message"`
}

type PoolCreateModel struct {
	Name                  string             `json:"name"`
	Description           string             `json:"description"`
	Algorithm             string             `json:"algorithm"`
	Protocol              string             `json:"protocol"`
	PersistenceType       string             `json:"persistence_type"`
	PersistenceCookieName string             `json:"persistence_cookie_name"`
	PoolMembers           []InputPoolMember  `json:"pool_members"`
	HealthMonitor         InputHealthMonitor `json:"health_monitor"`
	AlpnProtocols         []string           `json:"alpn_protocols"`
	TlsEnabled            bool               `json:"tls_enabled"`
}

type PoolUpdateModel struct {
	Name                  string             `json:"name"`
	Description           string             `json:"description"`
	Algorithm             string             `json:"algorithm"`
	PersistenceType       string             `json:"persistence_type"`
	PersistenceCookieName string             `json:"persistence_cookie_name"`
	PoolMembers           []InputPoolMember  `json:"pool_members"`
	HealthMonitor         InputHealthMonitor `json:"health_monitor"`
	AlpnProtocols         []string           `json:"alpn_protocols"`
	TlsEnabled            bool               `json:"tls_enabled"`
}

// L7 Policy
type L7Policy struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	Action             string `json:"action"`
	ProvisioningStatus string `json:"provisioning_status"`
	RedirectPool       struct {
		Id       string `json:"id"`
		Name     string `json:"name"`
		Protocol string `json:"protocol"`
	} `json:"redirect_pool"`
	RedirectUrl      string `json:"redirect_url"`
	RedirectPrefix   string `json:"redirect_prefix"`
	RedirectHttpCode int    `json:"redirect_http_code"`
	Position         string `json:"position"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type L7PolicyResponse struct {
	Data struct {
		Id                 string `json:"id"`
		ListenerId         string `json:"listener_id"`
		Name               string `json:"name"`
		Action             string `json:"action"`
		RedirectPoolId     string `json:"redirect_pool_id"`
		RedirectUrl        string `json:"redirect_url"`
		RedirectPrefix     string `json:"redirect_prefix"`
		RedirectHttpCode   int    `json:"redirect_http_code"`
		OperatingStatus    string `json:"operating_status"`
		ProvisioningStatus string `json:"provisioning_status"`
		Position           int    `json:"position"`
		CreatedAt          string `json:"created_at"`
		UpdatedAt          string `json:"updated_at"`
	} `json:"data"`
	Message string `json:"message"`
}

type L7PolicyDetailResponse struct {
	L7Policy L7Policy `json:"data"`
	Message  string   `json:"message"`
}

type L7PolicyListResponse struct {
	L7Policies []L7Policy `json:"data"`
	Total      int        `json:"total"`
	Message    string     `json:"message"`
}

type L7PolicyInput struct {
	Name             string `json:"name"`
	Action           string `json:"action"`
	RedirectPool     string `json:"redirect_pool"`
	RedirectUrl      string `json:"redirect_url"`
	RedirectPrefix   string `json:"redirect_prefix"`
	RedirectHttpCode int    `json:"redirect_http_code"`
	Position         int    `json:"position"`
}

// L7 Rule
type L7Rule struct {
	Id                 string `json:"id"`
	L7PolicyId         string `json:"l7_policy_id"`
	Type               string `json:"type"`
	CompareType        string `json:"compare_type"`
	Key                string `json:"key"`
	Value              string `json:"value"`
	Invert             bool   `json:"invert"`
	OperatingStatus    string `json:"operating_status"`
	ProvisioningStatus string `json:"provisioning_status"`
}

type L7RuleResponse struct {
	Data struct {
		Id                 string `json:"id"`
		L7PolicyId         string `json:"l7_policy_id"`
		Type               string `json:"type"`
		CompareType        string `json:"compare_type"`
		Key                string `json:"key"`
		Value              string `json:"value"`
		Invert             bool   `json:"invert"`
		OperatingStatus    string `json:"operating_status"`
		ProvisioningStatus string `json:"provisioning_status"`
		CreatedAt          string `json:"created_at"`
		UpdatedAt          string `json:"updated_at"`
	} `json:"data"`
	Message string `json:"message"`
}

type L7RuleDetailResponse struct {
	L7Rule  L7Rule `json:"data"`
	Message string `json:"message"`
}

type L7RuleListResponse struct {
	Data struct {
		L7PolicyName string   `json:"l7_policy_name"`
		L7Rules      []L7Rule `json:"rules"`
	} `json:"data"`
	Total   int    `json:"total"`
	Message string `json:"message"`
}

type L7RuleInput struct {
	Type        string `json:"type"`
	CompareType string `json:"compare_type"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Invert      bool   `json:"invert"`
}

// Size
type Size struct {
	Id                    string `json:"id"`
	Name                  string `json:"name"`
	VipAmount             int    `json:"vip_amount"`
	ActiveConnection      int    `json:"active_connection"`
	ApplicationThroughput int    `json:"application_throughput"`
}
type SizeListResponse struct {
	Sizes   []Size `json:"data"`
	Total   int    `json:"total"`
	Message string `json:"message"`
}
