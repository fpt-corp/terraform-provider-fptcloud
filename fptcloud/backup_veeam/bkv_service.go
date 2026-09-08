package fptcloud_backup_veeam

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	common "terraform-provider-fptcloud/commons"
)

// listPageSize là page_size dùng cho mọi lần gọi list. Đủ lớn để không phải
// phân trang trong thực tế; nếu tenant vượt quá thì FindJobInList sẽ không
// thấy job và Terraform báo drift - chấp nhận được ở phiên bản này.
const listPageSize = 1000

type BackupVeeamService interface {
	CreateJob(vpcId string, payload CreateJobPayload) (JobMutationResponse, error)
	UpdateJob(vpcId string, jobId string, payload CreateJobPayload) (JobMutationResponse, error)
	GetJobDetail(vpcId string, jobId string) (*JobDetail, error)
	ListJobs(vpcId string, name string, status string) (JobListResponse, error)
	FindJobInList(vpcId string, jobId string, name string) (*JobListItem, error)
	DeleteJob(vpcId string, jobId string) error
	ListInstances(vpcId string, notBackup bool, jobId string, status string) (InstanceListResponse, error)
}

type backupVeeamServiceImpl struct {
	client *common.Client
}

func NewBackupVeeamService(client *common.Client) BackupVeeamService {
	return &backupVeeamServiceImpl{client: client}
}

// describeErrorType dịch error_type của API thành câu nói rõ khách phải làm gì.
// Bảy mã lấy từ class BackupErrorType phía backend.
func describeErrorType(errorType string, message string) string {
	hints := map[string]string{
		"duplicateVm":          "một máy ảo chỉ được thuộc một backup job đang hoạt động; dùng data source fptcloud_backup_veeam_instances với not_backup = true để lấy danh sách máy ảo còn gán được",
		"vmNotFound":           "máy ảo không tồn tại; kiểm tra lại vm_ids",
		"vmNotInVpc":           "máy ảo không thuộc VPC này; kiểm tra lại vpc_id và vm_ids",
		"reachLimitQuota":      "đã hết quota backup của tenant; cần liên hệ FPT Cloud để tăng quota, Terraform không xử lý được",
		"jobNotEligible":       "job đang ở trạng thái không cho phép thao tác; chờ job về trạng thái ổn định rồi apply lại",
		"requestInProgress":    "một yêu cầu giống hệt đang được xử lý; chạy lại terraform apply sau ít phút",
		"idempotencyKeyReused": "idempotency key đã dùng cho nội dung khác (lỗi phía provider, không phải cấu hình của bạn)",
	}

	if hint, ok := hints[errorType]; ok {
		return fmt.Sprintf("%s (%s): %s", message, errorType, hint)
	}
	return fmt.Sprintf("%s (%s)", message, errorType)
}

// checkMutationResponse là chốt chặn duy nhất cho việc API trả HTTP 200 khi
// thất bại. Kiểm error_type TRƯỚC status, vì nhánh lỗi nghiệp vụ không set
// status - zero value false vẫn bắt được, nhưng error_type cho thông báo tốt hơn.
func checkMutationResponse(resp JobMutationResponse) error {
	if resp.ErrorType != "" {
		return fmt.Errorf("%s", describeErrorType(resp.ErrorType, resp.Message))
	}
	if !resp.Status {
		message := resp.Message
		if message == "" {
			message = "API từ chối yêu cầu nhưng không nêu lý do"
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}

// BuildIdempotencyKey sinh key TẤT ĐỊNH theo nội dung. Không dùng UUID ngẫu
// nhiên: retry phải mang lại đúng key thì backend mới nhận ra là cùng một yêu
// cầu và trả kết quả đã cache thay vì tạo job thứ hai.
func BuildIdempotencyKey(vpcId string, payload CreateJobPayload) string {
	// Key không được phụ thuộc chính nó, nên bỏ trường này ra khi băm.
	payload.IdempotencyKey = ""

	body, err := json.Marshal(payload)
	if err != nil {
		body = []byte(payload.Name)
	}
	sum := sha256.Sum256(append([]byte(vpcId+"|"), body...))
	return "tf-" + hex.EncodeToString(sum[:16])
}

// decorateServerError thêm gợi ý cho HTTP 500 lúc tạo/sửa job. Backend không
// null-check tenant, nên VPC chưa bật dịch vụ Backup Veeam sẽ trả 500 với
// thông báo vô nghĩa - đây là lỗi khách gặp ngay lần đầu dùng.
func decorateServerError(action string, err error) error {
	if strings.Contains(err.Error(), "500") {
		return fmt.Errorf("%s thất bại (HTTP 500): %v — kiểm tra dịch vụ Backup Veeam đã được bật cho VPC này chưa", action, err)
	}
	return fmt.Errorf("%s thất bại: %v", action, err)
}

func (s *backupVeeamServiceImpl) CreateJob(vpcId string, payload CreateJobPayload) (JobMutationResponse, error) {
	payload.IdempotencyKey = BuildIdempotencyKey(vpcId, payload)

	raw, err := s.client.SendPostRequest(common.ApiPath.BackupVeeamCreateJob(vpcId), payload)
	if err != nil {
		return JobMutationResponse{}, decorateServerError("tạo backup job", err)
	}

	var result JobMutationResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return JobMutationResponse{}, fmt.Errorf("không đọc được phản hồi tạo backup job: %v", err)
	}
	if err := checkMutationResponse(result); err != nil {
		return result, err
	}
	return result, nil
}

func (s *backupVeeamServiceImpl) UpdateJob(vpcId string, jobId string, payload CreateJobPayload) (JobMutationResponse, error) {
	raw, err := s.client.SendPostRequest(common.ApiPath.BackupVeeamUpdateJob(vpcId, jobId), payload)
	if err != nil {
		return JobMutationResponse{}, decorateServerError("cập nhật backup job", err)
	}

	var result JobMutationResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return JobMutationResponse{}, fmt.Errorf("không đọc được phản hồi cập nhật backup job: %v", err)
	}
	if err := checkMutationResponse(result); err != nil {
		return result, err
	}
	return result, nil
}

