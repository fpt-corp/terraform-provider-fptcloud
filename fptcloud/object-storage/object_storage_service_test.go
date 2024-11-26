package fptcloud_object_storage_test

import (
	"fmt"
	"testing"

	common "terraform-provider-fptcloud/commons"
	fptcloud_object_storage "terraform-provider-fptcloud/fptcloud/object-storage"

	"github.com/stretchr/testify/assert"
)

func TestCreateResourceAccessKey_ReturnsResourceAccessKeyIDWhenSuccess(t *testing.T) {
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

func TestCreateResourceAccessKey_ReturnsErrorWhenFailed(t *testing.T) {
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

func TestDeleteResouurceAccessKey_ReturnOkWhenSuccess(t *testing.T) {
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

func TestListAccessKeys_ReturnAccessKeysWhenSuccess(t *testing.T) {
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

func TestCreateBucket_ReturnsBucketIDWhenSuccess(t *testing.T) {
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
		Name: "bucket_name",
		Acl:  "private",
	}
	r := service.CreateBucket(bucketRequest, vpcId, s3ServiceId)
	assert.NotNil(t, r)
	assert.Equal(t, true, r.Status)
}

func TestCreateBucket_ReturnsErrorWhenFailed(t *testing.T) {
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
		Name: "bucket_name",
		Acl:  "private",
	}
	r := service.CreateBucket(bucketRequest, vpcId, s3ServiceId)
	assert.NotNil(t, r)
	assert.Equal(t, false, r.Status)
}

func TestListBuckets_ReturnsBucketsWhenSuccess(t *testing.T) {
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

func TestListBuckets_ReturnsErrorWhenFailed(t *testing.T) {
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

func TestDeleteBucket_ReturnsOkWhenSuccess(t *testing.T) {
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

func TestCreateSubUser_ReturnsTrueWhenSuccess(t *testing.T) {
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
	fmt.Println("Response: ", r)
	assert.NotNil(t, r)
	assert.Equal(t, true, r.Status)
	assert.Equal(t, "Sub-user created successfully", r.Message)
}

func TestCreateSubUser_ReturnsFalseWhenFailed(t *testing.T) {
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
	fmt.Println("Response: ", r)
	assert.NotNil(t, r)
	assert.Equal(t, false, r.Status)
}

func TestDeleteSubUser_ReturnOkWhenSuccess(t *testing.T) {
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

func TestListSubUsers_ReturnsSubUsersWhenSuccess(t *testing.T) {
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
	fmt.Println("SubUsers: ", subUsers)
	fmt.Println("err: ", err)
	assert.NotNil(t, subUsers)
	assert.Nil(t, err)
	assert.Equal(t, 1, subUsers.Total)
	assert.Equal(t, "sgn-replicate123123", subUsers.SubUsers[0].UserID)
	assert.Equal(t, "arn:aws:iam:::user/xxx:sgn-replicate123123", subUsers.SubUsers[0].Arn)
	assert.Equal(t, true, subUsers.SubUsers[0].Active)
	assert.Equal(t, "SubUserReadWrite", subUsers.SubUsers[0].Role)
}

func TestListSubUsers_ReturnsErrorWhenFailed(t *testing.T) {
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

func TestGetDetailSubUser_ReturnOkWhenSuccess(t *testing.T) {
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

func TestCreateSubUserAccessKey_ReturnsAccessKeyWhenSuccess(t *testing.T) {
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
	fmt.Println("AccessKey: ", accessKey)
	assert.NotNil(t, accessKey)
	assert.Equal(t, "example_access_key", accessKey.Credential.AccessKey)
	assert.Equal(t, "example_secret_key", accessKey.Credential.SecretKey)
	assert.Equal(t, true, accessKey.Status)
}

func TestCreateSubUserAccessKey_ReturnsErrorWhenFailed(t *testing.T) {
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
	fmt.Println("AccessKey: ", accessKey)
	assert.NotNil(t, accessKey)
	assert.Equal(t, "", accessKey.Credential.AccessKey)
	assert.Equal(t, "", accessKey.Credential.SecretKey)
	assert.Equal(t, false, accessKey.Status)
}

func TestDeleteSubUserAccessKey_ReturnOkWhenSuccess(t *testing.T) {
	
}