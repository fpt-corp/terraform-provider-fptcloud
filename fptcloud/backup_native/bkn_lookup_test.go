package fptcloud_backup_native

import (
	"testing"

	fptcloud_flavor "terraform-provider-fptcloud/fptcloud/flavor"
	fptcloud_security_group "terraform-provider-fptcloud/fptcloud/security-group"
	fptcloud_subnet "terraform-provider-fptcloud/fptcloud/subnet"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeLookups() keepLookups {
	return keepLookups{
		listFlavors: func(string) ([]fptcloud_flavor.Flavor, error) {
			return []fptcloud_flavor.Flavor{{ID: "f-1", Name: "2C4G", Cpu: 2, MemoryMb: 4096}}, nil
		},
		listSubnets: func(string) ([]fptcloud_subnet.Subnet, error) {
			return []fptcloud_subnet.Subnet{{ID: "sub-1", Name: "Subnet A", NetworkID: "net-uuid-1", NetworkName: "subnet-a"}}, nil
		},
		listSecurityGroups: func(string) ([]fptcloud_security_group.SecurityGroupListItem, error) {
			return []fptcloud_security_group.SecurityGroupListItem{{ID: "sg-1", Name: "web"}}, nil
		},
		listFloatingIps: func(string) ([]FloatingIpItem, error) {
			return []FloatingIpItem{{Id: "ip-1", IpAddress: "203.0.113.5"}}, nil
		},
	}
}

func TestBuildRestoreKeepPayloadResolvesIds(t *testing.T) {
	p, err := buildRestoreKeepPayload("vpc-1", keepArgs{
		VmId: "vm-1", CreatedAt: "2026-10-12T19:00:00", PowerOn: true, InstanceName: "web-restored",
		FlavorId: "f-1", SubnetId: "sub-1", SecurityGroupIds: []string{"sg-1"}, FloatingIp: "ip-1",
	}, fakeLookups())
	require.NoError(t, err)
	assert.Equal(t, RestoreFlavor{Id: "f-1", Name: "2C4G", Cpu: 2, Ram: 4096}, p.HwFlavor)
	assert.Equal(t, IdName{Id: "net-uuid-1", Name: "subnet-a"}, p.Subnet, "the backend matches network_id and network name")
	assert.Equal(t, []IdName{{Id: "sg-1", Name: "web"}}, p.SecurityGroups)
	assert.Equal(t, RestoreFloatingIp{Id: "ip-1", Ip: "203.0.113.5"}, p.FloatingIp)
	require.NotNil(t, p.InstanceName)
	assert.Equal(t, "web-restored", *p.InstanceName)
	assert.Nil(t, p.NewPassword)
	assert.False(t, p.IsOverwrite)
}

func TestBuildRestoreKeepPayloadFloatingIpSentinels(t *testing.T) {
	base := keepArgs{VmId: "vm-1", CreatedAt: "2026-10-12T19:00:00", FlavorId: "f-1", SubnetId: "net-uuid-1"}
	p, err := buildRestoreKeepPayload("vpc-1", base, fakeLookups())
	require.NoError(t, err)
	assert.Equal(t, RestoreFloatingIp{Id: "", Ip: ""}, p.FloatingIp)
	assert.Equal(t, []IdName{}, p.SecurityGroups, "never nil")
	assert.Equal(t, "net-uuid-1", p.Subnet.Id, "a network_id is accepted as subnet_id too")

	base.FloatingIp = "new"
	p, err = buildRestoreKeepPayload("vpc-1", base, fakeLookups())
	require.NoError(t, err)
	assert.Equal(t, RestoreFloatingIp{Id: "new", Ip: "new"}, p.FloatingIp)
}

func TestBuildRestoreKeepPayloadUnknownIds(t *testing.T) {
	base := keepArgs{VmId: "vm-1", CreatedAt: "2026-10-12T19:00:00", FlavorId: "f-1", SubnetId: "sub-1"}

	bad := base
	bad.FlavorId = "f-9"
	_, err := buildRestoreKeepPayload("vpc-1", bad, fakeLookups())
	assert.ErrorContains(t, err, "flavor f-9 was not found in VPC vpc-1")

	bad = base
	bad.SubnetId = "sub-9"
	_, err = buildRestoreKeepPayload("vpc-1", bad, fakeLookups())
	assert.ErrorContains(t, err, "subnet sub-9 was not found")

	bad = base
	bad.SecurityGroupIds = []string{"sg-9"}
	_, err = buildRestoreKeepPayload("vpc-1", bad, fakeLookups())
	assert.ErrorContains(t, err, "security group sg-9 was not found")

	bad = base
	bad.FloatingIp = "ip-9"
	_, err = buildRestoreKeepPayload("vpc-1", bad, fakeLookups())
	assert.ErrorContains(t, err, "already in use")
}
