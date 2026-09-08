package fptcloud_backup_veeam

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// Trạng thái đang xử lý. DISABLING_SCHEDULE và ENABLING_SCHEDULE nằm trong đây
// dù provider không gọi toggle: job có thể đang ở hai trạng thái đó do người
// khác bấm trên portal đúng lúc Terraform chạy.
var pendingStatuses = []string{
	"CREATING", "UPDATING", "DELETING", "STARTING",
	"DISABLING_SCHEDULE", "ENABLING_SCHEDULE",
}

// CREATE_FAILED nghĩa là job chưa từng lên Veeam (create hỏng, hoặc thua
// tie-break khi hai job tranh cùng một máy ảo). ERROR nghĩa là job có tồn tại
// nhưng đang lỗi. Cả hai đều là thất bại với Terraform.
var failedStatuses = []string{"ERROR", "CREATE_FAILED", "FAILED"}

// settledStatuses là các trạng thái nghỉ coi như thành công. NOT_AVAILABLE là
// trạng thái BÌNH THƯỜNG của job vừa tạo và chưa chạy lần nào.
var settledStatuses = []string{"NOT_AVAILABLE", "WORKING", "SUCCESS", "WARNING"}

func containsStatus(list []string, status string) bool {
	for _, s := range list {
		if s == status {
			return true
		}
	}
	return false
}

func IsPendingStatus(status string) bool { return containsStatus(pendingStatuses, status) }

func IsFailedStatus(status string) bool { return containsStatus(failedStatuses, status) }

// WaitForJobSettled chờ job rời khỏi trạng thái pending.
//
// Poll qua endpoint LIST, không phải detail: detail trả 200 ngay từ lúc row DB
// được ghi - trước cả khi Celery task chạy - nên poll bằng detail sẽ "thành
// công" tức thì và bỏ lọt cả CREATE_FAILED.
func WaitForJobSettled(ctx context.Context, svc BackupVeeamService, vpcId string, jobId string, name string, timeout time.Duration) (*JobListItem, error) {
	stateConf := &retry.StateChangeConf{
		Pending: pendingStatuses,
		Target:  settledStatuses,
		Refresh: func() (interface{}, string, error) {
			item, err := svc.FindJobInList(vpcId, jobId, name)
			if err != nil {
				return nil, "", err
			}
			if item == nil {
				return nil, "", fmt.Errorf("không tìm thấy backup job %s trong danh sách job của VPC", jobId)
			}
			if IsFailedStatus(item.Status) {
				return nil, "", failureMessage(item.Status, jobId)
			}
			return item, item.Status, nil
		},
		Timeout:                   timeout,
		Delay:                     3 * time.Second,
		MinTimeout:                3 * time.Second,
		ContinuousTargetOccurence: 1,
	}

	raw, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return nil, err
	}
	item, _ := raw.(*JobListItem)
	return item, nil
}

func failureMessage(status string, jobId string) error {
	if status == "CREATE_FAILED" {
		return fmt.Errorf("backup job %s ở trạng thái CREATE_FAILED — job chưa được tạo trên Veeam. "+
			"Nếu apply tạo nhiều job cùng lúc, nguyên nhân thường là hai job tranh cùng một máy ảo; "+
			"chạy lại terraform apply, hoặc dùng -parallelism=1", jobId)
	}
	return fmt.Errorf("backup job %s ở trạng thái %s — job tồn tại nhưng đang lỗi, kiểm tra trên portal", jobId, status)
}

// WaitForJobGone chờ tới khi detail không còn trả về job. Đây là trường hợp
// DUY NHẤT poll bằng detail được, vì tín hiệu cần là "job biến mất" chứ không
// phải status.
func WaitForJobGone(ctx context.Context, svc BackupVeeamService, vpcId string, jobId string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending: []string{"EXISTS"},
		Target:  []string{"GONE"},
		Refresh: func() (interface{}, string, error) {
			detail, err := svc.GetJobDetail(vpcId, jobId)
			if err != nil {
				return nil, "", err
			}
			if detail == nil {
				return jobId, "GONE", nil
			}
			return detail, "EXISTS", nil
		},
		Timeout:    timeout,
		Delay:      3 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForStateContext(ctx)
	return err
}
