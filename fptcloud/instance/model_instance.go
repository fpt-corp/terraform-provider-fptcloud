package fptcloud_instance

import "strings"

type FindInstanceDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	VpcId string `json:"vpc_id"`
}
type InstanceModel struct {
	VpcId            string   `json:"vpc_id"`
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	GuestOs          string   `json:"guest_os"`
	HostName         string   `json:"host_name"`
	Status           string   `json:"status"`
	PrivateIp        string   `json:"private_ip"`
	PublicIp         *string  `json:"public_ip,omitempty"`
	MemoryMb         int      `json:"memory_mb"`
	CpuNumber        int      `json:"cpu_number"`
	FlavorId         *string  `json:"flavor_id,omitempty"`
	FlavorName       *string  `json:"flavor_name,omitempty"`
	SubnetId         string   `json:"subnet_id"`
	StorageSizeGb    int      `json:"storage_size_gb"`
	StoragePolicy    string   `json:"storage_policy"`
	StoragePolicyId  string   `json:"storage_policy_id"`
	SecurityGroupIds []string `json:"security_group_ids,omitempty"`
	InstanceGroupId  *string  `json:"instance_group_id,omitempty"`
	CreatedAt        string   `json:"created_at"`
	TagIds           []string `json:"tag_ids,omitempty"`
	GpuName          *string  `json:"gpu_name,omitempty"`
	BillingType      *string  `json:"billing_type,omitempty"`
}

type CreateInstanceDTO struct {
	VpcId            string   `json:"vpc_id"`
	Name             string   `json:"name"`
	PrivateIp        *string  `json:"private_ip,omitempty"`
	PublicIp         *string  `json:"public_ip,omitempty"`
	FlavorName       string   `json:"flavor_name"`
	ImageName        string   `json:"image_name"`
	SubnetId         string   `json:"subnet_id"`
	StorageSizeGb    int      `json:"storage_size_gb"`
	StoragePolicyId  string   `json:"storage_policy_id"`
	SecurityGroupIds []string `json:"security_group_ids,omitempty"`
	InstanceGroupId  *string  `json:"instance_group_id,omitempty"`
	SshKey           *string  `json:"ssh_key,omitempty"`
	Password         *string  `json:"password,omitempty"`
	TagIds           []string `json:"tag_ids,omitempty"`
	BillingType      *string  `json:"billing_type,omitempty"`
}

type FlavorDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// mapGpuPlanToBillingType maps the Terraform-facing gpu_plan value to the API's billing_type value.
func mapGpuPlanToBillingType(plan string) string {
	switch plan {
	case "hold":
		return "reserved"
	case "detach":
		return "payg"
	default:
		return ""
	}
}

// mapBillingTypeToGpuPlan maps the API's billing_type value back to the Terraform-facing gpu_plan value.
func mapBillingTypeToGpuPlan(billingType *string) string {
	if billingType == nil {
		return ""
	}
	switch strings.ToLower(*billingType) {
	case "reserved":
		return "hold"
	case "payg":
		return "detach"
	default:
		return ""
	}
}

// deriveVmType derives the Terraform-facing vm_type ("cpu"/"gpu") from the server's gpu_name field.
func deriveVmType(gpuName *string) string {
	if gpuName != nil && *gpuName != "" {
		return "gpu"
	}
	return "cpu"
}
