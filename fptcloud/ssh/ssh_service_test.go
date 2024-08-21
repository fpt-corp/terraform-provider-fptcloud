package fptcloud_ssh_test

import (
	"strings"
	common "terraform-provider-fptcloud/commons"
	"testing"

	"github.com/stretchr/testify/assert"
	"terraform-provider-fptcloud/fptcloud/ssh"
)

func TestShouldItReturnListSSHKeys(t *testing.T) {
	mockResponse := `{
		"total": 1,
		"data": [
			{
				"id": "1",
				"name": "test-key",
				"created_at": "2024-05-18T12:08:43",
				"public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCg8gop..."
			}
		]
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/user/sshs?page=1&page_size=9999": mockResponse,
	})
	defer server.Close()

	sshService := fptcloud_ssh.NewSSHKeyService(mockClient)

	keys, err := sshService.ListSSHKeys()
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "1", keys[0].ID)
	assert.Equal(t, "test-key", keys[0].Name)
	assert.Equal(t, "2024-05-18T12:08:43", keys[0].CreatedAt)
	assert.Equal(t, "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCg8gop...", keys[0].PublicKey)
}

func TestShouldItFindSSHKey(t *testing.T) {
	mockResponse := `{
		"total": 2,
		"data": [
			{
				"id": "1",
				"name": "test-key-1",
				"created_at": "2024-05-18T12:08:43",
				"public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCg8gop..."
			},
			{
				"id": "2",
				"name": "test-key-2",
				"created_at": "2024-05-18T12:08:43",
				"public_key": "ssh-rsa BBBAB3NzaC1yc2EAAAADAQABAAABgQCg8gop..."
			}
		]
	}`

	mockClient, server, _ := common.NewClientForTesting(map[string]string{
		"/v1/user/sshs?page=1&page_size=9999": mockResponse,
	})
	defer server.Close()

	sshService := fptcloud_ssh.NewSSHKeyService(mockClient)

	t.Run("Find exact match by name", func(t *testing.T) {
		key, err := sshService.FindSSHKey("test-key-1")
		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.Equal(t, "1", key.ID)
		assert.Equal(t, "test-key-1", key.Name)
	})

	t.Run("Find exact match by ID", func(t *testing.T) {
		key, err := sshService.FindSSHKey("1")
		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.Equal(t, "1", key.ID)
		assert.Equal(t, "test-key-1", key.Name)
	})

	t.Run("Multiple partial matches", func(t *testing.T) {
		_, err := sshService.FindSSHKey("test-key")
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "multiple matches"))
	})

	t.Run("No matches", func(t *testing.T) {
		_, err := sshService.FindSSHKey("nonexistent-key")
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "zero matches"))
	})
}
