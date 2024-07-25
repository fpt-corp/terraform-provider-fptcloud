package commons

import "fmt"

var ApiPath = struct {
	SSH                   string
	Storage               func(vpcId string) string
	StorageUpdateAttached func(vpcId string, storageId string) string
	StoragePolicy         func(vpcId string) string
	Flavor                func(vpcId string) string
	Image                 func(vpcId string) string
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
}
