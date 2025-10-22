package fptcloud_object_storage_test

import (
	"testing"

	common "terraform-provider-fptcloud/commons"
	fptcloud_object_storage "terraform-provider-fptcloud/fptcloud/object-storage"

	"github.com/stretchr/testify/assert"
)

func TestCreateResourceAccessKeyReturnsResourceAccessKeyIDWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"message": "Create resource access key successfully",
		"credential": {
			"accessKey": "11111111-aaaa-1111-bbbb-111111111111",
			"secretKey": "22222222-bbbb-2222-cccc-222222222222"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/user/credentials": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	resourceAccessKeyID := service.CreateAccessKey(vpcId, s3ServiceId)
	assert.NotNil(t, resourceAccessKeyID)

	assert.Equal(t, "11111111-aaaa-1111-bbbb-111111111111", resourceAccessKeyID.Credential.AccessKey)
	assert.Equal(t, "22222222-bbbb-2222-cccc-222222222222", resourceAccessKeyID.Credential.SecretKey)
	assert.Equal(t, true, resourceAccessKeyID.Status)
	assert.Equal(t, "Create resource access key successfully", resourceAccessKeyID.Message)
}

func TestCreateResourceAccessKeyReturnsErrorWhenFailed(t *testing.T) {
	mockResponse := `{
		"status": false,
		"message": "Failed to create resource access key",
		"credential": {}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/user/credentials": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	resourceAccessKeyID := service.CreateAccessKey(vpcId, s3ServiceId)
	assert.NotNil(t, resourceAccessKeyID)

	assert.Equal(t, "", resourceAccessKeyID.Credential.AccessKey)
	assert.Equal(t, "", resourceAccessKeyID.Credential.SecretKey)
	assert.Equal(t, false, resourceAccessKeyID.Status)
	assert.Equal(t, "Failed to create resource access key", resourceAccessKeyID.Message)
}

func TestDeleteResouurceAccessKeyReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"message": "Delete resource access key successfully"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/user/credentials/credential_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	credentialId := "credential_id"
	res := service.DeleteAccessKey(vpcId, s3ServiceId, credentialId)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
	assert.Equal(t, "Access key deleted successfully", res.Message)
}

func TestListAccessKeysReturnAccessKeysWhenSuccess(t *testing.T) {
	mockResponse := `{
		"credentials": [
			{
				"id": "credential_id",
				"credentials": [
					{
						"accessKey": "11111111-aaaa-1111-bbbb-111111111111",
						"active": true
					}
				]
			}
		]
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/user/credentials?s3_service_id=s3_service_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	accessKeys, _ := service.ListAccessKeys(vpcId, s3ServiceId)
	assert.NotNil(t, accessKeys)
	assert.Equal(t, "credential_id", accessKeys.Credentials[0].ID)
	assert.Equal(t, "11111111-aaaa-1111-bbbb-111111111111", accessKeys.Credentials[0].Credentials[0].AccessKey)
	assert.Equal(t, true, accessKeys.Credentials[0].Credentials[0].Active)
}

func TestCreateBucketReturnsBucketIDWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"message": "Create bucket successfully"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	bucketRequest := fptcloud_object_storage.BucketRequest{
		Name:       "bucket_name",
		Acl:        "private",
		ObjectLock: false,
	}
	r := service.CreateBucket(bucketRequest, vpcId, s3ServiceId)
	assert.NotNil(t, r)
	assert.Equal(t, true, r.Status)
}

