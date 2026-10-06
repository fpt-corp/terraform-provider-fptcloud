package fptcloud_storage_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	common "terraform-provider-fptcloud/commons"
	fptcloud_storage "terraform-provider-fptcloud/fptcloud/storage"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"

	test_helper "terraform-provider-fptcloud/commons/test-helper"
)

// TestAccFptCloudStorages_basic lists the storages of VPC_ID twice — once in a
// single page and once one storage per page — and checks both return the same
// storages, and that the first one matches the fptcloud_storage data source.
// VPC_ID must contain at least one storage.
func TestAccFptCloudStorages_basic(t *testing.T) {
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
				Config: testAccStoragesConfig(vpcId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.fptcloud_storages.all", "id", vpcId),
					resource.TestCheckResourceAttr("data.fptcloud_storages.all", "page_size", "25"),
					resource.TestCheckResourceAttrSet("data.fptcloud_storages.all", "storages.0.id"),
					testAccCheckSameStorages("data.fptcloud_storages.all", "data.fptcloud_storages.paged"),
					resource.TestCheckResourceAttrPair("data.fptcloud_storages.all", "storages.0.name", "data.fptcloud_storage.first", "name"),
					resource.TestCheckResourceAttrPair("data.fptcloud_storages.all", "storages.0.type", "data.fptcloud_storage.first", "type"),
					resource.TestCheckResourceAttrPair("data.fptcloud_storages.all", "storages.0.size_gb", "data.fptcloud_storage.first", "size_gb"),
					resource.TestCheckResourceAttrPair("data.fptcloud_storages.all", "storages.0.storage_policy_id", "data.fptcloud_storage.first", "storage_policy_id"),
					resource.TestCheckResourceAttrPair("data.fptcloud_storages.all", "storages.0.instance_id", "data.fptcloud_storage.first", "instance_id"),
					resource.TestCheckResourceAttrPair("data.fptcloud_storages.all", "storages.0.created_at", "data.fptcloud_storage.first", "created_at"),
				),
			},
		},
	})
}

// testAccCheckSameStorages checks that two fptcloud_storages data sources hold the same storages in the same order.
func testAccCheckSameStorages(a, b string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		left, ok := s.RootModule().Resources[a]
		if !ok {
			return fmt.Errorf("%s not found in state", a)
		}
		right, ok := s.RootModule().Resources[b]
		if !ok {
			return fmt.Errorf("%s not found in state", b)
		}
		count := left.Primary.Attributes["storages.#"]
		if count != right.Primary.Attributes["storages.#"] {
			return fmt.Errorf("%s has %s storages, %s has %s", a, count, b, right.Primary.Attributes["storages.#"])
		}
		var n int
		_, _ = fmt.Sscan(count, &n)
		for i := 0; i < n; i++ {
			key := fmt.Sprintf("storages.%d.id", i)
			if left.Primary.Attributes[key] != right.Primary.Attributes[key] {
				return fmt.Errorf("%s differs: %q vs %q", key, left.Primary.Attributes[key], right.Primary.Attributes[key])
			}
		}
		return nil
	}
}

func TestDataSourceStorages_SchemaIsValid(t *testing.T) {
	r := fptcloud_storage.DataSourceStorages()
	assert.NoError(t, r.InternalValidate(nil, false))
	assert.Equal(t, 25, r.Schema["page_size"].Default)
}

func TestDataSourceStorages_ReadMapsListToSingularFields(t *testing.T) {
	// Trimmed from a production /storages response: size is in MB, the
	// display name is optional, encrypted is "0"/"1"/null and tags are objects.
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/storages": `{"total": 2, "data": [
			{"id": "st-1", "vpc_id": "vpc_id", "name": "tf-disk-01", "display_name": null, "description": null,
			 "size": 56320, "status": "ENABLED", "vm_id": "vm-1", "vm_name": "vm-26091611258",
			 "storage_type": "EXTERNAL", "storage_policy_id": "policy-1", "storage_policy_name": "Premium-SSD",
			 "policy_uuid": "uuid-1", "disk_id": null, "encrypted": "0", "zone_id": null,
			 "created_at": "2026-10-01T06:58:21", "has_snapshot": false, "can_modify": true,
			 "tags": [{"id": "tag-1", "key": "env", "value": "prod", "color": "red"}]},
			{"id": "st-2", "vpc_id": "vpc_id", "name": "disk-infra-name", "display_name": "data-disk",
			 "size": 45057, "status": "ENABLED", "vm_id": null, "vm_name": null,
			 "storage_type": "LOCAL", "storage_policy_id": "policy-1", "storage_policy_name": "Premium-SSD",
			 "disk_id": "disk-2", "encrypted": null, "zone_id": "zone-1",
			 "created_at": "2026-09-25T14:32:32", "tags": []}
		]}`,
	})
	defer server.Close()

	r := fptcloud_storage.DataSourceStorages()
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{"vpc_id": "vpc_id"})
	diags := r.ReadContext(context.Background(), d, mockClient)
	assert.False(t, diags.HasError(), "%v", diags)

	assert.Equal(t, "vpc_id", d.Id())
	storages := d.Get("storages").([]interface{})
	assert.Len(t, storages, 2)

	first := storages[0].(map[string]interface{})
	assert.Equal(t, "tf-disk-01", first["name"])
	assert.Equal(t, 55, first["size_gb"])
	assert.Equal(t, "EXTERNAL", first["type"])
	assert.Equal(t, "Premium-SSD", first["storage_policy"])
	assert.Equal(t, "policy-1", first["storage_policy_id"])
	assert.Equal(t, "vm-1", first["instance_id"])
	assert.Equal(t, "vm-26091611258", first["instance_name"])
	assert.Equal(t, []interface{}{"tag-1"}, first["tag_ids"])
	assert.Equal(t, false, first["encrypted"])

	second := storages[1].(map[string]interface{})
	assert.Equal(t, "data-disk", second["name"], "the display name wins, as in fptcloud_storage")
	assert.Equal(t, 45, second["size_gb"], "a partial GB rounds up, as in fptcloud_storage")
	assert.Equal(t, "", second["instance_id"])
	assert.Equal(t, "zone-1", second["zone_id"])
}

func testAccStoragesConfig(vpcId string) string {
	return fmt.Sprintf(`
data "fptcloud_storages" "all" {
  vpc_id = %[1]q
}

data "fptcloud_storages" "paged" {
  vpc_id    = %[1]q
  page_size = 1
}

data "fptcloud_storage" "first" {
  vpc_id = %[1]q
  id     = data.fptcloud_storages.all.storages[0].id
}
`, vpcId)
}