func (s *backupVeeamServiceImpl) GetJobDetail(vpcId string, jobId string) (*JobDetail, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupVeeamJobDetail(vpcId, jobId))
	if err != nil {
		// 404 nghĩa là job không còn - đây là drift, không phải lỗi.
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, fmt.Errorf("đọc chi tiết backup job thất bại: %v", err)
	}

	var result JobDetailResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("không đọc được chi tiết backup job: %v", err)
	}
	// Blueprint trả {"error": "can't get job detail."} - có thể kèm 404, nhưng
	// phòng cả trường hợp trả 200 với data rỗng.
	if result.Data.Id == "" {
		return nil, nil
	}
	return &result.Data, nil
}

func (s *backupVeeamServiceImpl) ListJobs(vpcId string, name string, status string) (JobListResponse, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupVeeamListJobs(vpcId, 1, listPageSize, name, status))
	if err != nil {
		return JobListResponse{}, fmt.Errorf("liệt kê backup job thất bại: %v", err)
	}

	var result JobListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return JobListResponse{}, fmt.Errorf("không đọc được danh sách backup job: %v", err)
	}
	return result, nil
}

// FindJobInList tìm job theo ID trong endpoint list - nguồn DUY NHẤT có status
// và enabled (detail không có hai field này).
//
// Lọc theo name trước cho nhẹ, nhưng nếu không thấy thì PHẢI quét lại không
// filter: job có thể đã bị đổi tên trên portal. Thiếu bước này thì Terraform
// tưởng job đã mất, tạo lại, và đâm thẳng vào lỗi duplicateVm.
func (s *backupVeeamServiceImpl) FindJobInList(vpcId string, jobId string, name string) (*JobListItem, error) {
	if name != "" {
		list, err := s.ListJobs(vpcId, name, "")
		if err != nil {
			return nil, err
		}
		if item := pickJobById(list.Items, jobId); item != nil {
			return item, nil
		}
	}

	list, err := s.ListJobs(vpcId, "", "")
	if err != nil {
		return nil, err
	}
	return pickJobById(list.Items, jobId), nil
}

func pickJobById(items []JobListItem, jobId string) *JobListItem {
	for i := range items {
		if items[i].Id == jobId {
			return &items[i]
		}
	}
	return nil
}

// DeleteJob bỏ qua lỗi 500: backend truy cập backup_job.name mà không
// null-check, nên xoá một job đã bị xoá trả 500 chứ không phải 404. Với
// Terraform thì "job không còn" là kết quả mong muốn của Delete.
func (s *backupVeeamServiceImpl) DeleteJob(vpcId string, jobId string) error {
	raw, err := s.client.SendDeleteRequest(common.ApiPath.BackupVeeamDeleteJob(vpcId, jobId))
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "500") {
			return nil
		}
		return fmt.Errorf("xoá backup job thất bại: %v", err)
	}

	var result JobMutationResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		// Delete thành công nhưng body lạ - không coi là lỗi.
		return nil
	}
	return checkMutationResponse(result)
}

func (s *backupVeeamServiceImpl) ListInstances(vpcId string, notBackup bool, jobId string, status string) (InstanceListResponse, error) {
	raw, err := s.client.SendGetRequest(common.ApiPath.BackupVeeamInstances(vpcId, notBackup, jobId, status))
	if err != nil {
		return InstanceListResponse{}, fmt.Errorf("liệt kê máy ảo gán được vào backup job thất bại: %v", err)
	}

	var result InstanceListResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return InstanceListResponse{}, fmt.Errorf("không đọc được danh sách máy ảo: %v", err)
	}
	return result, nil
}
