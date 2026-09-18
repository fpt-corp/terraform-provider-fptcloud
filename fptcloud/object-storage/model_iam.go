package fptcloud_object_storage

import "encoding/json"

// IamUser is one IAM user row as the API exposes it. The backend normalises
// Ceph's PascalCase IAM fields into these names and always serialises
// created_at as an ISO-8601 string, so no date parsing is needed here.
type IamUser struct {
	UserName  string `json:"user_name"`
	UserID    string `json:"user_id,omitempty"`
	Arn       string `json:"arn,omitempty"`
	Path      string `json:"path,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type IamUserRequest struct {
	UserName string `json:"user_name"`
}

type CreateIamUserResponse struct {
	Status  bool    `json:"status"`
	Message string  `json:"message,omitempty"`
	IamUser IamUser `json:"iam_user"`
}

// IamUserDetail is what the detail endpoint answers with. It is the user row
// plus the summary of what hangs off it. Access key ids are deliberately not
// part of it - the backend exposes only their count here.
type IamUserDetail struct {
	IamUser
	AccessKeyCount int      `json:"access_key_count"`
	HasPolicy      bool     `json:"has_policy"`
	PolicyNames    []string `json:"policy_names"`
}

type IamUserListResponse struct {
	IamUsers []IamUser `json:"iam_users"`
	Total    int       `json:"total"`
}

// IamUserAccessKey is one credential row. SecretAccessKey is populated only by
// the create call - no endpoint can read it back afterwards.
type IamUserAccessKey struct {
	AccessKeyID     string `json:"access_key_id"`
	UserName        string `json:"user_name,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	CreatedBy       string `json:"created_by,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
}

type IamUserAccessKeyListResponse struct {
	AccessKeys []IamUserAccessKey `json:"access_keys"`
	Total      int                `json:"total"`
}

type CreateIamUserAccessKeyResponse struct {
	Status    bool             `json:"status"`
	Message   string           `json:"message,omitempty"`
	AccessKey IamUserAccessKey `json:"access_key"`
}

// IamPolicyResponse covers the read of an inline policy on either an IAM user
// or an IAM role - the two endpoints answer with the same shape under a
// different name field.
//
// Policy and Template stay as raw JSON: Terraform carries the document as a
// string, and re-encoding it through a Go struct would silently drop any
// statement field this provider does not model.
type IamPolicyResponse struct {
	UserName   string          `json:"user_name,omitempty"`
	RoleName   string          `json:"role_name,omitempty"`
	PolicyName string          `json:"policy_name"`
	Policy     json.RawMessage `json:"policy"`
	HasPolicy  bool            `json:"has_policy"`
	Template   json.RawMessage `json:"template,omitempty"`
}

// IamPolicyRequest is the write payload. The document is sent as a JSON object,
// not as a quoted string, which is the form the backend validator expects.
type IamPolicyRequest struct {
	Policy json.RawMessage `json:"policy"`
}

// IamRole is one role row. TrustedUsers are plain IAM user names, parsed by the
// backend out of the role's trust policy - the ARNs never surface here.
type IamRole struct {
	RoleName     string   `json:"role_name"`
	RoleID       string   `json:"role_id,omitempty"`
	Arn          string   `json:"arn,omitempty"`
	Path         string   `json:"path,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	TrustedUsers []string `json:"trusted_users"`
}

type IamRoleRequest struct {
	RoleName     string   `json:"role_name"`
	TrustedUsers []string `json:"trusted_users"`
}

type CreateIamRoleResponse struct {
	Status  bool    `json:"status"`
	Message string  `json:"message,omitempty"`
	IamRole IamRole `json:"iam_role"`
}

type IamRoleListResponse struct {
	IamRoles []IamRole `json:"iam_roles"`
	Total    int       `json:"total"`
}

type IamRoleTrustedUsersRequest struct {
	TrustedUsers []string `json:"trusted_users"`
}

// IamRoleTrustedUsersResponse reports the trust policy that was actually
// written. The backend skips a name it cannot resolve instead of failing the
// call, so this list can be shorter than the one that was sent.
type IamRoleTrustedUsersResponse struct {
	Status       bool     `json:"status"`
	Message      string   `json:"message,omitempty"`
	RoleName     string   `json:"role_name"`
	TrustedUsers []string `json:"trusted_users"`
}
