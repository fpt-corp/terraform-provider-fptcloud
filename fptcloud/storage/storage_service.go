package fptcloud_storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	common "terraform-provider-fptcloud/commons"
	"terraform-provider-fptcloud/commons/utils"
)

const (
	External = "EXTERNAL"
	Local    = "LOCAL"
)

// FindStorageDTO find storage model defined
type FindStorageDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	VpcId string `json:"vpc_id"`
}

// StorageDTO storage dto model to create storage
type StorageDTO struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	SizeGb          int      `json:"size_gb"`
	StoragePolicyId string   `json:"storage_policy_id"`
	InstanceId      *string  `json:"instance_id"`
	VpcId           string   `json:"vpc_id"`
	TagIds          []string `json:"tag_ids,omitempty"`
}

// UpdateStorageDTO storage dto model to update storage
type UpdateStorageDTO struct {
	Name            string `json:"name"`
	SizeGb          int    `json:"size_gb"`
	StoragePolicyId string `json:"storage_policy_id"`
}

// ResizeStorageDTO storage dto model to resize storage
type ResizeStorageDTO struct {
	DiskId          string `json:"disk_id"`
	Size            int    `json:"size"`
	Name            string `json:"name"`
	StoragePolicyId string `json:"storagePolicyId"`
}

// Storage represents a storage model
type Storage struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	SizeGb          int      `json:"size_gb"`
	StoragePolicy   string   `json:"storage_policy"`
	StoragePolicyId string   `json:"storage_policy_id"`
	InstanceId      string   `json:"instance_id"`
	Status          string   `json:"status"`
	VpcId           string   `json:"vpc_id"`
	CreatedAt       string   `json:"created_at"`
	TagIds          []string `json:"tag_ids,omitempty"`
}

// StorageListModel is one storage row as returned by the paginated /storages
// list endpoint. Unlike the singular Storage model it reports the size in MB
// (`size`), the attached instance as vm_id, the policy name as
// storage_policy_name, and tags as objects instead of tag_ids.
type StorageListModel struct {
	ID                string           `json:"id"`
	VpcId             string           `json:"vpc_id"`
	Name              string           `json:"name"`
	DisplayName       *string          `json:"display_name"`
	Description       *string          `json:"description"`
	SizeMb            int              `json:"size"`
	Status            string           `json:"status"`
	InstanceId        *string          `json:"vm_id"`
	InstanceName      *string          `json:"vm_name"`
	StorageType       string           `json:"storage_type"`
	StoragePolicyId   *string          `json:"storage_policy_id"`
	StoragePolicyName *string          `json:"storage_policy_name"`
	DiskId            *string          `json:"disk_id"`
	Encrypted         *StringBool      `json:"encrypted"`
	ZoneId            *string          `json:"zone_id"`
	CreatedAt         string           `json:"created_at"`
	Tags              []StorageListTag `json:"tags"`
}

// StorageListTag is a tag attached to a storage as returned by the list endpoint.
// Its `id` is the tag id, the same value the singular endpoint returns in tag_ids.
type StorageListTag struct {
	ID string `json:"id"`
}

// StringBool unmarshals a value that the API sometimes emits as a string
// ("0"/"1"/"true"/"false") and sometimes as a JSON boolean. It normalises both
// into a bool. The /storages list endpoint, for example, returns the encrypted
// flag as the string "0"/"1".
type StringBool bool

// UnmarshalJSON implements json.Unmarshaler for StringBool.
func (b *StringBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	switch strings.ToLower(s) {
	case "1", "true":
		*b = true
	case "0", "false", "":
		*b = false
	default:
		return fmt.Errorf("cannot unmarshal %q as a boolean", data)
	}
	return nil
}

// StorageListDTO holds the parameters of a paginated storage list request.
type StorageListDTO struct {
	VpcId    string
	PageSize int
}

// ListStoragesResponse is the paginated list response of the /storages endpoint.
type ListStoragesResponse struct {
	Data  []StorageListModel `json:"data"`
	Total int                `json:"total"`
}

