package fptcloud_instance_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	fptcloud_instance "terraform-provider-fptcloud/fptcloud/instance"
)

func instanceState() *terraform.InstanceState {
	return &terraform.InstanceState{
		ID: "instance-id",
		Attributes: map[string]string{
			"id":                "instance-id",
			"vpc_id":            "vpc-id",
			"name":              "vm-example",
			"status":            "POWERED_ON",
			"image_name":        "UBUNTU-24.04-26052025",
			"subnet_id":         "subnet-id",
			"storage_size_gb":   "40",
			"storage_policy_id": "policy-1",
			"ssh_key":           "ssh-ed25519 AAAA user@fpt.com",
		},
	}
}

func instanceConfig(storageSizeGb int, storagePolicyId string) *terraform.ResourceConfig {
	return terraform.NewResourceConfigRaw(map[string]interface{}{
		"vpc_id":            "vpc-id",
		"name":              "vm-example",
		"status":            "POWERED_ON",
		"image_name":        "UBUNTU-24.04-26052025",
		"subnet_id":         "subnet-id",
		"storage_size_gb":   storageSizeGb,
		"storage_policy_id": storagePolicyId,
		"ssh_key":           "ssh-ed25519 AAAA user@fpt.com",
	})
}

func TestResourceInstance_RootDiskChangeDoesNotReplaceInstance(t *testing.T) {
	resource := fptcloud_instance.ResourceInstance()

	diff, err := resource.Diff(context.Background(), instanceState(), instanceConfig(80, "policy-2"), nil)
	assert.NoError(t, err)
	assert.NotNil(t, diff)
	assert.False(t, diff.RequiresNew(), "growing the root disk and changing its policy must be an in place update")
	assert.Equal(t, "80", diff.Attributes["storage_size_gb"].New)
	assert.Equal(t, "policy-2", diff.Attributes["storage_policy_id"].New)
}

func TestResourceInstance_ShrinkingRootDiskIsRejected(t *testing.T) {
	resource := fptcloud_instance.ResourceInstance()

	_, err := resource.Diff(context.Background(), instanceState(), instanceConfig(20, "policy-1"), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only be increased")
}

func TestResourceInstance_RootDiskFieldsAreNotForceNew(t *testing.T) {
	resource := fptcloud_instance.ResourceInstance()

	for _, field := range []string{"storage_size_gb", "storage_policy_id"} {
		assert.False(t, resource.Schema[field].ForceNew, "%s must not force a new instance", field)
	}

	// the fields that really need a rebuild keep ForceNew
	for _, field := range []string{"vpc_id", "subnet_id"} {
		assert.True(t, resource.Schema[field].ForceNew, "%s must still force a new instance", field)
	}
}

func instanceConfigWith(overrides map[string]interface{}) *terraform.ResourceConfig {
	raw := map[string]interface{}{
		"vpc_id":            "vpc-id",
		"name":              "vm-example",
		"status":            "POWERED_ON",
		"image_name":        "UBUNTU-24.04-26052025",
		"subnet_id":         "subnet-id",
		"storage_size_gb":   40,
		"storage_policy_id": "policy-1",
		"ssh_key":           "ssh-ed25519 AAAA user@fpt.com",
	}
	for k, v := range overrides {
		if v == nil {
			delete(raw, k)
		} else {
			raw[k] = v
		}
	}
	return terraform.NewResourceConfigRaw(raw)
}

func TestResourceInstance_CreateTimeInputsReplaceTheInstance(t *testing.T) {
	resource := fptcloud_instance.ResourceInstance()

	for name, overrides := range map[string]map[string]interface{}{
		"image":              {"image_name": "UBUNTU-22.04-01012025"},
		"ssh_key":            {"ssh_key": "ssh-ed25519 BBBB user@fpt.com"},
		"switch to password": {"ssh_key": nil, "password": "Passw0rd-1"},
	} {
		diff, err := resource.Diff(context.Background(), instanceState(), instanceConfigWith(overrides), nil)
		assert.NoError(t, err, name)
		if assert.NotNil(t, diff, name) {
			assert.True(t, diff.RequiresNew(), "%s must replace the instance", name)
		}
	}
}

// an imported instance has no value for the inputs the API cannot return: the first plan
// stores the configured one in place instead of replacing the instance
func TestResourceInstance_ImportedInstanceAdoptsCreateTimeInputsInPlace(t *testing.T) {
	resource := fptcloud_instance.ResourceInstance()

	imported := instanceState()
	delete(imported.Attributes, "ssh_key")
	delete(imported.Attributes, "image_name")

	diff, err := resource.Diff(context.Background(), imported, instanceConfigWith(nil), nil)
	assert.NoError(t, err)
	if assert.NotNil(t, diff) {
		assert.False(t, diff.RequiresNew(), "adopting ssh_key and image_name must not replace the instance")
		assert.Equal(t, "ssh-ed25519 AAAA user@fpt.com", diff.Attributes["ssh_key"].New)
		assert.Equal(t, "UBUNTU-24.04-26052025", diff.Attributes["image_name"].New)
	}
}

func TestResourceInstance_SchemaIsValid(t *testing.T) {
	provider := &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"fptcloud_instance": fptcloud_instance.ResourceInstance(),
		},
	}
	assert.NoError(t, provider.InternalValidate())
}