func TestCreateBucketReturnsErrorWhenFailed(t *testing.T) {
	mockResponse := `{
		"status": false,
		"message": "Failed to create bucket",
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	bucketRequest := fptcloud_object_storage.BucketRequest{
		Name:       "bucket_name",
		Acl:        "private",
		ObjectLock: false,
	}
	r := service.CreateBucket(bucketRequest, vpcId, s3ServiceId)
	assert.NotNil(t, r)
	assert.Equal(t, false, r.Status)
}

func TestListBucketsReturnsBucketsWhenSuccess(t *testing.T) {
	mockResponse := `{
		"buckets": [
			{
				"Name": "bucket_name",
				"CreationDate": "2024-11-26T16:43:55.121000+00:00",
				"isEmpty": false,
				"s3_service_id": "s3_service_id",
				"isEnabledLogging": false,
				"endpoint": "https://xxxx-xxx.xyz.com"
			}
		],
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/buckets?page=5&page_size=10&s3_service_id=s3_service_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	buckets := service.ListBuckets(vpcId, s3ServiceId, 5, 10)
	assert.NotNil(t, buckets)
	assert.Equal(t, "bucket_name", buckets.Buckets[0].Name)
	assert.Equal(t, "2024-11-26T16:43:55.121000+00:00", buckets.Buckets[0].CreationDate)
	assert.Equal(t, false, buckets.Buckets[0].IsEmpty)
	assert.Equal(t, "s3_service_id", buckets.Buckets[0].S3ServiceID)
	assert.Equal(t, false, buckets.Buckets[0].IsEnabledLogging)
	assert.Equal(t, "https://xxxx-xxx.xyz.com", buckets.Buckets[0].Endpoint)
}

func TestListBucketsReturnsErrorWhenFailed(t *testing.T) {
	mockResponse := `{
		"buckets": [],
		"total": 0
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/buckets?page=5&page_size=10&s3_service_id=s3_service_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	buckets := service.ListBuckets(vpcId, s3ServiceId, 5, 10)
	assert.NotNil(t, buckets)
	assert.Equal(t, 0, buckets.Total)
}

func TestDeleteBucketReturnsOkWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	bucketName := "bucket_name"
	res := service.DeleteBucket(vpcId, s3ServiceId, bucketName)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestCreateSubUserReturnsTrueWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"message": "Sub-user created successfully"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/create": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUserRequest := fptcloud_object_storage.SubUser{
		Role:   "admin",
		UserId: "user_id",
	}
	r := service.CreateSubUser(subUserRequest, vpcId, s3ServiceId)
	assert.NotNil(t, r)
	assert.Equal(t, true, r.Status)
	assert.Equal(t, "Sub-user created successfully", r.Message)
}

func TestCreateSubUserReturnsFalseWhenFailed(t *testing.T) {
	mockResponse := `{
		"status": false
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/wrong_endpoint": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUserRequest := fptcloud_object_storage.SubUser{
		Role:   "admin",
		UserId: "user_id",
	}
	r := service.CreateSubUser(subUserRequest, vpcId, s3ServiceId)
	assert.NotNil(t, r)
	assert.Equal(t, false, r.Status)
}

func TestDeleteSubUserReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `{}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/sub_user_id/delete": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUserId := "sub_user_id"
	err := service.DeleteSubUser(vpcId, s3ServiceId, subUserId)
	assert.Nil(t, err)
}

func TestListSubUsersReturnsSubUsersWhenSuccess(t *testing.T) {
	mockResponse := `{
		"sub_users": [
			{
				"user_id": "sgn-replicate123123",
				"arn": "arn:aws:iam:::user/xxx:sgn-replicate123123",
				"active": true,
				"role": "SubUserReadWrite"
			}
		],
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/list?page=5&page_size=25": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUsers, err := service.ListSubUsers(vpcId, s3ServiceId, 5, 25)
	assert.NotNil(t, subUsers)
	assert.Nil(t, err)
	assert.Equal(t, 1, subUsers.Total)
	assert.Equal(t, "sgn-replicate123123", subUsers.SubUsers[0].UserID)
	assert.Equal(t, "arn:aws:iam:::user/xxx:sgn-replicate123123", subUsers.SubUsers[0].Arn)
	assert.Equal(t, true, subUsers.SubUsers[0].Active)
	assert.Equal(t, "SubUserReadWrite", subUsers.SubUsers[0].Role)
}

func TestListSubUsersReturnsErrorWhenFailed(t *testing.T) {
	mockResponse := `{
		"sub_users": [],
		"total": 0,
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/list?page=5&page_size=25": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUsers, err := service.ListSubUsers(vpcId, s3ServiceId, 5, 25)
	assert.NotNil(t, subUsers)
	assert.NotNil(t, err)
	assert.Equal(t, 0, subUsers.Total)
}

func TestGetDetailSubUserReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `
		{
			"user_id": "sgn-replicate123123",
			"active": true,
			"role": "SubUserReadWrite",
			"access_keys": []
		}
	`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/sub_user_id": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUserId := "sub_user_id"
	subUser := service.DetailSubUser(vpcId, s3ServiceId, subUserId)
	assert.NotNil(t, subUser)
	assert.Equal(t, "sgn-replicate123123", subUser.UserID)
	assert.Equal(t, true, subUser.Active)
	assert.Equal(t, "SubUserReadWrite", subUser.Role)
}

func TestGetDetailSubUserReturnNilWhenError(t *testing.T) {
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/nonexistent_user": "",
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUserId := "nonexistent_user"
	subUser := service.DetailSubUser(vpcId, s3ServiceId, subUserId)
	assert.Nil(t, subUser, "DetailSubUser should return nil when user is not found")
}

func TestCreateSubUserAccessKeyReturnsAccessKeyWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"credential": {
			"accessKey": "example_access_key",
			"secretKey": "example_secret_key"
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/sub_user_id/credentials/create": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUserId := "sub_user_id"
	accessKey := service.CreateSubUserAccessKey(vpcId, s3ServiceId, subUserId)
	assert.NotNil(t, accessKey)
	assert.Equal(t, "example_access_key", accessKey.Credential.AccessKey)
	assert.Equal(t, "example_secret_key", accessKey.Credential.SecretKey)
	assert.Equal(t, true, accessKey.Status)
}

func TestCreateSubUserAccessKeyReturnsErrorWhenFailed(t *testing.T) {
	mockResponse := `{
		"status": false,
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/sub_user_id/credentials/create": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUserId := "sub_user_id"
	accessKey := service.CreateSubUserAccessKey(vpcId, s3ServiceId, subUserId)
	assert.NotNil(t, accessKey)
	assert.Equal(t, "", accessKey.Credential.AccessKey)
	assert.Equal(t, "", accessKey.Credential.SecretKey)
	assert.Equal(t, false, accessKey.Status)
}

func TestDeleteSubUserAccessKeyReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/sub-users/sub_user_id/credentials/delete": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	subUserId := "sub_user_id"
	accessKeyId := "access_key_id"
	res := service.DeleteSubUserAccessKey(vpcId, s3ServiceId, subUserId, accessKeyId)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestPutBucketPolicyReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/put-policy": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	bucketName := "bucket_name"
	policy := map[string]interface {
	}{"Version": "2012-10-17"}
	res := service.PutBucketPolicy(vpcId, s3ServiceId, bucketName, policy)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestGetBucketPolicyReturnsPolicyWhenSuccess(t *testing.T) {
	mockResponse := `{
		"policy": "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Sid\":\"AllowAllS3Actions\",\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"s3:*\",\"Resource\":\"arn:aws:s3:::bucket_name/*\"}]}",
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/get-policy": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	bucketName := "bucket_name"
	policy := service.GetBucketPolicy(vpcId, s3ServiceId, bucketName)
	assert.NotNil(t, policy)
	assert.Equal(t, "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Sid\":\"AllowAllS3Actions\",\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"s3:*\",\"Resource\":\"arn:aws:s3:::bucket_name/*\"}]}", policy.Policy)
	assert.Equal(t, true, policy.Status)
}

func TestGetBucketPolicyReturnsFalseWhenFailed(t *testing.T) {
	mockResponse := `{
		"policy": "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Sid\":\"AllowAllS3Actions\",\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"s3:*\",\"Resource\":\"arn:aws:s3:::bucket_name/*\"}]}",
		"status": false,
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/get-policy": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	s3ServiceId := "s3_service_id"
	bucketName := "bucket_name"
	policy := service.GetBucketPolicy(vpcId, s3ServiceId, bucketName)
	assert.NotNil(t, policy)
	assert.Equal(t, false, policy.Status)
}

func TestCreateBucketCorsReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/create-bucket-cors": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	cors := map[string]interface{}{
		"AllowedHeaders": []string{"*"},
	}
	res := service.CreateBucketCors("vpc_id", "s3_service_id", bucketName, cors)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestUpdateBucketCorsReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/create-bucket-cors": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	cors := map[string]interface{}{
		"AllowedHeaders": []string{"*"},
	}
	arrCors := append([]map[string]interface{}{}, cors)
	res := service.UpdateBucketCors("vpc_id", "s3_service_id", bucketName, arrCors)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestGetBucketCorsReturnCorsWhenSuccess(t *testing.T) {
	mockResponse := `{
		"cors_rules": [
			{
				"AllowedHeaders": [
					"*"
				]
			}
		],
		"status": true,
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/cors?page=5&page_size=25": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	cors, err := service.GetBucketCors("vpc_id", "s3_service_id", bucketName, 5, 25)
	assert.NotNil(t, cors)
	assert.Nil(t, err)
	assert.Equal(t, true, cors.Status)
	assert.Equal(t, "*", cors.CorsRules[0].AllowedHeaders[0])
}

func TestGetBucketCorsReturnFalseWhenFailed(t *testing.T) {
	mockResponse := `{
		"cors_rules": [],
		"status": false,
		"total": 0,
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/cors?page=5&page_size=25": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	cors, err := service.GetBucketCors("vpc_id", "s3_service_id", bucketName, 5, 25)
	assert.Nil(t, cors)
	assert.NotNil(t, err)
}

func TestPutBucketVersioningReturnNilWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/put-versioning": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	versioning := fptcloud_object_storage.BucketVersioningRequest{
		Status: "Enabled",
	}
	res := service.PutBucketVersioning("vpc_id", "s3_service_id", bucketName, versioning)
	assert.Nil(t, res)
}

