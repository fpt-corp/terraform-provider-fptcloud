package fptcloud_security_group_rule_test

import (
	"terraform-provider-fptcloud/fptcloud/security-group-rule"
	"testing"

	"github.com/stretchr/testify/assert"
	common "terraform-provider-fptcloud/commons"
)

func TestFindSecurityGroupRule_ReturnsSecurityGroupRule(t *testing.T) {
	mockResponse := `{
		"data": {
			"id": "622ae917-febe-42b3-91c4-724d92466519",
			"direction": "INGRESS",
			"action": "ALLOW",
			"protocol": "TCP",
			"port_range": "80",
			"sources": [
				"0.0.0.0/0"
			],
			"ip_type": "IPV4",
			"description": "allow http",
			"status": "ACTIVE",
			"vpc_id": "vpc_id",
			"security_group_id": ""
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group-rule": mockResponse,
	})
	defer server.Close()
	service := fptcloud_security_group_rule.NewSecurityGroupRuleService(mockClient)
	rule, err := service.Find("vpc_id", "622ae917-febe-42b3-91c4-724d92466519")
	assert.NoError(t, err)
	assert.NotNil(t, rule)
	assert.Equal(t, "622ae917-febe-42b3-91c4-724d92466519", rule.ID)
	assert.Equal(t, "INGRESS", rule.Direction)
	assert.Equal(t, "ALLOW", rule.Action)
	assert.Equal(t, "TCP", rule.Protocol)
	assert.Equal(t, "80", rule.PortRange)
	assert.Equal(t, []string{"0.0.0.0/0"}, rule.Sources)
	assert.Equal(t, "IPV4", rule.IpType)
	assert.Equal(t, "allow http", rule.Description)
	assert.Equal(t, "ACTIVE", rule.Status)
	assert.Equal(t, "vpc_id", rule.VpcId)
	assert.Equal(t, "", rule.SecurityGroupId)
}

func TestFindSecurityGroupRule_ReturnsErrorOnRequestFailure(t *testing.T) {
	mockResponse := `invalid`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group-rule": mockResponse,
	})
	defer server.Close()
	service := fptcloud_security_group_rule.NewSecurityGroupRuleService(mockClient)
	rule, err := service.Find("vpc_id", "rule_id")
	assert.Error(t, err)
	assert.Nil(t, rule)
}

func TestCreateSecurityGroupRule_ReturnsSecurityGroupRuleId(t *testing.T) {
	mockResponse := `{"security_group_rule_id": "security_group_rule_id"}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group-rule": mockResponse,
	})
	defer server.Close()
	service := fptcloud_security_group_rule.NewSecurityGroupRuleService(mockClient)
	createModel := fptcloud_security_group_rule.CreateSecurityGroupRuleDto{
		Direction:       "INGRESS",
		Action:          "ALLOW",
		Protocol:        "TCP",
		PortRange:       "80",
		Sources:         []string{"0.0.0.0/0"},
		Description:     nil,
		SecurityGroupId: "",
	}
	ruleId, err := service.Create("vpc_id", createModel)
	assert.NoError(t, err)
	assert.Equal(t, "security_group_rule_id", ruleId)
}

func TestDeleteSecurityGroupRule_ReturnsSuccess(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v2/vpc/vpc_id/security-group-rule": "",
	})
	defer server.Close()
	service := fptcloud_security_group_rule.NewSecurityGroupRuleService(mockClient)
	response, err := service.Delete("vpc_id", "rule_id")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Successfully", response.Data)
}
