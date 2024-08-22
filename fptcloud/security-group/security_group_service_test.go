package fptcloud_security_group_test

import (
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
				"status": "active"
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
