package fptcloud_backup_native

import (
	"fmt"

	common "terraform-provider-fptcloud/commons"
	fptcloud_flavor "terraform-provider-fptcloud/fptcloud/flavor"
	fptcloud_security_group "terraform-provider-fptcloud/fptcloud/security-group"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"
)

// keepArgs are the arguments of a keep restore, as configured.
type keepArgs struct {
	VmId             string
	CreatedAt        string
	InstanceName     string
	NewPassword      string
	FlavorId         string
	SubnetId         string
	FloatingIp       string
	PrivateIp        string
	SecurityGroupIds []string
	PowerOn          bool
}

// keepLookups lists what a keep restore needs to turn ids into the objects
// the restore-new API wants. Function fields keep it testable.
type keepLookups struct {
	listFlavors        func(vpcId string) ([]fptcloud_flavor.Flavor, error)
	listSubnets        func(vpcId string) ([]fptcloud_subnet.Subnet, error)
	listSecurityGroups func(vpcId string) ([]fptcloud_security_group.SecurityGroupListItem, error)
	listFloatingIps    func(vpcId string) ([]FloatingIpItem, error)
}

func newKeepLookups(client *common.Client) keepLookups {
	flavors := fptcloud_flavor.NewFlavorService(client)
	subnets := fptcloud_subnet.NewSubnetService(client)
	groups := fptcloud_security_group.NewSecurityGroupService(client)
	vm := NewVmService(client)
	return keepLookups{
		listFlavors: func(vpcId string) ([]fptcloud_flavor.Flavor, error) {
			list, err := flavors.ListFlavor(vpcId)
			if err != nil || list == nil {
				return nil, err
			}
			return *list, nil
		},
		listSubnets: func(vpcId string) ([]fptcloud_subnet.Subnet, error) {
			list, err := subnets.ListSubnet(vpcId)
			if err != nil || list == nil {
				return nil, err
			}
			return *list, nil
		},
		listSecurityGroups: func(vpcId string) ([]fptcloud_security_group.SecurityGroupListItem, error) {
			return groups.ListAll(fptcloud_security_group.SecurityGroupListDTO{VpcId: vpcId, PageSize: 100})
		},
		listFloatingIps: vm.ListFloatingIps,
	}
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// buildRestoreKeepPayload resolves the configured ids into the restore-new
// body (spec, fptcloud_backup_native_vm_restore, Create).
func buildRestoreKeepPayload(vpcId string, a keepArgs, l keepLookups) (VmRestoreKeepPayload, error) {
	p := VmRestoreKeepPayload{
		VmId:           a.VmId,
		CreatedAt:      a.CreatedAt,
		IsOverwrite:    false,
		InstanceName:   optional(a.InstanceName),
		NewPassword:    optional(a.NewPassword),
		PowerOn:        a.PowerOn,
		PrivateIp:      optional(a.PrivateIp),
		SecurityGroups: []IdName{},
	}

	flavors, err := l.listFlavors(vpcId)
	if err != nil {
		return p, fmt.Errorf("listing flavors: %w", err)
	}
	foundFlavor := false
	for _, f := range flavors {
		if f.ID == a.FlavorId {
			// The backend finds the flavor by name and checks quota with cpu and ram.
			p.HwFlavor = RestoreFlavor{Id: f.ID, Name: f.Name, Cpu: f.Cpu, Ram: f.MemoryMb}
			foundFlavor = true
			break
		}
	}
	if !foundFlavor {
		return p, fmt.Errorf("flavor %s was not found in VPC %s", a.FlavorId, vpcId)
	}

	subnets, err := l.listSubnets(vpcId)
	if err != nil {
		return p, fmt.Errorf("listing subnets: %w", err)
	}
	foundSubnet := false
	for _, s := range subnets {
		// subnet_id is the fptcloud_subnet id; a network_id is accepted too.
		if s.ID == a.SubnetId || s.NetworkID == a.SubnetId {
			p.Subnet = IdName{Id: s.NetworkID, Name: s.NetworkName}
			foundSubnet = true
			break
		}
	}
	if !foundSubnet {
		return p, fmt.Errorf("subnet %s was not found in VPC %s", a.SubnetId, vpcId)
	}

	if len(a.SecurityGroupIds) > 0 {
		groups, err := l.listSecurityGroups(vpcId)
		if err != nil {
			return p, fmt.Errorf("listing security groups: %w", err)
		}
		byId := make(map[string]string, len(groups))
		for _, g := range groups {
			byId[g.ID] = g.Name
		}
		for _, id := range a.SecurityGroupIds {
			name, ok := byId[id]
			if !ok {
				return p, fmt.Errorf("security group %s was not found in VPC %s", id, vpcId)
			}
			p.SecurityGroups = append(p.SecurityGroups, IdName{Id: id, Name: name})
		}
	}

	switch a.FloatingIp {
	case "", "new":
		// The portal sends empty values for "new" too, so the backend
		// receives the same payload as for no floating IP.
		p.FloatingIp = RestoreFloatingIp{Id: "", Ip: ""}
	default:
		ips, err := l.listFloatingIps(vpcId)
		if err != nil {
			return p, fmt.Errorf("listing floating IPs: %w", err)
		}
		found := false
		for _, ip := range ips {
			if ip.Id == a.FloatingIp {
				p.FloatingIp = RestoreFloatingIp{Id: ip.Id, Ip: ip.IpAddress}
				found = true
				break
			}
		}
		if !found {
			return p, fmt.Errorf("floating IP %s is not available in VPC %s: it does not exist or is already in use", a.FloatingIp, vpcId)
		}
	}
	return p, nil
}
