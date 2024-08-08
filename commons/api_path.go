package commons

import "fmt"

var ApiPath = struct {
	SSH                        string
	Storage                    func(vpcId string) string
	StorageUpdateAttached      func(vpcId string, storageId string) string
	StoragePolicy              func(vpcId string) string
	Flavor                     func(vpcId string) string
	Image                      func(vpcId string) string
	SecurityGroup              func(vpcId string) string
	RenameSecurityGroup        func(vpcId string, securityGroupId string) string
	UpdateApplyToSecurityGroup func(vpcId string, securityGroupId string) string
	SecurityGroupRule          func(vpcId string, securityGroupRuleId string) string
	CreateSecurityGroupRule    func(vpcId string) string
	Instance                   func(vpcId string) string
	RenameInstance             func(vpcId string, instanceId string) string
	ChangeStatusInstance       func(vpcId string, instanceId string) string
	ResizeInstance             func(vpcId string, instanceId string) string
}{
	SSH: "/v1/user/sshs",
	Storage: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/storage", vpcId)
	},
	StorageUpdateAttached: func(vpcId string, storageId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/storage/%s/update-attached", vpcId, storageId)
	},
	StoragePolicy: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/storage-policies", vpcId)
	},
	Flavor: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/flavors", vpcId)
	},
	Image: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/images", vpcId)
	},
	SecurityGroup: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/security-group", vpcId)
	},
	RenameSecurityGroup: func(vpcId string, securityGroupId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/security-group/%s/rename", vpcId, securityGroupId)
	},
	UpdateApplyToSecurityGroup: func(vpcId string, securityGroupId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/security-group/%s/apply-to", vpcId, securityGroupId)
	},
	SecurityGroupRule: func(vpcId string, securityGroupRuleId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/security-group-rule/%s", vpcId, securityGroupRuleId)
	},
	CreateSecurityGroupRule: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/security-group-rule", vpcId)
	},
	Instance: func(vpcId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/instance", vpcId)
	},
	RenameInstance: func(vpcId string, instanceId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/instance/%s/rename", vpcId, instanceId)
	},
	ChangeStatusInstance: func(vpcId string, instanceId string) string {
		return fmt.Sprintf("/v1/terraform/vpc/%s/instance/%s/change-status", vpcId, instanceId)
	},
	ResizeInstance: func(vpcId string, instanceId string) string {
		return fmt.Sprintf("/v1/vmware/vpc/%s/compute/instance/%s/reconfigure-vm", vpcId, instanceId)
	},
}