// StorageService defines the interface for storage service
type StorageService interface {
	FindStorage(searchModel FindStorageDTO) (*Storage, error)
	Get(vpcId string, storageId string) (*Storage, error)
	ListAll(listModel StorageListDTO) ([]StorageListModel, error)
	CreateStorage(createdModel StorageDTO) (string, error)
	CreateStorageAsync(createdModel StorageDTO) (string, error)
	LookupStorageByName(vpcId string, name string) (StorageNameLookup, error)
	UpdateStorage(vpcId string, storageId string, updatedModel UpdateStorageDTO) (*common.SimpleResponse, error)
	ResizeStorage(vpcId string, resizeModel ResizeStorageDTO) (*common.SimpleResponse, error)
	UpdateTags(vpcId string, storageId string, tagIds []string) (*common.SimpleResponse, error)
	UpdateAttachedInstance(vpcId string, storageId string, instanceId *string) (*common.SimpleResponse, error)
	DeleteStorage(vpcId string, storageId string) (*common.SimpleResponse, error)
}

// StorageServiceImpl is the implementation of StorageServiceImpl
type StorageServiceImpl struct {
	client *common.Client
}

// NewStorageService creates a new instance of storage Service with the given client
func NewStorageService(client *common.Client) StorageService {
	return &StorageServiceImpl{client: client}
}

// ListAll returns every storage in a VPC by walking the paginated list endpoint.
// It issues ceil(total/pageSize) requests instead of one per storage.
func (s *StorageServiceImpl) ListAll(listModel StorageListDTO) ([]StorageListModel, error) {
	return common.ListAllPages(listModel.PageSize, func(page int) ([]StorageListModel, int, error) {
		resp, err := s.client.SendGetRequest(common.ApiPath.ListStorages(listModel.VpcId, page, listModel.PageSize))
		if err != nil {
			return nil, 0, common.DecodeError(err)
		}

		var response ListStoragesResponse
		if err := json.Unmarshal(resp, &response); err != nil {
			return nil, 0, common.DecodeError(err)
		}
		return response.Data, response.Total, nil
	}, func(storage StorageListModel) string {
		return storage.ID
	})
}

// FindStorage finds a storage by either part of the ID or part of the name
func (s *StorageServiceImpl) FindStorage(searchModel FindStorageDTO) (*Storage, error) {
	var apiPath = common.ApiPath.Storage(searchModel.VpcId) + utils.ToQueryParams(searchModel)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	result := Storage{}
	err = json.Unmarshal(resp, &result)

	if err != nil {
		return nil, common.DecodeError(err)
	}
	return &result, nil
}

func (s *StorageServiceImpl) Get(vpcId string, storageId string) (*Storage, error) {
	resp, err := s.client.SendGetRequest(common.ApiPath.Storage(vpcId) + utils.ToQueryParams(FindStorageDTO{ID: storageId}))
	if err != nil {
		return nil, err
	}
	var found Storage
	if err := json.Unmarshal(resp, &found); err != nil {
		return nil, fmt.Errorf("unreadable storage response: %s", err)
	}
	if found.ID == "" {
		return nil, fmt.Errorf("the API returned no storage for id %s", storageId)
	}
	return &found, nil
}

// CreateStorage create a new storage
func (s *StorageServiceImpl) CreateStorage(createdModel StorageDTO) (string, error) {
	var apiPath = common.ApiPath.Storage(createdModel.VpcId)
	resp, err := s.client.SendPostRequest(apiPath, createdModel)

	if err != nil {
		return "", common.DecodeError(err)
	}

	var createStorageResponse struct {
		StorageId string `json:"storage_id"`
	}

	err = json.Unmarshal(resp, &createStorageResponse)

	if err != nil {
		return "", common.DecodeError(err)
	}

	return createStorageResponse.StorageId, nil
}

