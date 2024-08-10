package fptcloud_subnet

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net"
	"strings"
)

var resourceSubnet = map[string]*schema.Schema{
	"vpc_id": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The vpc id of the subnet",
		ForceNew:    true,
	},
	"name": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validation.NoZeroValues,
		Description:  "The name of the subnet",
		ForceNew:     true,
	},
	"type": {
		Type:        schema.TypeString,
		Required:    true,
		Description: "The type of the subnet. `NAT_ROUTED`: To the Internet via a NAT gateway. `ISOLATED`: Subnet won't route to the Internet",
		ForceNew:    true,
		ValidateFunc: validation.StringInSlice([]string{
			"ISOLATED", "NAT_ROUTED",
		}, false)},

	"cidr": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validateCIDR,
		Description:  "The network address (CIDR) of the subnet. CIDR block format: 10.0.0.1/24",
		ForceNew:     true,
	},
	"gateway_ip": {
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validateIPv4Address,
		Description:  "The gateway ip of the subnet",
		ForceNew:     true,
	},
	"static_ip_pool": {
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validateIPv4Range,
		Description:  "The static ip pool of the instance. Only if you want to create subnet with static IP pool, enter an valid IP range within provided CIDR.",
		ForceNew:     true,
	},
	"network_name": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"gateway": {
		Type:     schema.TypeString,
		Computed: true,
	},
	"created_at": {
		Type:     schema.TypeString,
		Computed: true,
	},
}

// validateCIDR is a ValidateFunc that checks if a given value is a valid CIDR block.
func validateCIDR(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if _, _, err := net.ParseCIDR(value); err != nil {
		es = append(es, fmt.Errorf("%q must be a valid CIDR block: %s", k, err))
	}

	return
}

// validateIPv4Address checks if a given value is a valid IPv4 address.
func validateIPv4Address(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)

	ip := net.ParseIP(value)
	if ip == nil || ip.To4() == nil {
		es = append(es, fmt.Errorf("%q must be a valid IPv4 address", k))
	}

	return
}

// validateIPv4Range checks if a given value is a valid range of IPv4 addresses.
func validateIPv4Range(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)

	// Split the range into two IP addresses
	ips := strings.Split(value, "-")
	if len(ips) != 2 {
		es = append(es, fmt.Errorf("%q must be a valid IPv4 range (e.g., '172.168.1.2 - 172.168.1.254 ')", k))
		return
	}

	startIP := net.ParseIP(ips[0])
	endIP := net.ParseIP(ips[1])

	if startIP == nil || startIP.To4() == nil {
		es = append(es, fmt.Errorf("%q: start IP must be a valid IPv4 address", k))
	}

	if endIP == nil || endIP.To4() == nil {
		es = append(es, fmt.Errorf("%q: end IP must be a valid IPv4 address", k))
	}

	if len(es) > 0 {
		return // No need to proceed if there are already errors
	}

	// Convert IPs to 4-byte representation for comparison
	startIPBytes := startIP.To4()
	endIPBytes := endIP.To4()

	// Compare the start and end IPs
	if compareIPs(startIPBytes, endIPBytes) > 0 {
		es = append(es, fmt.Errorf("%q: start IP must be less than or equal to end IP", k))
	}

	return
}

// compareIPs compares two IPv4 addresses.
// Returns a negative number if ip1 < ip2, zero if ip1 == ip2, positive if ip1 > ip2.
func compareIPs(ip1, ip2 net.IP) int {
	for i := 0; i < 4; i++ {
		if ip1[i] < ip2[i] {
			return -1
		}
		if ip1[i] > ip2[i] {
			return 1
		}
	}
	return 0
}

// parseIPRange extracts the start and end IPs from the range string
func parseIPRange(ipRange string) (string, string) {
	if ipRange == "" {
		return "", ""
	}
	ips := strings.Split(ipRange, "-")
	if len(ips) != 2 {
		return "", ""
	}
	return ips[0], ips[1]
}
