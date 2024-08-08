package fptcloud_instance

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
	common "terraform-provider-fptcloud/commons"
)

// DataSourceInstance function returns a schema.Resource that represents an instance.
// This can be used to query and retrieve details about a specific instance in the infrastructure using its id or name.
func DataSourceInstance() *schema.Resource {
	return &schema.Resource{
		Description: strings.Join([]string{
			"Get information on a instance for use in other resources. This data source provides all of the instance properties as configured on your account.",
			"An error will be raised if the provided instance does not exist in your account.",
		}, "\n\n"),
		ReadContext: dataSourceInstanceRead,
		Schema:      dataSourceInstanceSchema,
	}
}

func dataSourceInstanceRead(_ context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*common.Client)
	instanceService := NewInstanceService(apiClient)

	findInstanceModel := FindInstanceDTO{}

	if id, ok := d.GetOk("id"); ok {
		findInstanceModel.ID = id.(string)
	}

	if name, ok := d.GetOk("name"); ok {
		findInstanceModel.Name = name.(string)
	}

	if vpcId, ok := d.GetOk("vpc_id"); ok {
		findInstanceModel.VpcId = vpcId.(string)
	}

	foundInstance, err := instanceService.Find(findInstanceModel)
	if err != nil {
		return diag.Errorf("[ERR] Failed to retrieve instance: %s", err)
	}

	// Set other attributes
	var setError error
	d.SetId(foundInstance.ID)
	setError = d.Set("vpc_id", foundInstance.VpcId)
	setError = d.Set("name", foundInstance.Name)
	setError = d.Set("guest_os", foundInstance.GuestOs)
	setError = d.Set("host_name", foundInstance.HostName)
	setError = d.Set("status", foundInstance.Status)
	setError = d.Set("private_ip", foundInstance.PrivateIp)
	setError = d.Set("public_ip", foundInstance.PublicIp)
	setError = d.Set("memory_mb", foundInstance.MemoryMb)
	setError = d.Set("cpu_number", foundInstance.CpuNumber)
	setError = d.Set("flavor_id", foundInstance.FlavorId)
	setError = d.Set("subnet_id", foundInstance.SubnetId)
	setError = d.Set("storage_size_gb", foundInstance.StorageSizeGb)
	setError = d.Set("storage_policy", foundInstance.StoragePolicy)
	setError = d.Set("security_group_ids", foundInstance.SecurityGroupIds)
	setError = d.Set("instance_group_id", foundInstance.InstanceGroupId)
	setError = d.Set("created_at", foundInstance.CreatedAt)

	if setError != nil {
		return diag.Errorf("[ERR] Instance could not be found")
	}

	return nil
}
