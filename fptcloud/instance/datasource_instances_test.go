package fptcloud_instance_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	common "terraform-provider-fptcloud/commons"
	fptcloud_instance "terraform-provider-fptcloud/fptcloud/instance"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"

	test_helper "terraform-provider-fptcloud/commons/test-helper"
)

// TestAccFptCloudInstances_basic lists the instances of VPC_ID twice — once in a
// single page and once one instance per page — and checks both return the same
// instances, and that the first one matches the fptcloud_instance data source.
// VPC_ID must contain at least one instance.
func TestAccFptCloudInstances_basic(t *testing.T) {
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
				Config: testAccInstancesConfig(vpcId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.fptcloud_instances.all", "id", vpcId),
					resource.TestCheckResourceAttr("data.fptcloud_instances.all", "page_size", "25"),
					resource.TestCheckResourceAttrSet("data.fptcloud_instances.all", "instances.0.id"),
					testAccCheckSameInstances("data.fptcloud_instances.all", "data.fptcloud_instances.paged"),
					resource.TestCheckResourceAttrPair("data.fptcloud_instances.all", "instances.0.name", "data.fptcloud_instance.first", "name"),
					resource.TestCheckResourceAttrPair("data.fptcloud_instances.all", "instances.0.private_ip", "data.fptcloud_instance.first", "private_ip"),
					resource.TestCheckResourceAttrPair("data.fptcloud_instances.all", "instances.0.cpu_number", "data.fptcloud_instance.first", "cpu_number"),
					resource.TestCheckResourceAttrPair("data.fptcloud_instances.all", "instances.0.memory_mb", "data.fptcloud_instance.first", "memory_mb"),
					resource.TestCheckResourceAttrPair("data.fptcloud_instances.all", "instances.0.flavor_name", "data.fptcloud_instance.first", "flavor_name"),
					resource.TestCheckResourceAttrPair("data.fptcloud_instances.all", "instances.0.created_at", "data.fptcloud_instance.first", "created_at"),
				),
			},
		},
	})
}

// testAccCheckSameInstances checks that two fptcloud_instances data sources hold the same instances in the same order.
func testAccCheckSameInstances(a, b string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		left, ok := s.RootModule().Resources[a]
		if !ok {
			return fmt.Errorf("%s not found in state", a)
		}
		right, ok := s.RootModule().Resources[b]
		if !ok {
			return fmt.Errorf("%s not found in state", b)
		}
		count := left.Primary.Attributes["instances.#"]
		if count != right.Primary.Attributes["instances.#"] {
			return fmt.Errorf("%s has %s instances, %s has %s", a, count, b, right.Primary.Attributes["instances.#"])
		}
		var n int
		_, _ = fmt.Sscan(count, &n)
		for i := 0; i < n; i++ {
			key := fmt.Sprintf("instances.%d.id", i)
			if left.Primary.Attributes[key] != right.Primary.Attributes[key] {
				return fmt.Errorf("%s differs: %q vs %q", key, left.Primary.Attributes[key], right.Primary.Attributes[key])
			}
		}
		return nil
	}
}

func TestDataSourceInstances_SchemaIsValid(t *testing.T) {
	r := fptcloud_instance.DataSourceInstances()
	assert.NoError(t, r.InternalValidate(nil, false))
	assert.Equal(t, 25, r.Schema["page_size"].Default)
}

func TestDataSourceInstances_ReadMapsListToSingularFields(t *testing.T) {
	// Trimmed from a production /compute/instances?with_flavor=true response.
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/compute/instances": `{"total": 1, "data": [
			{"id": "vm-1", "vpc_id": "vpc_id", "created_at": "2026-09-16T16:03:26", "updated_at": "2026-10-05T08:17:10",
			 "name": "vm-26091611258", "status": "POWERED_ON", "guest_os": "Ubuntu Linux (64-bit)",
			 "host_name": "vm-26091611258", "ip_address": "192.168.37.2", "ipv6_address": null,
			 "number_of_cpus": 1, "memory_mb": 1024, "network_name": "BSS-test_VPC_VMW_03",
			 "platform": null, "vm_group_id": "group-1", "flavor_id": "flavor-1", "is_nvme": false,
			 "name_infra": "vm-26091611258", "vGpuIds": [], "ip_public": "203.0.113.10",
			 "enabled_allocate_ip": true, "storage_size_gb": 0,
			 "vm_tags": [{"id": "tag-1", "key": "env", "value": "prod", "color": "red"}],
			 "billing_type": null, "gpu_name": null, "is_multi_storage": false,
			 "flavor": {"id": "flavor-1", "name": "Small-1", "info": {"vcpu": 1}, "is_nvme": false}}
		]}`,
	})
	defer server.Close()

	r := fptcloud_instance.DataSourceInstances()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": "vpc_id"})
	diags := r.ReadContext(context.Background(), d, mockClient)
	assert.False(t, diags.HasError(), "%v", diags)

	assert.Equal(t, "vpc_id", d.Id())
	instances := d.Get("instances").([]interface{})
	assert.Len(t, instances, 1)

	vm := instances[0].(map[string]interface{})
	assert.Equal(t, "vm-1", vm["id"])
	assert.Equal(t, "vm-26091611258", vm["name"])
	assert.Equal(t, "192.168.37.2", vm["private_ip"])
	assert.Equal(t, "203.0.113.10", vm["public_ip"])
	assert.Equal(t, 1, vm["cpu_number"])
	assert.Equal(t, 1024, vm["memory_mb"])
	assert.Equal(t, "flavor-1", vm["flavor_id"])
	assert.Equal(t, "Small-1", vm["flavor_name"])
	assert.Equal(t, "group-1", vm["instance_group_id"])
	assert.Equal(t, []interface{}{"tag-1"}, vm["tag_ids"])
	assert.Equal(t, "BSS-test_VPC_VMW_03", vm["subnet_name"])
	assert.Equal(t, "", vm["gpu_name"])
	assert.Equal(t, "", vm["ipv6_address"])
}

func testAccInstancesConfig(vpcId string) string {
	return fmt.Sprintf(`
data "fptcloud_instances" "all" {
  vpc_id = %[1]q
}

data "fptcloud_instances" "paged" {
  vpc_id    = %[1]q
  page_size = 1
}

data "fptcloud_instance" "first" {
  vpc_id = %[1]q
  id     = data.fptcloud_instances.all.instances[0].id
}
`, vpcId)
}