// CreateStorageAsync queues the creation of an EXTERNAL storage and returns
// its id at once. The storage then goes from a null status to ENABLED once
// the Celery task and the following sync finish; callers must wait for it.
func (s *StorageServiceImpl) CreateStorageAsync(createdModel StorageDTO) (string, error) {
	// This endpoint takes the infrastructure id of the policy, while the
	// resource schema holds the portal id returned by fptcloud_storage_policy.
	infraPolicyId, err := s.resolveInfraPolicyId(createdModel.VpcId, createdModel.StoragePolicyId)
	if err != nil {
		return "", err
	}

	vmId := ""
	if createdModel.InstanceId != nil {
		vmId = *createdModel.InstanceId
	}

	resp, err := s.client.SendPostRequest(common.ApiPath.CreateStorageAsync(createdModel.VpcId), map[string]interface{}{
		"name":              createdModel.Name,
		"size":              createdModel.SizeGb,
		"storage_policy_id": infraPolicyId,
		"description":       "",
		"vm_id":             vmId,
		"storage_type":      "DEFAULT",
		"snapshot_id":       "",
	})
	if err != nil {
		if isCreateRequestTimeout(err) {
			return "", fmt.Errorf("%w: %v", ErrCreateRequestTimeout, err)
		}
		var httpErr common.HTTPError
		if errors.As(err, &httpErr) {
			return "", fmt.Errorf("create storage failed (HTTP %d): %s", httpErr.Code, apiMessage([]byte(httpErr.Reason)))
		}
		return "", common.DecodeError(err)
	}

	var createResponse struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    *struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &createResponse); err != nil {
		return "", fmt.Errorf("create storage returned an unreadable response: %s", string(resp))
	}
	// The endpoint can answer 2xx with status=false, and can also return
	// without any body data (e.g. a failed quota check), so the HTTP code
	// alone does not prove a storage was queued.
	if !createResponse.Status {
		return "", fmt.Errorf("create storage failed: %s", apiMessage(resp))
	}
	if createResponse.Data == nil || createResponse.Data.Id == "" {
		return "", fmt.Errorf("create storage returned no storage id: %s", string(resp))
	}
	return createResponse.Data.Id, nil
}

// infraPolicyIds caches portal policy id -> infrastructure policy id for the
// life of the provider process. The mapping never changes for a policy, and
// the list behind it is read from the infrastructure (9-23 s on production),
// so a plan with many disks would otherwise pay that once per disk.
var infraPolicyIds sync.Map

