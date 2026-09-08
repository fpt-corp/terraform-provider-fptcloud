package fptcloud_alert

import (
	"encoding/json"
	"fmt"

	common "terraform-provider-fptcloud/commons"
)

type AlertService interface {
	ListNotificationMethods(vpcId string, level string) (NotificationMethodListResponse, error)
}

type alertServiceImpl struct {
	client *common.Client
}

func NewAlertService(client *common.Client) AlertService {
	return &alertServiceImpl{client: client}
}

func (s *alertServiceImpl) ListNotificationMethods(vpcId string, level string) (NotificationMethodListResponse, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.AlertNotificationMethods(vpcId, level))
	if err != nil {
		return NotificationMethodListResponse{}, fmt.Errorf("liệt kê kênh thông báo thất bại: %v", err)
	}

	var result NotificationMethodListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return NotificationMethodListResponse{}, fmt.Errorf("không đọc được danh sách kênh thông báo: %v", err)
	}
	return result, nil
}
