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
		return NotificationMethodListResponse{}, fmt.Errorf("listing notification methods failed: %v", err)
	}

	var result NotificationMethodListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return NotificationMethodListResponse{}, fmt.Errorf("could not parse the notification method list: %v", err)
	}
	return result, nil
}
