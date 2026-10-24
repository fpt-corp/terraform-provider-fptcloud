package fptcloud_database

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakePortal imitates the backup and restore endpoints, with the delays the real portal has:
// a job is "creating"/"updating"/"deleting" for one list, the service is "processing"/"updating"/"deleting" for one read.
type fakePortal struct {
	mu sync.Mutex

	clusterStatus string
	typeDb        string
	isCluster     string

	backupEnabled int
	capacity      int
	isArchived    int
	retentionType string
	serviceId     int
	backupTicks   int
	backupTarget  struct{ status, capacity int }

	jobs      []*fakeJob
	nextJobId int

	busy        bool
	capacityErr string // body returned (as success HTTP) instead of enabling
	createFails bool   // a new job ends in created_fail

	restorePoints []map[string]string
	restored      []map[string]string
	flavors       []map[string]interface{}
	clusters      []map[string]string // clusters of the VPC
	nextCluster   int

	calls  []string
	bodies map[string][]string
}

type fakeJob struct {
	id        int
	name, typ string
	retention int
	minute    string
	hour      string
	days      string
	status    string
	runNow    int
	ticks     int
	after     string // status when ticks run out; "gone" removes the job
}

func newFakePortal() *fakePortal {
	return &fakePortal{
		clusterStatus: "running", typeDb: "psql_patroni", isCluster: "yes",
		retentionType: "quantity", nextJobId: 100, nextCluster: 1,
		bodies: map[string][]string{},
		restorePoints: []map[string]string{
			{"backup_part": "20261023-020002F", "backup_type": "full", "backup_state": "successful", "backup_size": "0", "start_time": "t1", "finish_time": "t2"},
			{"backup_part": "20261022-020002F", "backup_type": "full", "backup_state": "failed", "backup_size": "0", "start_time": "t1", "finish_time": "t2"},
		},
		flavors: []map[string]interface{}{
			{"flavor_id": "f-4c8g", "flavor_name": "4C8G", "flavor_vcpu": 4, "flavor_ram": 8192},
			{"flavor_id": "f-2c4g", "flavor_name": "2C4G", "flavor_vcpu": 2, "flavor_ram": 4096},
		},
	}
}

func (p *fakePortal) start(t *testing.T) *databaseApiClient {
	return newTestApi(t, http.HandlerFunc(p.serve))
}

func (p *fakePortal) callsLike(sub string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []string
	for _, c := range p.calls {
		if strings.Contains(c, sub) {
			out = append(out, c)
		}
	}
	return out
}

func (p *fakePortal) lastBody(call string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	b := p.bodies[call]
	if len(b) == 0 {
		return ""
	}
	return b[len(b)-1]
}

func (p *fakePortal) job(id int) *fakeJob {
	for _, j := range p.jobs {
		if j.id == id {
			return j
		}
	}
	return nil
}

func (p *fakePortal) tickJobs() {
	kept := p.jobs[:0]
	for _, j := range p.jobs {
		if j.ticks > 0 {
			j.ticks--
			if j.ticks == 0 {
				if j.after == "gone" {
					continue
				}
				j.status = j.after
			}
		}
		kept = append(kept, j)
	}
	p.jobs = kept
}

