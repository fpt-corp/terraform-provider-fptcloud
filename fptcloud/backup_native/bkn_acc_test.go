package fptcloud_backup_native_test

import (
	"fmt"
	"os"
	"testing"

	test_helper "terraform-provider-fptcloud/commons/test-helper"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func requireEnv(t *testing.T, names ...string) map[string]string {
	t.Helper()
	values := map[string]string{}
	for _, name := range append([]string{"FPTCLOUD_TOKEN", "FPTCLOUD_TENANT_NAME", "FPTCLOUD_REGION"}, names...) {
		value := os.Getenv(name)
		if value == "" {
			t.Skipf("%s must be set to run this acceptance test", name)
		}
		values[name] = value
	}
	return values
}

func importId(kind string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs := s.RootModule().Resources["fptcloud_backup_native_"+kind+".test"]
		return fmt.Sprintf("vpc/%s/backup_native_%s/%s", rs.Primary.Attributes["vpc_id"], kind, rs.Primary.ID), nil
	}
}

// TestAccBackupNativeVmJob_basic creates, changes and imports a VM backup job.
// BKN_VM_ID must not be in another backup job. Destroying the job deletes the
// VM's restore points as long as the backend's delete endpoint does so: use a
// VM whose restore points can be lost.
func TestAccBackupNativeVmJob_basic(t *testing.T) {
	env := requireEnv(t, "VPC_ID", "BKN_VM_ID", "BKN_NOTIFICATION_ID")
	config := func(frequency string, day string, enabled bool) string {
		return fmt.Sprintf(`
resource "fptcloud_backup_native_vm_job" "test" {
  vpc_id                  = %q
  name                    = "tfacc-vm-job"
  description             = "terraform acceptance test"
  instance_ids            = [%q]
  notification_method_ids = [%q]
  retention               = 3
  schedule_enabled        = %t
  schedule {
    full_backup_frequency = %q
    %s
    time                  = "02:00"
    time_zone             = "UTC"
  }
}

data "fptcloud_backup_native_vm_jobs" "test" {
  vpc_id     = fptcloud_backup_native_vm_job.test.vpc_id
  name       = fptcloud_backup_native_vm_job.test.name
}
`, env["VPC_ID"], env["BKN_VM_ID"], env["BKN_NOTIFICATION_ID"], enabled, frequency, day)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { test_helper.TestPreCheck(t) },
		ProviderFactories: test_helper.TestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("daily", "", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("fptcloud_backup_native_vm_job.test", "id"),
					resource.TestCheckResourceAttr("fptcloud_backup_native_vm_job.test", "schedule.0.time", "02:00"),
					resource.TestCheckResourceAttr("data.fptcloud_backup_native_vm_jobs.test", "jobs.#", "1"),
				),
			},
			{
				Config: config("weekly", "full_backup_day = 0", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fptcloud_backup_native_vm_job.test", "schedule.0.full_backup_day", "0"),
					resource.TestCheckResourceAttr("fptcloud_backup_native_vm_job.test", "schedule_enabled", "false"),
				),
			},
			{
				ResourceName:      "fptcloud_backup_native_vm_job.test",
				ImportState:       true,
				ImportStateIdFunc: importId("vm_job"),
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccBackupNativeVolumeJob_basic creates a volume job with a description,
// moves it from monthly to daily (the stale-day case) and imports it.
// BKN_VOLUME_ID must be an FKE persistent volume (pv-fke*) not in another job.
func TestAccBackupNativeVolumeJob_basic(t *testing.T) {
	env := requireEnv(t, "VPC_ID", "BKN_VOLUME_ID", "BKN_NOTIFICATION_ID")
	config := func(frequency string, day string) string {
		return fmt.Sprintf(`
resource "fptcloud_backup_native_volume_job" "test" {
  vpc_id                  = %q
  name                    = "tfacc-volume-job"
  description             = "terraform acceptance test"
  volume_ids              = [%q]
  notification_method_ids = [%q]
  retention               = 3
  schedule {
    full_backup_frequency = %q
    %s
    time                  = "03:30"
    time_zone             = "Asia/Ho_Chi_Minh"
  }
}
`, env["VPC_ID"], env["BKN_VOLUME_ID"], env["BKN_NOTIFICATION_ID"], frequency, day)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { test_helper.TestPreCheck(t) },
		ProviderFactories: test_helper.TestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("monthly", "full_backup_day = 31"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fptcloud_backup_native_volume_job.test", "description", "terraform acceptance test"),
					resource.TestCheckResourceAttr("fptcloud_backup_native_volume_job.test", "schedule.0.full_backup_day", "31"),
				),
			},
			{
				Config: config("daily", ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fptcloud_backup_native_volume_job.test", "schedule.0.full_backup_frequency", "daily"),
					resource.TestCheckResourceAttr("fptcloud_backup_native_volume_job.test", "schedule.0.full_backup_day", "0"),
				),
			},
			{
				ResourceName:      "fptcloud_backup_native_volume_job.test",
				ImportState:       true,
				ImportStateIdFunc: importId("volume_job"),
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccBackupNativeRestorePoints_basic reads the restore points of
// BKN_VM_ID and BKN_VOLUME_ID. Either list may be empty.
func TestAccBackupNativeRestorePoints_basic(t *testing.T) {
	env := requireEnv(t, "VPC_ID", "BKN_VM_ID", "BKN_VOLUME_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { test_helper.TestPreCheck(t) },
		ProviderFactories: test_helper.TestProviderFactories,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
data "fptcloud_backup_native_vm_restore_points" "test" {
  vpc_id = %q
  vm_id  = %q
}

data "fptcloud_backup_native_volume_restore_points" "test" {
  vpc_id    = %q
  volume_id = %q
}
`, env["VPC_ID"], env["BKN_VM_ID"], env["VPC_ID"], env["BKN_VOLUME_ID"]),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("data.fptcloud_backup_native_vm_restore_points.test", "restore_points.#"),
				resource.TestCheckResourceAttrSet("data.fptcloud_backup_native_volume_restore_points.test", "restore_points.#"),
			),
		}},
	})
}

// TestAccBackupNativeVmJobRun_basic creates a VM backup job, runs it once, then
// changes triggers to run it a second time. It runs two real backups, and each
// creates a restore point of BKN_VM_ID. BKN_VM_ID must not be in another job.
func TestAccBackupNativeVmJobRun_basic(t *testing.T) {
	env := requireEnv(t, "VPC_ID", "BKN_VM_ID", "BKN_NOTIFICATION_ID")
	config := func(run string) string {
		return fmt.Sprintf(`
resource "fptcloud_backup_native_vm_job" "test" {
  vpc_id                  = %q
  name                    = "tfacc-vm-backup"
  instance_ids            = [%q]
  notification_method_ids = [%q]
  retention               = 3
  schedule {
    full_backup_frequency = "daily"
    time                  = "02:00"
  }
}

resource "fptcloud_backup_native_vm_job_run" "test" {
  vpc_id   = fptcloud_backup_native_vm_job.test.vpc_id
  job_id   = fptcloud_backup_native_vm_job.test.id
  triggers = { run = %q }
}
`, env["VPC_ID"], env["BKN_VM_ID"], env["BKN_NOTIFICATION_ID"], run)
	}
	check := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("fptcloud_backup_native_vm_job_run.test", "status", "Success"),
		resource.TestCheckResourceAttrSet("fptcloud_backup_native_vm_job_run.test", "started_at"),
	)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { test_helper.TestPreCheck(t) },
		ProviderFactories: test_helper.TestProviderFactories,
		Steps:             []resource.TestStep{{Config: config("1"), Check: check}, {Config: config("2"), Check: check}},
	})
}

// TestAccBackupNativeVolumeJobRun_basic creates a volume backup job, runs it
// once, then changes triggers to run it a second time. It runs two real
// backups, and each creates a restore point of BKN_VOLUME_ID, which must be an
// FKE persistent volume (pv-fke*) not in another job.
func TestAccBackupNativeVolumeJobRun_basic(t *testing.T) {
	env := requireEnv(t, "VPC_ID", "BKN_VOLUME_ID", "BKN_NOTIFICATION_ID")
	config := func(run string) string {
		return fmt.Sprintf(`
resource "fptcloud_backup_native_volume_job" "test" {
  vpc_id                  = %q
  name                    = "tfacc-volume-backup"
  volume_ids              = [%q]
  notification_method_ids = [%q]
  retention               = 3
  schedule {
    full_backup_frequency = "daily"
    time                  = "03:30"
  }
}

resource "fptcloud_backup_native_volume_job_run" "test" {
  vpc_id   = fptcloud_backup_native_volume_job.test.vpc_id
  job_id   = fptcloud_backup_native_volume_job.test.id
  triggers = { run = %q }
}
`, env["VPC_ID"], env["BKN_VOLUME_ID"], env["BKN_NOTIFICATION_ID"], run)
	}
	check := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("fptcloud_backup_native_volume_job_run.test", "status", "success"),
		resource.TestCheckResourceAttrSet("fptcloud_backup_native_volume_job_run.test", "started_at"),
	)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { test_helper.TestPreCheck(t) },
		ProviderFactories: test_helper.TestProviderFactories,
		Steps:             []resource.TestStep{{Config: config("1"), Check: check}, {Config: config("2"), Check: check}},
	})
}

// TestAccBackupNativeVmRestorePointsPurge_basic deletes every restore point of
// BKN_PURGE_VM_ID, under any job, then checks that none is left. It has its
// own variable so that no other test's VM loses its restore points.
func TestAccBackupNativeVmRestorePointsPurge_basic(t *testing.T) {
	env := requireEnv(t, "VPC_ID", "BKN_PURGE_VM_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { test_helper.TestPreCheck(t) },
		ProviderFactories: test_helper.TestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "fptcloud_backup_native_vm_restore_points_purge" "test" {
  vpc_id = %q
  vm_id  = %q
}
`, env["VPC_ID"], env["BKN_PURGE_VM_ID"]),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fptcloud_backup_native_vm_restore_points_purge.test", "status", "Success"),
					resource.TestCheckResourceAttrSet("fptcloud_backup_native_vm_restore_points_purge.test", "restore_point_count"),
				),
			},
			{
				Config: fmt.Sprintf(`
data "fptcloud_backup_native_vm_restore_points" "test" {
  vpc_id = %q
  vm_id  = %q
}
`, env["VPC_ID"], env["BKN_PURGE_VM_ID"]),
				Check: resource.TestCheckResourceAttr("data.fptcloud_backup_native_vm_restore_points.test", "restore_points.#", "0"),
			},
		},
	})
}
