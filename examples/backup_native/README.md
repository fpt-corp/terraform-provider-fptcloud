# Backup Native example

A runnable configuration that exercises every Backup Native resource and data source: a VM backup job, a volume backup job, a manual run of each, their restore point lists, a VM and a volume restore, and a purge of the VM's restore points. Use it to try the provider by hand against a real VPC.

Applying it creates real jobs and restores, which may incur cost. The purge deletes restore points for good.

## What you need

- A VPC with Backup Native enabled.
- A throwaway VM that is not in another backup job. Destroying the VM job deletes the VM's restore points.
- An FKE persistent volume (name starts with `pv-fke`) that is not in another backup job.
- The ID of a notification method, for example from the `fptcloud_alert_notification_methods` data source.

Everything is in `backup_native.tf`: the provider, the VPC lookup and the shared variables, then the VM backup (v1) and the volume backup (v2). Leave `vm_id` or `volume_id` empty to skip that half.

## Run it

Run every command from this folder. Nothing needs to be pushed or published: `.terraformrc` makes Terraform load the provider from `.bin/` (git-ignored) instead of the registry.

1. Build the provider into `.bin/`. Repeat after every code change:

   ```sh
   go build -o .bin/terraform-provider-fptcloud ../..
   ```

2. Copy `.env.example` to `.env` (git-ignored), fill in the credentials, and load it:

   ```sh
   source .env
   ```

   This also sets `TF_CLI_CONFIG_FILE` to `.terraformrc`, for this shell only. Run `unset TF_CLI_CONFIG_FILE` to go back to the published provider.

3. Copy `terraform.tfvars.example` to `terraform.tfvars` and fill in the IDs. `*.tfvars` is git-ignored. Set `vpc_name` to the VPC's name in the portal; the `fptcloud_vpc` data source looks up its ID.
4. Create the jobs and read the lists. `terraform init` is not needed, and Terraform warns that development overrides are in effect:

   ```sh
   terraform plan
   terraform apply
   ```

   The `vm_jobs` and `volume_jobs` outputs show the new jobs. The restore point outputs stay empty until a job has run.

5. To run a job now instead of waiting for its schedule, set `vm_backup_run` or `volume_backup_run` to the time you ask for the run, for example `"2026-10-26 09:00:00"`, and apply. Apply waits for the run to end; the `vm_job_run` and `volume_job_run` outputs show its status. Set a new time to run it again; the state keeps the time of the last request. Use the same convention for every `triggers` value (runs, restores, purge), and write it as a literal: `timestamp()` changes on every plan and repeats the action on every apply.
6. After a backup has run, set `vm_restore` or `volume_restore` to the time you ask for the restore and apply again. The restore uses the newest restore point. The `vm_restore_status` and `volume_restore_status` outputs show the result. Clear the variable before the next manual backup, or the apply after it restores again.
7. To delete every restore point of `vm_id`, under any job, clear `vm_restore`, set `vm_restore_points_purge` to the time you ask for the purge, and apply. Apply waits for the deletion to end; the `vm_restore_points_purge` output shows how many restore points there were. A new time purges again.
8. Run `terraform destroy` to delete the jobs. Destroying a restore only removes it from state; the restored VM or volume stays and must be deleted in the portal.

## Things to check

| Check | How |
|---|---|
| Schedule change | Change `full_backup_frequency` in `backup_native.tf` (weekly needs `full_backup_day` 0-6 with 0 as Monday, monthly 1-31, daily none) and apply. The next plan shows no changes. |
| Schedule off and on | Add `schedule_enabled = false` to a job and apply, then remove it and apply. |
| Import | `terraform import 'fptcloud_backup_native_vm_job.example[0]' vpc/<vpc_id>/backup_native_vm_job/<job_id>` after `terraform state rm`. |
| Manual backup while a restore is on | The restores use the newest restore point, so the apply after a manual backup runs them again; with `vm_restore_mode = "overwrite"` that overwrites `vm_id` again. Clear `vm_restore` and `volume_restore` first, or apply only the backup: `terraform apply -target='fptcloud_backup_native_vm_job_run.example[0]'`. |
| Overwrite restore | Set `vm_restore_mode = "overwrite"`. This replaces the disk of `vm_id`. |
| Requests sent | `TF_LOG=DEBUG terraform apply` logs every API call; all of them are user API paths. |
| VM sync | After each VM action the API accepts (job create, update, delete, manual backup, restore, purge), the provider calls `GET .../osp_native_backup/sync` once, so the portal shows the change. It does not wait for the sync, and a failed sync is a warning, not an error. The volume API has no sync. |

Run one restore per VM or volume at a time.