func (p *fakePortal) tickBackup() {
	if p.backupTicks > 0 {
		p.backupTicks--
		if p.backupTicks == 0 {
			p.backupEnabled = p.backupTarget.status
			p.capacity = p.backupTarget.capacity
			if p.backupEnabled == 0 {
				p.serviceId = 0
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	_ = json.NewEncoder(w).Encode(v)
}

func success(w http.ResponseWriter, msg string) {
	writeJSON(w, map[string]string{"message": msg, "type": "success"})
}

func (p *fakePortal) serve(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	raw, _ := io.ReadAll(r.Body)
	key := r.Method + " " + r.URL.Path
	p.calls = append(p.calls, key)
	p.bodies[key] = append(p.bodies[key], string(raw))
	path := strings.TrimPrefix(r.URL.Path, "/v1/xplat/database/")
	last := path[strings.LastIndex(path, "/")+1:]

	switch {
	case strings.HasPrefix(path, "management/cluster/detail/"):
		writeJSON(w, map[string]interface{}{"code": "200", "data": map[string]interface{}{
			"cluster": map[string]interface{}{
				"cluster_id": last, "vpc_id": "vpc-1", "status": p.clusterStatus, "type_db": p.typeDb, "is_cluster": p.isCluster,
				"storage_profile": "SSD", "node_cpu": 4, "node_ram": 8, "flavor_id": "f-4c8g",
			},
			"nodes": map[string]interface{}{"total": 1, "items": []interface{}{}},
		}})

	case strings.HasPrefix(path, "backup/detail/"):
		p.tickBackup()
		d := map[string]interface{}{
			"cluster_id": last, "backup_enabled": p.backupEnabled, "capacity": p.capacity, "is_archived": p.isArchived,
			"backup_retention_type": p.retentionType, "percent_backup_used": 10,
		}
		if p.serviceId != 0 {
			d["backup_service_id"] = p.serviceId
		}
		writeJSON(w, d)

	case path == "backup/service/capacity":
		if p.capacityErr != "" {
			_, _ = w.Write([]byte(p.capacityErr))
			return
		}
		var b map[string]interface{}
		_ = json.Unmarshal(raw, &b)
		target := int(b["capacity"].(float64))
		if p.backupEnabled == 0 {
			p.serviceId, p.isArchived = 229, int(b["is_archived"].(float64))
			p.backupEnabled, p.backupTicks = backupStatusProcessing, 2
			p.backupTarget.status = backupStatusEnabled
		} else {
			if target <= p.capacity {
				writeJSON(w, map[string]interface{}{"error_code": 160006, "message": "The backup capacity must be greater than the current size.", "type": "error"})
				return
			}
			p.backupEnabled, p.backupTicks = backupStatusUpdating, 2
			p.backupTarget.status = backupStatusEnabled
			p.isArchived = int(b["is_archived"].(float64))
		}
		p.backupTarget.capacity = target
		success(w, "Change backup capacity is being processed")

	case strings.HasPrefix(path, "backup/service/delete/"):
		p.backupEnabled, p.backupTicks = backupStatusDeleting, 2
		p.backupTarget.status, p.backupTarget.capacity = 0, 0
		success(w, "Backup service is being deleted")

	case strings.HasPrefix(path, "backup/service/job/list/"):
		p.tickJobs()
		jobs := []map[string]interface{}{}
		for _, j := range p.jobs {
			if p.backupEnabled != backupStatusEnabled && p.backupEnabled != backupStatusActive {
				break // the real portal lists no jobs while the service is changing
			}
			jobs = append(jobs, map[string]interface{}{
				"job_id": j.id, "backup_job_name": j.name, "backup_type": j.typ, "backup_retention": j.retention,
				"backup_schedule": map[string]string{"minute": j.minute, "hour": j.hour, "day_of_week": j.days},
				"status":          j.status, "is_run_now": j.runNow,
			})
		}
		writeJSON(w, map[string]interface{}{"jobs": jobs, "total": len(jobs)})

	case path == "backup/service/job/create":
		var b map[string]interface{}
		_ = json.Unmarshal(raw, &b)
		after := "ready"
		if p.createFails {
			after = "created_fail"
		}
		p.nextJobId++
		p.jobs = append(p.jobs, &fakeJob{
			id: p.nextJobId, name: b["backup_job_name"].(string), typ: b["backup_type"].(string),
			retention: int(b["backup_retention"].(float64)), minute: b["minute"].(string), hour: b["hour"].(string),
			days: b["day_of_week"].(string), status: "creating", ticks: 2, after: after,
		})
		success(w, "Creating new backup job is being processed")

	case path == "backup/service/job/update":
		var b map[string]interface{}
		_ = json.Unmarshal(raw, &b)
		j := p.job(int(b["job_id"].(float64)))
		if j == nil {
			writeJSON(w, map[string]interface{}{"error_code": 1, "message": "no such job", "type": "error"})
			return
		}
		// The portal ignores a new name; the schedule and retention are applied.
		j.retention, j.minute, j.hour, j.days = int(b["backup_retention"].(float64)), b["minute"].(string), b["hour"].(string), b["day_of_week"].(string)
		j.status, j.ticks, j.after = "updating", 2, "ready"
		success(w, "Updating current backup job is being processed")
		if j.typ == "full" {
			for _, other := range p.jobs {
				if other.typ == "diff" {
					other.retention = j.retention
				}
			}
		}

	case strings.HasPrefix(path, "backup/service/job/delete/"):
		id, _ := strconv.Atoi(last)
		j := p.job(id)
		if j != nil {
			j.status, j.ticks, j.after = "deleting", 2, "gone"
		}
		success(w, "Current Backup Job will be deleted")

	case strings.HasPrefix(path, "backup/service/job/runnow/"):
		id, _ := strconv.Atoi(last)
		if j := p.job(id); j != nil {
			j.runNow = 1
		}
		success(w, "Current job is being processed")

	case strings.HasPrefix(path, "restore/list-backup-request/"):
		data := []map[string]string{{"task_name": "Create Backup Service", "status": "successful"}}
		if p.busy {
			data = append(data, map[string]string{"task_name": "Run full backup", "status": "pending"})
		}
		writeJSON(w, map[string]interface{}{"code": "200", "data": data, "metadata": map[string]int{"total_items": len(data)}})

	case strings.HasPrefix(path, "restore/list-restore-point/"):
		writeJSON(w, map[string]interface{}{"code": "200", "data": p.restorePoints, "metadata": map[string]int{"total_items": len(p.restorePoints)}})

	case strings.HasPrefix(path, "restore/list-restore-cluster/"):
		// a restored cluster is "processing" for its first read, then running
		for _, c := range p.restored {
			if c["status"] == "processing" {
				c["status"] = "running"
			}
		}
		writeJSON(w, map[string]interface{}{"code": "200", "data": p.restored, "metadata": map[string]int{"total_items": len(p.restored)}})

	case path == "restore/create":
		var b map[string]interface{}
		_ = json.Unmarshal(raw, &b)
		p.nextCluster++
		p.restored = append(p.restored, map[string]string{
			"cluster_id": fmt.Sprintf("restored%d", p.nextCluster), "cluster_name": b["restore_name"].(string), "status": "processing",
			"version": "17", "type_db": p.typeDb,
		})
		writeJSON(w, map[string]interface{}{"code": "200", "message": "Successful", "data": []interface{}{}, "type": "success"})

	case strings.HasPrefix(path, "provision/delete/"):
		for _, c := range p.restored {
			if c["cluster_id"] == last {
				c["status"] = "deleted"
			}
		}
		success(w, "Cluster is being deleted")

	case strings.HasPrefix(path, "configure_management/get_list_flavor_v2"):
		data := p.flavors
		if r.URL.Query().Get("is_ops") == "no" {
			data = nil
		}
		writeJSON(w, map[string]interface{}{"code": "200", "data": data})

	case strings.HasPrefix(path, "management/cluster/list/"):
		writeJSON(w, map[string]interface{}{"code": "200", "data": p.clusters, "metadata": map[string]int{"total_items": len(p.clusters)}})

	default:
		http.Error(w, "not found: "+key, http.StatusNotFound)
	}
}
