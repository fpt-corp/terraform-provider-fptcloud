package fptcloud_security_group_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"terraform-provider-fptcloud/fptcloud/security-group"
	"testing"

	"github.com/stretchr/testify/assert"
	common "terraform-provider-fptcloud/commons"
)

func TestFindSecurityGroup_ReturnsSecurityGroup(t *testing.T) {
	mockResponse := `{ 
		"data": {
				"vpc_id": "12345678-aaaa-bbbb-cccc-123456789012",
				"id": "87654321-bbbb-cccc-dddd-210987654321",
				"name": "example-security-group",
				"edge_gateway_id": "11223344-dddd-eeee-ffff-443322110011",
				"firewall_type": "application",
				"apply_to": ["ip_instance"],
				"rules": [
					{
						"id": "abcd1234-5678-90ef-ghij-klmnopqrstuv",
						"direction": "inbound",
						"action": "allow",
						"protocol": "tcp",
						"port_range": "22",
						"sources": "ALL",
						"ip_type": "ipv4",
						"description": "Allow SSH access",
						"status": "active"
					},
					{
						"id": "wxyz9876-5432-10ef-ghij-lmnopqrstuvw",
						"direction": "outbound",
						"action": "allow",
						"protocol": "tcp",
						"port_range": "80",
						"sources": "0.0.0.0/0",
						"ip_type": "ipv4",
						"description": "Allow HTTP access",
						"status": "active"
					}
				],
				"created_at": "2024-01-01T00:00:00",
				"status": "active",
				"tag_ids": ["tag-1"]
			}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group": mockResponse,
	})
	defer server.Close()
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)
	searchModel := fptcloud_security_group.FindSecurityGroupDTO{VpcId: "vpc_id", Name: "example-security-group"}
	securityGroup, err := service.Find(searchModel)
	assert.NoError(t, err)
	assert.NotNil(t, securityGroup)
	assert.Equal(t, "87654321-bbbb-cccc-dddd-210987654321", securityGroup.ID)
	assert.Equal(t, "example-security-group", securityGroup.Name)
}

func TestFindSecurityGroup_ReturnsErrorOnRequestFailure(t *testing.T) {
	mockResponse := `invalid`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group": mockResponse,
	})
	defer server.Close()
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)
	searchModel := fptcloud_security_group.FindSecurityGroupDTO{VpcId: "vpc_id", Name: "security-group-name"}
	securityGroup, err := service.Find(searchModel)
	assert.Error(t, err)
	assert.Nil(t, securityGroup)
}

func TestCreateSecurityGroup_ReturnsSecurityGroupId(t *testing.T) {
	mockResponse := `{"security_group_id": "security_group_id"}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group": mockResponse,
	})
	defer server.Close()
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)
	createModel := fptcloud_security_group.CreatedSecurityGroupDTO{VpcId: "vpc_id", Name: "security-group-name"}
	securityGroupId, err := service.Create(createModel)
	assert.NoError(t, err)
	assert.Equal(t, "security_group_id", securityGroupId)
}

func TestDeleteSecurityGroup_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group": "",
	})
	defer server.Close()
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)
	response, err := service.Delete("vpc_id", "security-group-name")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestRenameSecurityGroup_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group": "",
	})
	defer server.Close()
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)
	response, err := service.Rename("vpc_id", "security-group-name", "new-name")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestUpdateApplyToSecurityGroup_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group": "",
	})
	defer server.Close()
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)
	response, err := service.UpdateApplyTo("vpc_id", "security_id", []string{"ip"})
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestUpdateSecurityGroupTags_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group/security_group_id/tags": "",
	})
	defer server.Close()
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)
	response, err := service.UpdateTags("vpc_id", "security_group_id", []string{"tag-1"})
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}

