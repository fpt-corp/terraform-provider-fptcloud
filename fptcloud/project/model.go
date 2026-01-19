package fptcloud_project

type Project struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Response struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type FindProjectResponse struct {
	Response
	Data []Project `json:"data,omitempty"`
}

type FindProjectParam struct {
	Name string `json:"name,omitempty"`
}

