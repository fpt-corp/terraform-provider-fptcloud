package fptcloud_instance

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
}

type FlavorDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InstanceStorageModel is one disk of an instance as persisted by the portal
type InstanceStorageModel struct {
	ID              string `json:"id"`
	DiskId          string `json:"disk_id"`
	StorageType     string `json:"storage_type"`
	SizeMb          int    `json:"size"`
	Status          string `json:"status"`
	StoragePolicyId string `json:"storage_policy_id"`
	PolicyUuid      string `json:"policy_uuid"`
}

// InstanceStorageInfraModel is one disk of an instance as reported by the infrastructure
type InstanceStorageInfraModel struct {
	DiskName           *string `json:"disk_name"`
	DiskId             string  `json:"disk_id"`
	SizeMb             int     `json:"size_mb"`
	StorageProfileName string  `json:"storage_profile_name"`
	IsRoot             bool    `json:"is_root"`
	StorageType        string  `json:"storage_type"`
}

// RootStorageModel is the boot disk, whichever listing it could be read from
type RootStorageModel struct {
	DiskId            string
	SizeMb            int
	StoragePolicyId   string
	StoragePolicyName string
}

type ResizeRootDiskDTO struct {
	DiskId           string  `json:"disk_id"`
	IncreaseInSizeMb int     `json:"increase_in_size_mb"`
	StoragePolicyId  *string `json:"storage_policy_id,omitempty"`
}

type StoragePolicyDTO struct {
	ID      string `json:"id"`
	InfraId string `json:"infra_id"`
	Name    string `json:"name"`
}