func TestGetBucketVersioningReturnBucketVersioning(t *testing.T) {
	mockResponse := `{
		"status": true,
		"config": "Enabled"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/get-versioning": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	versioning := service.GetBucketVersioning("vpc_id", "s3_service_id", bucketName)
	assert.NotNil(t, versioning)
	assert.Equal(t, true, versioning.Status)
}

func TestPutBucketAclReturnAclWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"taskId": "task_id"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/acl": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	acl := fptcloud_object_storage.BucketAclRequest{
		CannedAcl:    "private",
		ApplyObjects: true,
	}
	res := service.PutBucketAcl("vpc_id", "s3_service_id", bucketName, acl)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
	assert.Equal(t, "task_id", res.TaskID)
}

func TestGetBucketAclReturnAclWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"Owner": {
			"DisplayName": "example_user_id",
			"ID": "example_user_id"
		},
		"Grants": [
			{
				"Grantee": {
					"DisplayName": "example_user_id",
					"ID": "example_user_id",
					"Type": "CanonicalUser"
				},
				"Permission": "FULL_CONTROL"
			}
		],
		"CannedACL": "private"
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/acl": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	acl := service.GetBucketAcl("vpc_id", "s3_service_id", bucketName)
	assert.NotNil(t, acl)
	assert.Equal(t, true, acl.Status)
	assert.Equal(t, "example_user_id", acl.Owner.DisplayName)
	assert.Equal(t, "example_user_id", acl.Owner.ID)
	assert.Equal(t, "example_user_id", acl.Grants[0].Grantee.DisplayName)
	assert.Equal(t, "example_user_id", acl.Grants[0].Grantee.ID)
	assert.Equal(t, "CanonicalUser", acl.Grants[0].Grantee.Type)
	assert.Equal(t, "FULL_CONTROL", acl.Grants[0].Permission)
}

func TestGetBucketAclReturnFalseWhenFailed(t *testing.T) {
	mockResponse := `{
		"status": false
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/acl-wrong-endpoint": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	acl := service.GetBucketAcl("vpc_id", "s3_service_id", bucketName)
	assert.NotNil(t, acl)
	assert.Equal(t, false, acl.Status)
}

func TestGetBucketAclReturnFalseWhenFailedUnmarshalJson(t *testing.T) {
	mockResponse := `{
		"status": false,,,
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/acl-wrong-endpoint": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	acl := service.GetBucketAcl("vpc_id", "s3_service_id", bucketName)
	assert.NotNil(t, acl)
	assert.Equal(t, false, acl.Status)
}

func TestPutBucketWebsiteReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/put-config": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	website := fptcloud_object_storage.BucketWebsiteRequest{
		Key:    "index.html",
		Suffix: "index2.html",
		Bucket: "bucket_name",
	}
	res := service.PutBucketWebsite("vpc_id", "s3_service_id", bucketName, website)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestPutBucketWebsiteReturnOFalseWhenFailed(t *testing.T) {
	mockResponse := `{
		"status": false,
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/put-config": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	website := fptcloud_object_storage.BucketWebsiteRequest{
		Key:    "example.html",
		Suffix: "index2.html",
		Bucket: "bucket_name",
	}
	res := service.PutBucketWebsite("vpc_id", "s3_service_id", bucketName, website)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestDeleteBucketStaticWebsiteReturnTrueWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/delete-config": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	res := service.DeleteBucketStaticWebsite("vpc_id", "s3_service_id", bucketName)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestDeleteBucketStaticWebsiteReturnFalseWhenError(t *testing.T) {
	mockResponse := `{
		"status": false
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/delete-config-wrong-endpoint": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	res := service.DeleteBucketStaticWebsite("vpc_id", "s3_service_id", bucketName)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestGetBucketWebsiteReturnWebsiteWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"config": {
			"ResponseMetadata": {
				"RequestId": "tx000000976595dcbf0f8e1-006746c273-326c5-han02-1",
				"HostId": "",
				"HTTPStatusCode": 200,
				"HTTPHeaders": {
					"x-amz-request-id": "tx000000976595dcbf0f8e1-006746c273-326c5-han02-1",
					"content-type": "application/xml",
					"content-length": "241",
					"date": "Wed, 27 Nov 2024 06:55:47 GMT",
					"strict-transport-security": "max-age=16000000; includeSubDomains; preload;",
					"access-control-allow-origin": "*"
				},
				"RetryAttempts": 0
			},
			"IndexDocument": {
				"Suffix": "some_index.html"
			},
			"ErrorDocument": {
				"Key": "error.html"
			}
		}
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/get-config": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	website := service.GetBucketWebsite("vpc_id", "s3_service_id", bucketName)
	assert.NotNil(t, website)
	assert.Equal(t, true, website.Status)
	assert.Equal(t, "some_index.html", website.Config.IndexDocument.Suffix)
	assert.Equal(t, "error.html", website.Config.ErrorDocument.Key)
}

func TestGetBucketWebsiteReturnFalseWhenError(t *testing.T) {
	mockResponse := `{
		"status": false
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/get-config-wrong-endpoint": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	website := service.GetBucketWebsite("vpc_id", "s3_service_id", bucketName)
	assert.NotNil(t, website)
	assert.Equal(t, false, website.Status)
}

func TestGetBucketLifecycleReturnRuleWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true,
		"rules": [
			{
				"ID": "rule_id",
				"Prefix": "prefix",
				"Status": "Enabled",
				"Expiration": {
					"Days": 30
				}
			}
		],
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/lifecycles?page=5&page_size=25": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	lifecycle := service.GetBucketLifecycle("vpc_id", "s3_service_id", bucketName, 5, 25)
	assert.NotNil(t, lifecycle)
	assert.Equal(t, true, lifecycle.Status)
	assert.Equal(t, "rule_id", lifecycle.Rules[0].ID)
	assert.Equal(t, "prefix", lifecycle.Rules[0].Prefix)
	assert.Equal(t, "Enabled", lifecycle.Rules[0].Status)
	assert.Equal(t, 30, lifecycle.Rules[0].Expiration.Days)
	assert.Equal(t, 1, lifecycle.Total)
}

func TestGetBucketLifecycleReturnFalseWhenFailed(t *testing.T) {
	mockResponse := `{
		"status": false,
		"rules": [],
		"total": 0
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/lifecycles-wrong-endpoint?page=5&page_size=25": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	lifecycle := service.GetBucketLifecycle("vpc_id", "s3_service_id", bucketName, 5, 25)
	assert.NotNil(t, lifecycle)
	assert.Equal(t, false, lifecycle.Status)
	assert.Equal(t, 0, lifecycle.Total)
}

func TestPutBucketLifecycleReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name22/create-bucket-lifecycle-configuration": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name22"
	rule := map[string]interface{}{
		"ID":     "rule_id",
		"Prefix": "prefix2222222",
		"Status": "Disabled",
		"Expiration": map[string]interface{}{
			"Days": 8,
		},
	}
	res := service.PutBucketLifecycle("vpc_id", "s3_service_id", bucketName, rule)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestPutBucketLifecycleReturnFalseWhenError(t *testing.T) {
	mockResponse := `{
		"status": false
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name1111/create-bucket-lifecycle-configuration-wrong-endpoint": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name1111"
	rule := map[string]interface{}{
		"ID": "rule_id",
		"Expiration": map[string]interface{}{
			"Days": 90,
		},
		"Prefix": "filer",
		"Status": "Enabled",
	}
	res := service.PutBucketLifecycle("vpc_id", "s3_service_id", bucketName, rule)
	assert.NotNil(t, res)
	assert.Equal(t, false, res.Status)
}

func TestPutBucketLifecycleReturnFalseWhenErrorUnmarshalJson(t *testing.T) {
	mockResponse := `{
		"status": false,,,,@#$@#$234
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/create-bucket-lifecycle-configuration-wrong-endpoint": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	rule := map[string]interface{}{
		"Status": "Enabled",
		"ID":     "rule_id",
		"Expiration": map[string]interface{}{
			"Days": 30,
		},
		"Prefix": "prefix",
	}
	res := service.PutBucketLifecycle("vpc_id", "s3_service_id", bucketName, rule)
	assert.NotNil(t, res)
	assert.Equal(t, false, res.Status)
}

func TestDeleteBucketLifecycleReturnOkWhenSuccess(t *testing.T) {
	mockResponse := `{
		"status": true
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/delete-bucket-lifecycle-configuration": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	rule := map[string]interface{}{
		"ID": "rule_id",
		"Expiration": map[string]interface{}{
			"Days": 12,
		},
		"Prefix": "prefix",
		"Status": "Enabled",
	}
	res := service.DeleteBucketLifecycle("vpc_id", "s3_service_id", bucketName, rule)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Status)
}

func TestDeleteBucketLifecycleReturnFalseWhenError(t *testing.T) {
	mockResponse := `{
		"status": false
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/delete-bucket-lifecycle-configuration-wrong-endpoint": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	rule := map[string]interface{}{
		"ID": "rule_id",
		"Prefix": map[string]interface{}{
			"Filter": "filter",
		},
		"Status": "Disabled",
		"Expiration": map[string]interface{}{
			"Days": 12,
		},
	}
	res := service.DeleteBucketLifecycle("vpc_id", "s3_service_id", bucketName, rule)
	assert.NotNil(t, res)
	assert.Equal(t, false, res.Status)
}

func TestDeleteBucketLifecycleReturnFalseWhenErrorUnmarshalJson(t *testing.T) {
	mockResponse := `{
		"status": false,,,,@#$@#$234
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/s3_service_id/bucket/bucket_name/delete-bucket-lifecycle-configuration-wrong-endpoint": mockResponse,
	})
	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	bucketName := "bucket_name"
	rule := map[string]interface{}{
		"Prefix": "prefix",
		"ID":     "rule_id9999",
		"Expiration": map[string]interface{}{
			"Days": 30,
		},
		"Status": "Disabled",
	}
	res := service.DeleteBucketLifecycle("vpc_id", "s3_service_id", bucketName, rule)
	assert.NotNil(t, res)
	assert.Equal(t, false, res.Status)
}

