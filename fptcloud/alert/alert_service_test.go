package fptcloud_alert_test

import (
	"testing"

	common "terraform-provider-fptcloud/commons"
	alert "terraform-provider-fptcloud/fptcloud/alert"

	"github.com/stretchr/testify/assert"
)

func TestListNotificationMethods(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/alert/alarm-notification/list": `{
			"data": [
				{"id": "n-1", "name": "Ops", "type": "EMAIL", "address": "ops@corp.vn", "level": "VPC"},
				{"id": "n-2", "name": "Slack", "type": "SLACK", "address": "https://hooks", "level": "VPC"}
			]
		}`,
	})
	defer server.Close()

	resp, err := alert.NewAlertService(mockClient).ListNotificationMethods("vpc-1", "VPC")
	assert.Nil(t, err)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, "ops@corp.vn", resp.Data[0].Address)
	assert.Equal(t, "EMAIL", resp.Data[0].Type)
}

func TestListNotificationMethodsEmptyIsNotAnError(t *testing.T) {
	// Tenant chưa cấu hình kênh nào - trả rỗng, không phải lỗi provider.
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc-1/alert/alarm-notification/list": `{"data": []}`,
	})
	defer server.Close()

	resp, err := alert.NewAlertService(mockClient).ListNotificationMethods("vpc-1", "VPC")
	assert.Nil(t, err)
	assert.Empty(t, resp.Data)
}

func TestDataSourceShape(t *testing.T) {
	ds := alert.DataSourceAlertNotificationMethods()
	assert.NotNil(t, ds.ReadContext)
	assert.Nil(t, ds.InternalValidate(nil, false))
}