func (s *StorageServiceImpl) resolveInfraPolicyId(vpcId string, policyId string) (string, error) {
	cacheKey := s.client.BaseURL.String() + "|" + vpcId + "|" + policyId
	if cached, ok := infraPolicyIds.Load(cacheKey); ok {
		return cached.(string), nil
	}

	resp, err := s.client.SendGetRequest(common.ApiPath.StoragePolicy(vpcId))
	if err != nil {
		return "", common.DecodeError(err)
	}
	var policies struct {
		Data []struct {
			Id      string `json:"id"`
			InfraId string `json:"infra_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &policies); err != nil {
		return "", common.DecodeError(err)
	}
	found := ""
	for _, p := range policies.Data {
		if p.InfraId != "" {
			infraPolicyIds.Store(s.client.BaseURL.String()+"|"+vpcId+"|"+p.Id, p.InfraId)
		}
		if p.Id == policyId {
			found = p.InfraId
		}
	}
	if found != "" {
		return found, nil
	}
	return "", fmt.Errorf("storage policy %s not found in vpc %s", policyId, vpcId)
}

// storageNotFoundCode is what the name lookup answers (inside an HTTP 500)
// when no storage has that name.
const storageNotFoundCode = "1501002"

var storageImportNotFoundCodes = []string{storageNotFoundCode, "120617"}

func isStorageNotFound(err error) bool {
	code := common.ApiErrorCode(err)
	for _, notFound := range storageImportNotFoundCodes {
		if code == notFound {
			return true
		}
	}
	return false
}

// ErrCreateRequestTimeout marks a create request whose outcome is unknown: the
// client or the gateway gave up waiting, so the storage may or may not have
// been queued. Callers must look it up by name instead of failing.
var ErrCreateRequestTimeout = errors.New("create storage request timed out")

// isCreateRequestTimeout reports whether a create request failed without an
// answer from the API: a client-side timeout, or a gateway 502/504 (a 502
// "invalid response from upstream" was measured on production 2026-09-17,
// after ~55 s, with no storage created). Any other status is a real answer.
func isCreateRequestTimeout(err error) bool {
	var httpErr common.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Code == http.StatusBadGateway || httpErr.Code == http.StatusGatewayTimeout
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// StorageNameLookup is the latest storage row carrying a name, if any.
type StorageNameLookup struct {
	Found  bool
	Id     string
	Status string
}

// LookupStorageByName returns the latest storage row with this name, whatever
// its status: deleted storages keep a DISABLED row, and a storage still being
// created has a null status.
func (s *StorageServiceImpl) LookupStorageByName(vpcId string, name string) (StorageNameLookup, error) {
	apiPath := common.ApiPath.Storage(vpcId) + utils.ToQueryParams(FindStorageDTO{Name: name})
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		var httpErr common.HTTPError
		if errors.As(err, &httpErr) && strings.Contains(httpErr.Reason, storageNotFoundCode) {
			return StorageNameLookup{}, nil
		}
		return StorageNameLookup{}, common.DecodeError(err)
	}
	var found Storage
	if err := json.Unmarshal(resp, &found); err != nil {
		return StorageNameLookup{}, common.DecodeError(err)
	}
	return StorageNameLookup{Found: found.ID != "", Id: found.ID, Status: found.Status}, nil
}

func apiMessage(body []byte) string {
	var parsed struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Message != "" {
		return parsed.Message
	}
	return string(body)
}

// UpdateStorage update a storage
func (s *StorageServiceImpl) UpdateStorage(vpcId string, storageId string, updatedModel UpdateStorageDTO) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.Storage(vpcId) + "/" + storageId
	_, err := s.client.SendPutRequest(apiPath, updatedModel)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}

// ResizeStorage resize a storage
func (s *StorageServiceImpl) ResizeStorage(vpcId string, resizeModel ResizeStorageDTO) (*common.SimpleResponse, error) {
	resp, err := s.client.SendPostRequest(common.ApiPath.ResizeStorage(vpcId), resizeModel)
	if err != nil {
		var httpErr common.HTTPError
		if errors.As(err, &httpErr) {
			return nil, fmt.Errorf("resize storage failed (HTTP %d): %s", httpErr.Code, apiMessage([]byte(httpErr.Reason)))
		}
		return nil, common.DecodeError(err)
	}

	var resizeResponse struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp, &resizeResponse); err != nil {
		return nil, fmt.Errorf("resize storage returned an unreadable response: %s", string(resp))
	}
	if !resizeResponse.Status {
		return nil, fmt.Errorf("resize storage failed: %s", apiMessage(resp))
	}

	return &common.SimpleResponse{Data: "Successfully", Status: "200"}, nil
}

// DeleteStorage delete a storage
func (s *StorageServiceImpl) DeleteStorage(vpcId string, storageId string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.Storage(vpcId) + "/" + storageId
	_, err := s.client.SendDeleteRequest(apiPath)

	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}

func (s *StorageServiceImpl) UpdateAttachedInstance(vpcId string, storageId string, instanceId *string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.StorageUpdateAttached(vpcId, storageId)

	_, err := s.client.SendPutRequest(apiPath, map[string]interface{}{
		"instance_id": instanceId,
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

// UpdateTags updates the tags associated with a storage
func (s *StorageServiceImpl) UpdateTags(vpcId string, storageId string, tagIds []string) (*common.SimpleResponse, error) {
	var apiPath = common.ApiPath.UpdateStorageTags(vpcId, storageId)
	payload := map[string][]string{
		"tag_ids": tagIds,
	}
	_, err := s.client.SendPutRequest(apiPath, payload)
	if err != nil {
		return nil, common.DecodeError(err)
	}

	var result = &common.SimpleResponse{
		Data:   "Successfully",
		Status: "200",
	}

	return result, nil
}
