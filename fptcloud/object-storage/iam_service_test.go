package fptcloud_object_storage_test

import (
	"testing"

	common "terraform-provider-fptcloud/commons"
	fptcloud_object_storage "terraform-provider-fptcloud/fptcloud/object-storage"

	"github.com/stretchr/testify/assert"
)

func TestCreateIamUserReturnsTheCreatedUserWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"iam_user": {
			"user_name": "alice",
			"user_id": "AAAABBBBCCCCDDDD",
			"arn": "arn:aws:iam:::user/alice",
			"path": "/",
			"created_at": "2026-09-17T03:21:00+00:00"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/create": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.CreateIamUser("vpc_id", "s3_service_id", "alice")

	assert.NotNil(t, resp)
	assert.True(t, resp.Status)
	assert.Equal(t, "alice", resp.IamUser.UserName)
	assert.Equal(t, "AAAABBBBCCCCDDDD", resp.IamUser.UserID)
	assert.Equal(t, "arn:aws:iam:::user/alice", resp.IamUser.Arn)
	assert.Equal(t, "2026-09-17T03:21:00+00:00", resp.IamUser.CreatedAt)
}

func TestCreateIamUserReportsTheApiMessageWhenTheNameIsTaken(t *testing.T) {
	// The backend answers a taken name with a 409 carrying status:false. Reading
	// success from the HTTP layer alone would record a user this apply did not
	// create.
	mockResponse := `{
		"status": false,
		"message": "User name already exists"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/create": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.CreateIamUser("vpc_id", "s3_service_id", "alice")

	assert.NotNil(t, resp)
	assert.False(t, resp.Status)
	assert.Equal(t, "User name already exists", resp.Message)
	assert.Equal(t, "", resp.IamUser.UserName)
}

func TestGetIamUserReturnsTheDetailWhenFound(t *testing.T) {
	mockResponse := `{
		"user_name": "alice",
		"user_id": "AAAABBBBCCCCDDDD",
		"arn": "arn:aws:iam:::user/alice",
		"path": "/",
		"created_at": "2026-09-17T03:21:00+00:00",
		"access_key_count": 2,
		"has_policy": true,
		"policy_names": ["portal-managed-policy"]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/detail": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	detail := service.GetIamUser("vpc_id", "s3_service_id", "alice")

	assert.NotNil(t, detail)
	assert.Equal(t, "alice", detail.UserName)
	assert.Equal(t, "arn:aws:iam:::user/alice", detail.Arn)
	assert.Equal(t, 2, detail.AccessKeyCount)
	assert.True(t, detail.HasPolicy)
	assert.Equal(t, []string{"portal-managed-policy"}, detail.PolicyNames)
}

func TestGetIamUserReturnsNilWhenTheApiReportsAFailure(t *testing.T) {
	// The detail endpoint answers with the user object itself, so a failure
	// envelope is the only signal that nothing was found. Returning an empty
	// detail instead of nil would let a read record a user that is not there.
	mockResponse := `{
		"status": false,
		"message": "IAM user not found"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/ghost/detail": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	assert.Nil(t, service.GetIamUser("vpc_id", "s3_service_id", "ghost"))
}

func TestListIamUsersReturnsThePageWhenSuccess(t *testing.T) {
	mockResponse := `{
		"iam_users": [
			{"user_name": "alice", "user_id": "AAAA", "arn": "arn:aws:iam:::user/alice", "created_at": "2026-09-17T03:21:00+00:00"},
			{"user_name": "bob", "user_id": "BBBB", "arn": "arn:aws:iam:::user/bob", "created_at": "2026-09-16T03:21:00+00:00"}
		],
		"total": 2
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/list": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	list, err := service.ListIamUsers("vpc_id", "s3_service_id", 1, 25)

	assert.NoError(t, err)
	assert.Equal(t, 2, list.Total)
	assert.Len(t, list.IamUsers, 2)
	assert.Equal(t, "alice", list.IamUsers[0].UserName)
	assert.Equal(t, "bob", list.IamUsers[1].UserName)
}

func TestDeleteIamUserSucceedsAndReportsWhatTheCleanupRemoved(t *testing.T) {
	// Deleting a user cascades: keys are revoked and policies removed first.
	// The report is what a destroy has to be able to surface.
	mockResponse := `{
		"status": true,
		"user_name": "alice",
		"access_keys_revoked": ["AKIAEXAMPLE1"],
		"policies_deleted": ["portal-managed-policy"],
		"managed_policies_detached": []
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/delete": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	err := service.DeleteIamUser("vpc_id", "s3_service_id", "alice")

	assert.NoError(t, err)
}

func TestDeleteIamUserReturnsTheApiMessageWhenTheCleanupFails(t *testing.T) {
	mockResponse := `{
		"status": false,
		"message": "Failed to revoke access keys for IAM user alice"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/delete": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	err := service.DeleteIamUser("vpc_id", "s3_service_id", "alice")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to revoke access keys for IAM user alice")
}

func TestListIamUserAccessKeysReturnsTheKeysWithoutSecrets(t *testing.T) {
	// The listing endpoint can never return a secret - it is revealed once, at
	// create. A row here carries only the id and its provenance.
	mockResponse := `{
		"access_keys": [
			{"access_key_id": "AKIAEXAMPLE1", "user_name": "alice", "created_at": "2026-09-17T03:21:00+00:00", "created_by": "lam.vt"},
			{"access_key_id": "AKIAEXAMPLE2", "user_name": "alice", "created_at": "2026-09-16T03:21:00+00:00", "created_by": ""}
		],
		"total": 2
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/access-keys/list": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	list, err := service.ListIamUserAccessKeys("vpc_id", "s3_service_id", "alice")

	assert.NoError(t, err)
	assert.Equal(t, 2, list.Total)
	assert.Equal(t, "AKIAEXAMPLE1", list.AccessKeys[0].AccessKeyID)
	assert.Equal(t, "lam.vt", list.AccessKeys[0].CreatedBy)
	assert.Equal(t, "AKIAEXAMPLE2", list.AccessKeys[1].AccessKeyID)
}

func TestCreateIamUserAccessKeyReturnsTheSecretOnlyTimeItIsAvailable(t *testing.T) {
	mockResponse := `{
		"status": true,
		"access_key": {
			"access_key_id": "AKIAEXAMPLE1",
			"user_name": "alice",
			"created_at": "2026-09-17T03:21:00+00:00",
			"secret_access_key": "s3cr3t-never-readable-again"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/access-keys/create": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	created := service.CreateIamUserAccessKey("vpc_id", "s3_service_id", "alice")

	assert.NotNil(t, created)
	assert.True(t, created.Status)
	assert.Equal(t, "AKIAEXAMPLE1", created.AccessKey.AccessKeyID)
	assert.Equal(t, "s3cr3t-never-readable-again", created.AccessKey.SecretAccessKey)
}

func TestCreateIamUserAccessKeyReportsTheCapWhenTwoKeysAlreadyExist(t *testing.T) {
	mockResponse := `{
		"status": false,
		"message": "An IAM user can have at most 2 access keys"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/access-keys/create": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	created := service.CreateIamUserAccessKey("vpc_id", "s3_service_id", "alice")

	assert.NotNil(t, created)
	assert.False(t, created.Status)
	assert.Equal(t, "An IAM user can have at most 2 access keys", created.Message)
	assert.Equal(t, "", created.AccessKey.SecretAccessKey)
}

func TestDeleteIamUserAccessKeyRevokesTheKey(t *testing.T) {
	mockResponse := `{
		"status": true,
		"user_name": "alice",
		"access_key_id": "AKIAEXAMPLE1"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/access-keys/AKIAEXAMPLE1/delete": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.DeleteIamUserAccessKey("vpc_id", "s3_service_id", "alice", "AKIAEXAMPLE1")

	assert.True(t, resp.Status)
}

func TestGetIamUserPolicyReturnsTheAttachedDocument(t *testing.T) {
	mockResponse := `{
		"user_name": "alice",
		"policy_name": "portal-managed-policy",
		"policy": {"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": ["s3:GetObject"], "Resource": ["arn:aws:s3:::my-bucket/*"]}]},
		"has_policy": true,
		"template": {"Version": "2012-10-17", "Statement": []}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/policy": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	policy := service.GetIamUserPolicy("vpc_id", "s3_service_id", "alice")

	assert.NotNil(t, policy)
	assert.True(t, policy.HasPolicy)
	assert.Equal(t, "portal-managed-policy", policy.PolicyName)
	assert.JSONEq(t,
		`{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": ["s3:GetObject"], "Resource": ["arn:aws:s3:::my-bucket/*"]}]}`,
		string(policy.Policy))
}

func TestGetIamUserPolicyReportsNoPolicyWhenTheUserHasNone(t *testing.T) {
	// A user without a policy is a normal state, not an error: the backend
	// answers with a null document and the starter template. Reading that as a
	// failure would make every fresh user look broken.
	mockResponse := `{
		"user_name": "alice",
		"policy_name": "portal-managed-policy",
		"policy": null,
		"has_policy": false,
		"template": {"Version": "2012-10-17", "Statement": []}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/policy": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	policy := service.GetIamUserPolicy("vpc_id", "s3_service_id", "alice")

	assert.NotNil(t, policy)
	assert.False(t, policy.HasPolicy)
}

func TestPutIamUserPolicyStoresTheDocument(t *testing.T) {
	mockResponse := `{
		"status": true,
		"user_name": "alice",
		"policy_name": "portal-managed-policy",
		"policy": {"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": ["s3:GetObject"], "Resource": ["arn:aws:s3:::my-bucket/*"]}]}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/policy": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.PutIamUserPolicy("vpc_id", "s3_service_id", "alice",
		`{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": ["s3:GetObject"], "Resource": ["arn:aws:s3:::my-bucket/*"]}]}`)

	assert.True(t, resp.Status)
}

func TestPutIamUserPolicyRejectsADocumentThatIsNotJson(t *testing.T) {
	// The payload has to go out as a JSON object, so a document that will not
	// parse must fail here rather than reaching the API as a quoted string the
	// validator then rejects with a confusing message.
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/policy": `{"status": true}`,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.PutIamUserPolicy("vpc_id", "s3_service_id", "alice", "not json at all")

	assert.False(t, resp.Status)
	assert.NotEmpty(t, resp.Message)
}

func TestDeleteIamUserPolicyRemovesTheDocument(t *testing.T) {
	mockResponse := `{
		"status": true,
		"user_name": "alice",
		"policy_name": "portal-managed-policy",
		"deleted_policy": {"Version": "2012-10-17", "Statement": []}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-users/alice/policy": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.DeleteIamUserPolicy("vpc_id", "s3_service_id", "alice")

	assert.True(t, resp.Status)
}

func TestCreateIamRoleReturnsTheCreatedRoleWithItsTrustedUsers(t *testing.T) {
	mockResponse := `{
		"status": true,
		"iam_role": {
			"role_name": "read-only",
			"role_id": "RRRRSSSSTTTT",
			"arn": "arn:aws:iam:::role/read-only",
			"path": "/",
			"created_at": "2026-09-17T03:21:00+00:00",
			"trusted_users": ["alice", "bob"]
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/create": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.CreateIamRole("vpc_id", "s3_service_id", "read-only", []string{"alice", "bob"})

	assert.NotNil(t, resp)
	assert.True(t, resp.Status)
	assert.Equal(t, "read-only", resp.IamRole.RoleName)
	assert.Equal(t, "arn:aws:iam:::role/read-only", resp.IamRole.Arn)
	assert.Equal(t, []string{"alice", "bob"}, resp.IamRole.TrustedUsers)
}

func TestGetIamRoleReturnsTheDetailWhenFound(t *testing.T) {
	mockResponse := `{
		"role_name": "read-only",
		"role_id": "RRRRSSSSTTTT",
		"arn": "arn:aws:iam:::role/read-only",
		"path": "/",
		"created_at": "2026-09-17T03:21:00+00:00",
		"trusted_users": ["alice"]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/read-only/detail": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	role := service.GetIamRole("vpc_id", "s3_service_id", "read-only")

	assert.NotNil(t, role)
	assert.Equal(t, "read-only", role.RoleName)
	assert.Equal(t, []string{"alice"}, role.TrustedUsers)
}

func TestGetIamRoleReturnsNilWhenTheApiReportsAFailure(t *testing.T) {
	mockResponse := `{"status": false, "message": "IAM role not found"}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/ghost/detail": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	assert.Nil(t, service.GetIamRole("vpc_id", "s3_service_id", "ghost"))
}

func TestListIamRolesReturnsThePageWhenSuccess(t *testing.T) {
	mockResponse := `{
		"iam_roles": [
			{"role_name": "read-only", "role_id": "RRRR", "trusted_users": ["alice"]},
			{"role_name": "admin", "role_id": "SSSS", "trusted_users": []}
		],
		"total": 2
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/list": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	list, err := service.ListIamRoles("vpc_id", "s3_service_id", 1, 25)

	assert.NoError(t, err)
	assert.Equal(t, 2, list.Total)
	assert.Equal(t, "read-only", list.IamRoles[0].RoleName)
	assert.Equal(t, "admin", list.IamRoles[1].RoleName)
}

func TestUpdateIamRoleTrustedUsersReportsWhatActuallyLanded(t *testing.T) {
	// The update is a full replace and the backend silently skips a name it
	// cannot resolve, so the response - not the request - is what the caller
	// must record.
	mockResponse := `{
		"status": true,
		"role_name": "read-only",
		"trusted_users": ["alice"]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/read-only/trusted-users": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.UpdateIamRoleTrustedUsers("vpc_id", "s3_service_id", "read-only", []string{"alice", "ghost"})

	assert.NotNil(t, resp)
	assert.True(t, resp.Status)
	assert.Equal(t, []string{"alice"}, resp.TrustedUsers)
}

func TestDeleteIamRoleSucceedsWhenTheCleanupCompletes(t *testing.T) {
	mockResponse := `{
		"status": true,
		"role_name": "read-only",
		"policies_deleted": ["portal-managed-policy"],
		"managed_policies_detached": []
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/read-only/delete": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	assert.NoError(t, service.DeleteIamRole("vpc_id", "s3_service_id", "read-only"))
}

func TestDeleteIamRoleReturnsTheApiMessageWhenTheCleanupFails(t *testing.T) {
	mockResponse := `{"status": false, "message": "Failed to delete inline policies for IAM role read-only"}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/read-only/delete": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	err := service.DeleteIamRole("vpc_id", "s3_service_id", "read-only")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to delete inline policies for IAM role read-only")
}

func TestGetIamRolePolicyReturnsTheAttachedDocument(t *testing.T) {
	mockResponse := `{
		"role_name": "read-only",
		"policy_name": "portal-managed-policy",
		"policy": {"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": ["s3:ListBucket"], "Resource": ["arn:aws:s3:::my-bucket"]}]},
		"has_policy": true,
		"template": {"Version": "2012-10-17", "Statement": []}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/read-only/policy": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	policy := service.GetIamRolePolicy("vpc_id", "s3_service_id", "read-only")

	assert.NotNil(t, policy)
	assert.True(t, policy.HasPolicy)
	assert.Equal(t, "read-only", policy.RoleName)
	assert.JSONEq(t,
		`{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": ["s3:ListBucket"], "Resource": ["arn:aws:s3:::my-bucket"]}]}`,
		string(policy.Policy))
}

func TestPutIamRolePolicyStoresTheDocument(t *testing.T) {
	mockResponse := `{
		"status": true,
		"role_name": "read-only",
		"policy_name": "portal-managed-policy",
		"policy": {"Version": "2012-10-17", "Statement": []}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/read-only/policy": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.PutIamRolePolicy("vpc_id", "s3_service_id", "read-only", `{"Version": "2012-10-17", "Statement": []}`)

	assert.True(t, resp.Status)
}

func TestPutIamRolePolicyReportsTheValidatorMessageWhenTheDocumentIsRefused(t *testing.T) {
	// The backend rejects a policy naming a bucket the account does not own.
	// That verdict has to reach the practitioner verbatim.
	mockResponse := `{
		"status": false,
		"message": "The policy refers to a bucket outside this account"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/read-only/policy": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	resp := service.PutIamRolePolicy("vpc_id", "s3_service_id", "read-only", `{"Version": "2012-10-17", "Statement": []}`)

	assert.False(t, resp.Status)
	assert.Equal(t, "The policy refers to a bucket outside this account", resp.Message)
}

func TestDeleteIamRolePolicyRemovesTheDocument(t *testing.T) {
	mockResponse := `{
		"status": true,
		"role_name": "read-only",
		"policy_name": "portal-managed-policy",
		"deleted_policy": {"Version": "2012-10-17", "Statement": []}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/iam-roles/read-only/policy": mockResponse,
	})
	defer server.Close()

	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	assert.True(t, service.DeleteIamRolePolicy("vpc_id", "s3_service_id", "read-only").Status)
}
