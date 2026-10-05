package fptcloud_security_group_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	common "terraform-provider-fptcloud/commons"
	fptcloud_security_group "terraform-provider-fptcloud/fptcloud/security-group"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"

	test_helper "terraform-provider-fptcloud/commons/test-helper"
)

// TestAccFptCloudSecurityGroups_basic lists the security groups of VPC_ID twice —
// once in a single page and once one security group per page — and checks both
// return the same security groups, and that the first one matches the
// fptcloud_security_group data source. VPC_ID must contain at least one security group.
func TestAccFptCloudSecurityGroups_basic(t *testing.T) {
	for _, name := range []string{
		"FPTCLOUD_TOKEN",
		"FPTCLOUD_TENANT_NAME",
		"FPTCLOUD_REGION",
		"VPC_ID",
	} {
		if os.Getenv(name) == "" {
			t.Skipf("%s must be set to run this acceptance test", name)
		}
	}

	vpcId := os.Getenv("VPC_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { test_helper.TestPreCheck(t) },
		ProviderFactories: test_helper.TestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityGroupsConfig(vpcId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.fptcloud_security_groups.all", "id", vpcId),
					resource.TestCheckResourceAttr("data.fptcloud_security_groups.all", "page_size", "100"),
					resource.TestCheckResourceAttrSet("data.fptcloud_security_groups.all", "security_groups.0.id"),
					testAccCheckSameSecurityGroups("data.fptcloud_security_groups.all", "data.fptcloud_security_groups.paged"),
					resource.TestCheckResourceAttrPair("data.fptcloud_security_groups.all", "security_groups.0.name", "data.fptcloud_security_group.first", "name"),
					resource.TestCheckResourceAttrPair("data.fptcloud_security_groups.all", "security_groups.0.type", "data.fptcloud_security_group.first", "type"),
					resource.TestCheckResourceAttrPair("data.fptcloud_security_groups.all", "security_groups.0.apply_to.#", "data.fptcloud_security_group.first", "apply_to.#"),
					resource.TestCheckResourceAttrPair("data.fptcloud_security_groups.all", "security_groups.0.created_at", "data.fptcloud_security_group.first", "created_at"),
				),
			},
		},
	})
}

// testAccCheckSameSecurityGroups checks that two fptcloud_security_groups data sources hold the same groups in the same order.
func testAccCheckSameSecurityGroups(a, b string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		left, ok := s.RootModule().Resources[a]
		if !ok {
			return fmt.Errorf("%s not found in state", a)
		}
		right, ok := s.RootModule().Resources[b]
		if !ok {
			return fmt.Errorf("%s not found in state", b)
		}
		count := left.Primary.Attributes["security_groups.#"]
		if count != right.Primary.Attributes["security_groups.#"] {
			return fmt.Errorf("%s has %s security groups, %s has %s", a, count, b, right.Primary.Attributes["security_groups.#"])
		}
		var n int
		_, _ = fmt.Sscan(count, &n)
		for i := 0; i < n; i++ {
			key := fmt.Sprintf("security_groups.%d.id", i)
			if left.Primary.Attributes[key] != right.Primary.Attributes[key] {
				return fmt.Errorf("%s differs: %q vs %q", key, left.Primary.Attributes[key], right.Primary.Attributes[key])
			}
		}
		return nil
	}
}

func TestDataSourceSecurityGroups_SchemaIsValid(t *testing.T) {
	r := fptcloud_security_group.DataSourceSecurityGroups()
	assert.NoError(t, r.InternalValidate(nil, false))
	assert.Equal(t, 100, r.Schema["page_size"].Default)
}

func TestDataSourceSecurityGroups_ReadMapsListToSingularFields(t *testing.T) {
	// Trimmed from a production /security-groups response: `type` is the group
	// kind, `firewall_type` is what fptcloud_security_group calls `type`, and
	// rules are always returned empty by the list.
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/security-groups": `{"total": 1, "data": [
			{"id": "sg-1", "name": "default-zlhe00to", "display_name": "default",
			 "ip_addresses": ["10.7.9.247", "203.0.113.0/24"],
			 "instances": [
				{"id": "vm-1", "name": "vm-1", "status": "POWERED_ON", "ip_address": "10.7.9.247", "ipv6_address": null, "type": "instance"},
				{"id": "203.0.113.0/24", "name": "203.0.113.0/24", "status": "Unknown", "ip_address": "203.0.113.0/24", "type": "other"}
			 ],
			 "created_at": "2026-09-18T16:44:43", "updated_at": "2026-10-05T08:15:42",
			 "edge_gateway": {"name": "gw", "id": "egw-1", "vdc_group_id": null, "vdc_group_name": null, "edge_gateway_id": "egw-infra-1"},
			 "is_vdc_group": false, "rules": [], "status": "REALIZED", "firewall_group_id": "fg-1",
			 "has_firewall_ip_address": true, "has_firewall_rules": true, "firewall_type": "ACL",
			 "tags": [{"id": "tag-1", "key": "env", "value": "prod", "color": "red"}], "type": "IP_SET"}
		]}`,
	})
	defer server.Close()

	r := fptcloud_security_group.DataSourceSecurityGroups()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": "vpc_id"})
	diags := r.ReadContext(context.Background(), d, mockClient)
	assert.False(t, diags.HasError(), "%v", diags)

	assert.Equal(t, "vpc_id", d.Id())
	groups := d.Get("security_groups").([]interface{})
	assert.Len(t, groups, 1)

	sg := groups[0].(map[string]interface{})
	assert.Equal(t, "sg-1", sg["id"])
	assert.Equal(t, "default", sg["name"], "the display name wins, as in fptcloud_security_group")
	assert.Equal(t, "ACL", sg["type"])
	assert.Equal(t, []interface{}{"10.7.9.247", "203.0.113.0/24"}, sg["apply_to"])
	assert.Equal(t, "REALIZED", sg["status"])
	assert.Equal(t, []interface{}{"tag-1"}, sg["tag_ids"])

	members := sg["members"].([]interface{})
	assert.Len(t, members, 2)
	assert.Equal(t, "instance", members[0].(map[string]interface{})["type"])
	assert.Equal(t, "other", members[1].(map[string]interface{})["type"])
}

func testAccSecurityGroupsConfig(vpcId string) string {
	return fmt.Sprintf(`
data "fptcloud_security_groups" "all" {
  vpc_id = %[1]q
}

data "fptcloud_security_groups" "paged" {
  vpc_id    = %[1]q
  page_size = 1
}

data "fptcloud_security_group" "first" {
  vpc_id = %[1]q
  id     = data.fptcloud_security_groups.all.security_groups[0].id
}
`, vpcId)
}
