package fptcloud_ssh

import (
	"encoding/json"
	"fmt"
	"strings"
	common "terraform-provider-fptcloud/commons"
)

// SSHKey represents an SSH public key, uploaded to access instances
type SSHKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	PublicKey string `json:"public_key"`
}

// SSHKeyService defines the interface for fetching SSH key details
type SSHKeyService interface {
	ListSSHKeys() ([]SSHKey, error)
	FindSSHKey(search string) (*SSHKey, error)
	NewSSHKey(name string, publicKey string) (*SSHKey, error)
	DeleteSSHKey(ID string) (*common.SimpleResponse, error)
}

// SSHKeyServiceImpl is the implementation of SSHKeyService
type SSHKeyServiceImpl struct {
	client *common.Client
}

// NewSSHKeyService creates a new instance of SSHKeyService with the given client
func NewSSHKeyService(client *common.Client) SSHKeyService {
	return &SSHKeyServiceImpl{client: client}
}

// ListSSHKeys list all SSH key for an account
func (s *SSHKeyServiceImpl) ListSSHKeys() ([]SSHKey, error) {
	var apiPath = common.ApiPath.SSH + "?page=1&page_size=9999"
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, err
	}

	var sshResponse struct {
		Total int      `json:"total"`
		Data  []SSHKey `json:"data"`
	}

	err = json.Unmarshal(resp, &sshResponse)
	if err != nil {
		return nil, err
	}

	return sshResponse.Data, nil
}

// FindSSHKey finds an SSH key by either part of the ID or part of the name
func (s *SSHKeyServiceImpl) FindSSHKey(search string) (*SSHKey, error) {
	keys, err := s.ListSSHKeys()
	if err != nil {
		return nil, common.DecodeError(err)
	}

	exactMatch := false
	partialMatchesCount := 0
	result := SSHKey{}

	for _, value := range keys {
		if value.Name == search || value.ID == search {
			exactMatch = true
			result = value
		} else if strings.Contains(value.Name, search) || strings.Contains(value.ID, search) {
			if !exactMatch {
				result = value
				partialMatchesCount++
			}
		}
	}

	if exactMatch || partialMatchesCount == 1 {
		return &result, nil
	} else if partialMatchesCount > 1 {
		err := fmt.Errorf("unable to find %s because there were multiple matches", search)
		return nil, common.MultipleMatchesError.Wrap(err)
	} else {
		err := fmt.Errorf("unable to find %s, zero matches", search)
		return nil, common.ZeroMatchesError.Wrap(err)
	}
}

// DeleteSSHKey deletes an SSH key
func (s *SSHKeyServiceImpl) DeleteSSHKey(id string) (*common.SimpleResponse, error) {
	_, err := s.client.SendDeleteRequestWithBody(common.ApiPath.SSH, map[string]string{
		"ssh_id": id,
	})
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}

// NewSSHKey creates a new SSH key record
func (s *SSHKeyServiceImpl) NewSSHKey(name string, publicKey string) (*SSHKey, error) {
	resp, err := s.client.SendPostRequest(common.ApiPath.SSH, map[string]string{
		"name":       name,
		"public_key": publicKey,
	})
	if err != nil {
		return nil, common.DecodeError(err)
	}

	result := &SSHKey{}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
