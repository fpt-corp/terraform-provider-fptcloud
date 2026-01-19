package fptcloud_user

type Response struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type User struct {
	Id    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
}

type ListUsersResponse struct {
	Response
	Data struct {
		Total int    `json:"total,omitempty"`
		Data  []User `json:"data,omitempty"`
	} `json:"data,omitempty"`
}
