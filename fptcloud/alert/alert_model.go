package fptcloud_alert

// NotificationMethod là một kênh nhận thông báo đã cấu hình cho VPC.
//
// Endpoint gộp kết quả theo address (GROUP BY address, lấy max(id)), nên
// ĐỊA CHỈ là khoá duy nhất trong kết quả, còn Name chỉ là max(name) của nhóm.
// Lọc theo Address, đừng lọc theo Name.
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
