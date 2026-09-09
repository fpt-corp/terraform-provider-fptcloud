package fptcloud_alert

// NotificationMethod is a notification channel configured for the VPC.
//
// The endpoint groups its results by address (GROUP BY address, taking
// max(id)), so ADDRESS is the unique key in the response while Name is only
// the max(name) of the group. Match on Address, not on Name.
type NotificationMethod struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Address string `json:"address"`
	Level   string `json:"level"`
}

type NotificationMethodListResponse struct {
	Data []NotificationMethod `json:"data"`
}