func TestListAllSecurityGroups_UsesListShape(t *testing.T) {
	mockResponse := `{
		"total": 1,
		"data": [
			{
				"id": "ad73e655-064c-420f-b7e1-544891d8e50d",
				"name": "default",
				"display_name": "Default security group",
				"created_at": "2024-08-03T19:24:25",
				"updated_at": "2026-10-05T08:15:42",
				"edge_gateway": {
					"name": "BSS-TEST_VPC_OSP02",
					"id": "f096ff0f-58ab-4140-bf8d-8e255363c2ce",
					"vdc_group_id": null,
					"vdc_group_name": null,
					"edge_gateway_id": "ce4d475a-7426-40a5-80c9-d8ff7ac0d4a6"
				},
				"is_vdc_group": false,
				"rules": [],
				"status": "REALIZED",
				"firewall_group_id": "6bb3cc3fbb7d459785d059c7f32cf74c",
				"has_firewall_ip_address": true,
				"has_firewall_rules": false,
				"firewall_type": "ACL",
				"tags": [{"id": "tag-1", "key": "env", "value": "prod", "color": "red"}],
				"type": "IP_SET",
				"ip_addresses": ["10.7.9.247"],
				"instances": [{"id": "vm-1", "name": "vm-1", "status": "POWERED_ON", "ip_address": "10.7.9.247", "ipv6_address": null, "type": "instance"}]
			}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/security-groups": mockResponse,
	})
	defer server.Close()
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)

	groups, err := service.ListAll(fptcloud_security_group.SecurityGroupListDTO{VpcId: "vpc_id", PageSize: 25})
	assert.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.Equal(t, "ad73e655-064c-420f-b7e1-544891d8e50d", groups[0].ID)
	assert.Equal(t, "Default security group", *groups[0].DisplayName)
	assert.Equal(t, "ACL", groups[0].FirewallType)
	assert.Equal(t, []string{"10.7.9.247"}, groups[0].IpAddresses)
	assert.Equal(t, "tag-1", groups[0].Tags[0].ID)
	assert.Equal(t, "instance", groups[0].Instances[0].Type)
}

func TestListAllSecurityGroups_EmptyList(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/security-groups": `{"total": 0, "data": []}`,
	})
	defer server.Close()
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)

	groups, err := service.ListAll(fptcloud_security_group.SecurityGroupListDTO{VpcId: "vpc_id", PageSize: 25})
	assert.NoError(t, err)
	assert.Empty(t, groups)
}

func TestListAllSecurityGroups_WalksMultiplePages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("page") == "1" {
			data := "["
			for i := 0; i < 25; i++ {
				if i > 0 {
					data += ","
				}
				data += fmt.Sprintf(`{"id":"sg-p1-%d","name":"group-%d","rules":[]}`, i, i)
			}
			data += "]"
			_, _ = rw.Write([]byte(fmt.Sprintf(`{"total":30,"data":%s}`, data)))
			return
		}
		_, _ = rw.Write([]byte(`{"total":30,"data":[{"id":"sg-p2-0","name":"group-last","rules":[]}]}`))
	}))
	defer server.Close()
	mockClient, _ := common.NewClientForTestingWithServer(server)
	service := fptcloud_security_group.NewSecurityGroupService(mockClient)

	groups, err := service.ListAll(fptcloud_security_group.SecurityGroupListDTO{VpcId: "vpc_id", PageSize: 25})
	assert.NoError(t, err)
	assert.Len(t, groups, 26)
	assert.Equal(t, "sg-p1-0", groups[0].ID)
	assert.Equal(t, "sg-p2-0", groups[25].ID)
}

func TestListAllSecurityGroups_RejectsOutOfRangePageSize(t *testing.T) {
	service := fptcloud_security_group.NewSecurityGroupService(nil)

	_, err := service.ListAll(fptcloud_security_group.SecurityGroupListDTO{VpcId: "vpc_id", PageSize: 0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "page_size")

	_, err = service.ListAll(fptcloud_security_group.SecurityGroupListDTO{VpcId: "vpc_id", PageSize: 26})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "page_size")
}