func TestCheckServiceEnableReturnServicesWhenSuccess(t *testing.T) {
	mockResponse := `{
		"data": [
			{
				"s3_service_name": "HN-02",
				"s3_service_id": "s3_service_id",
				"s3_platform": "ceph",
				"default_user": "fake-default-user",
				"migrate_quota": 3,
				"sync_quota": 3,
				"rgw_total_nodes": 4,
				"rgw_user_active_nodes": 2
			}
		],
		"total": 1
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/check-service-enabled?check_unlimited=undefined": mockResponse,
	})

	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	services := service.CheckServiceEnable(vpcId)
	assert.NotNil(t, services)
	assert.Equal(t, 1, services.Total)
	assert.Equal(t, "HN-02", services.Data[0].S3ServiceName)
	assert.Equal(t, "s3_service_id", services.Data[0].S3ServiceID)
	assert.Equal(t, "ceph", services.Data[0].S3Platform)
}

func TestCheckServiceEnableReturnFalseWhenError(t *testing.T) {
	mockResponse := `{
		"total": 0
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/check-service-enabled?check_unlimited=wrong-param": mockResponse,
	})

	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	services := service.CheckServiceEnable(vpcId)
	assert.NotNil(t, services)
	assert.Equal(t, 0, services.Total)
}

func TestCheckServiceEnableReturnFalseWhenErrorUnmarshal(t *testing.T) {
	mockResponse := `{
		"total": #$%#$%#$%#$%#$%!@#!23,
	}`
	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/vmware/vpc/vpc_id/s3/check-service-enabled?check_unlimited=wrong-param": mockResponse,
	})

	defer server.Close()
	service := fptcloud_object_storage.NewObjectStorageService(mockClient)
	vpcId := "vpc_id"
	services := service.CheckServiceEnable(vpcId)
	assert.NotNil(t, services)
	assert.Equal(t, 0, services.Total)
}
