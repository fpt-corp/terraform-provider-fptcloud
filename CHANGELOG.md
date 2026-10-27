## [0.3.87] - 2026-10-28

### Resource

- Feat: add `fptcloud_backup_native_vm_job` to manage VM backup jobs of the Backup Native service: the instances, the retention (1-1000 restore points per instance), the EMAIL notification methods and a `daily`, `weekly` or `monthly` schedule at a time of day in a time zone. `schedule_enabled = false` turns the schedule off. Import with `vpc/<vpc_id>/backup_native_vm_job/<job_id>`
- Feat: add `fptcloud_backup_native_volume_job` to manage backup jobs of FKE persistent volumes (names starting with `pv-fke`), with the same arguments. Import with `vpc/<vpc_id>/backup_native_volume_job/<job_id>`
- Feat: add `fptcloud_backup_native_vm_job_run` and `fptcloud_backup_native_volume_job_run` to run a backup job once, outside its schedule, and wait for the run to end. A job that is still running a backup, a restore or a change fails the apply without starting a run
- Feat: add `fptcloud_backup_native_vm_restore` to restore a VM onto itself (`overwrite`) or into a new VM (`keep`) and wait for the restore to finish. `restore_point_created_at` takes the `created_at` of a restore point; the backend picks the closest one within 10 minutes
- Feat: add `fptcloud_backup_native_volume_restore` to restore a volume restore point into a new volume and wait for the restore to finish
- Feat: add `fptcloud_backup_native_vm_restore_points_purge` to delete every restore point of a VM, under any job, past or current, and free their backup quota; the VM's next backup is a full backup. It waits while a backup or a restore of the VM runs, retries while the API answers busy, waits for the deletion to end and warns when restore points remain. `restore_point_count = 0` means there was nothing to delete and no request was sent
- Feat: the job runs, the restores and the purge run again when a value in `triggers` changes. Any string works; the time of the request, for example `2026-10-26 09:00:00`, is recommended so the state records it. Avoid `timestamp()`: it changes on every plan and repeats the action on every apply. Destroying these resources only removes them from the state
- Feat: once a job run, a restore or a purge has started, it is saved whatever the outcome, and the next apply does not run it again. A failure, a timeout or an interrupted apply (Ctrl+C) ends the apply with a warning, not an error. `status` keeps the API's value (`Success`, `Failed`, `error`, `failed`), or `tf_timeout`, `tf_interrupted` or `tf_failed` when the API has none; `message` says why. Each later plan or apply checks `tf_timeout` and `tf_interrupted` once and stores the API's outcome when the action has ended. A job run's synced status may come from a later scheduled run of the job. Press Ctrl+C only once: a second one stops Terraform before it saves, and the next apply runs the action again
- Feat: after each VM action the API accepts (job create, update or delete, manual backup, restore, purge), the provider asks the portal to sync the VM backups once, so the portal shows the change. It does not wait for the sync, and a failed sync is a warning. The volume API has no sync
- Feat: a VM backup job can be updated while a backup or a restore runs; the update waits only for another update to finish
- Feat: creating a job retries the lookup of the new job by name, so a passing read error does not leave a created job out of the state
- Feat: API errors carry a hint when there is something to do: Backup Native not enabled for the VPC, backup quota exhausted, a VM or volume already in another job, a volume that is not an FKE persistent volume, a busy job or restore point

### Datasource

- Feat: add `fptcloud_backup_native_vm_jobs` and `fptcloud_backup_native_volume_jobs` to list the backup jobs of a VPC, or only the job with a given `name`
- Feat: add `fptcloud_backup_native_vm_restore_points` and `fptcloud_backup_native_volume_restore_points` to list restore points, newest first. Restore points of a deleted volume backup job are listed too

### Docs

- Add docs and examples for every Backup Native resource and data source, and a runnable configuration in `examples/backup_native` to try them against a real VPC

## [0.3.86] - 2026-10-27

### Datasource

- Feat: add `fptcloud_database_resources` to list every database cluster and DB proxy of a VPC (`vpc_id`), with the nodes, CPU, RAM, disk, backup and PITR usage of each cluster and its DB proxies, and the totals of the VPC

### Resource

- Feat: `is_cluster` of `fptcloud_database` accepts only `yes` or `no`; any other value fails `terraform plan`

## [0.3.85] - 2026-10-26

### Datasource

- Feat: add `fptcloud_managed_kubernetes_engines_v1` to list all managed FKE clusters in a VPC.
- Feat: `fptcloud_managed_kubernetes_engine_v1` reads clusters

### Resource

- Fix: `auto_upgrade_timezone` on `fptcloud_managed_kubernetes_engine_v1` is validated.
- Fix: the documentation examples used `Asia/Bangkok` and `Asia/Ho_Chi_Minh`, which the API rejects; they now use `Asia/Saigon`

## [0.3.84] - 2026-10-24

### Resource

- Breaking: `fptcloud_database` no longer destroys and creates the cluster when an argument changes. `vpc_id`, `network_id`, `vm_network`, `type_config`, `type_db`, `version`, `vdc_name`, `is_cluster`, `master_count`, `worker_count`, `node_core`, `cluster_name`, `database_name`, `vhost_name`, `is_public`, `edge_id`, `edition`, `is_ops`, `number_of_node`, `number_of_shard` and `domain_name` cannot be changed after creation: `terraform plan` now fails with the value currently on the portal. To create the cluster again, use `terraform apply -replace=fptcloud_database.<name>`
- Feat: `fptcloud_database` warns when `node_cpu`, `node_ram`, `data_disk_size` or `storage_profile` was changed outside of Terraform (applied value vs. value on the portal now)
- Fix: changing `admin_password`, `maintenance_email`, `day_of_week_maintenance` or `time_maintenance` of `fptcloud_database` now fails `terraform plan` with a clear message (change it on the portal; use `-replace` to create the cluster again). Before, the apply ended with `Provider produced inconsistent result after apply`
- Fix: `fptcloud_database` keeps `domain_name` from the state instead of reading it back as an empty string, and keeps `edition` and `vdc_name` when the portal returns the same text in another letter case
- Fix: `fptcloud_database` saves the state as soon as the cluster is created, before the tags are applied, so a failure while applying tags no longer loses track of the cluster
- Fix: after a resize or a storage policy change, `fptcloud_database` fails with a clear error when the cluster went through a busy status (for example `resizing`) and came back `running`, `stopped` or `warning` twice in a row without the new values, instead of waiting 30 minutes. `warning` with the new values now counts as done
- Known limitation: `fptcloud_database_status` still replaces the resource when `status` or `vpc_id` changes
- Feat: backup and restore for databases. New resources `fptcloud_database_backup` (backup service, capacity and the `full_job`/`diff_job` jobs, importable by cluster id), `fptcloud_database_backup_run_now` (runs a job once; run again by changing `triggers`) and `fptcloud_database_restore` (restores a restore point into a new cluster; the restored cluster is kept on destroy unless `delete_cluster_on_destroy = true`). New data sources `fptcloud_database_backup`, `fptcloud_database_restore_points` and `fptcloud_database_restored_clusters`
- Feat: none of the new resources uses ForceNew. An argument that cannot be changed (`cluster_id`, `vpc_id`, a job `name`, a smaller `capacity_gb`) fails `terraform plan` with the value currently on the portal; changing an argument of `fptcloud_database_backup_run_now` or `fptcloud_database_restore` only updates the state and warns, use `terraform apply -replace=...` to run again
- Feat: `fptcloud_database_backup` warns when `capacity_gb` or a job schedule was changed outside of Terraform, and when a job was deleted on the portal (the next apply creates it again)
- Known limitation: `full_job` and `diff_job` are blocks (`full_job { ... }`), not attributes, because the provider is served over protocol 5; use `full_job[0].job_id` in expressions
- Known limitation: point-in-time recovery, MongoDB Enterprise and MongoDB Sharding are not supported. Enabling backup needs an alert with `alert_service = "backup"` and a recipient on the cluster, otherwise the API answers `error_code 160009`

## [0.3.83] - 2026-10-21

### Resource

- Fix: changing `tag_ids` on `fptcloud_snapshot` updates the tags in place instead of destroying and recreating the snapshot. Changing `vpc_id`, `instance_id`, `name` or `include_ram` still replaces it
- Feat: `tag_ids` on `fptcloud_snapshot` is read back from the API, so tags changed outside Terraform show up as drift, and `tag_ids = []` removes every tag
- Feat: `tag_ids` on `fptcloud_security_group` and `fptcloud_subnet` is sent on create and applied in place on update
- Fix: `fptcloud_subnet` keeps waiting while the portal is still creating the subnet, instead of failing with `UnknownError: System error, please try again !` when the create task takes longer than the first check (10 seconds). The subnet was created anyway and was left out of the state, so the next apply failed on the CIDR. Other errors still stop the wait, with the API message
- Feat: `color` on `fptcloud_tagging` accepts the full portal palette of twelve hex colors: `#00bfff`, `#35994b`, `#e7a600`, `#db4646`, `#4b5563`, `#c81e1e`, `#d03801`, `#dc6803`, `#057a55`, `#047481`, `#5850ec`, `#6c2bd9`. Leaving `color` unset or empty still creates a tag without a color
- Fix: `color` on `fptcloud_tagging` is validated at plan time, so a color name or a hex outside the palette fails before apply instead of with `Data does not match the required format`, and an uppercase hex no longer shows a diff on every plan
- Known limitation: the portal API must be updated for these to work end to end. Until then, `tag_ids` on `fptcloud_security_group`, `fptcloud_subnet` and `fptcloud_storage` reads back empty, so a plan keeps showing a diff on `tag_ids`, and the API accepts only `#e7a600`, `#db4646` and `#35994b` as a tag `color`
- Known limitation: as on `fptcloud_storage`, `tag_ids = []` on `fptcloud_security_group` and `fptcloud_subnet` leaves the existing tags in place

## [0.3.82] - 2026-10-20

### Docs

- Add docs and examples for Object Storage IAM user and IAM role

## [0.3.81] - 2026-10-12

### Resource

- Feat: import `fptcloud_instance` by id, `vpc/<vpc_id>/instance/<instance_id>`, or by name, `vpc/<vpc_id>/instance_name/<instance_name>`, with `terraform import` or an `import` block. The state keeps the instance id; every attribute is read from the API. `ssh_key`, `password` and an unknown `image_name` cannot be read: the first apply after the import records the configured values without calling the API
- Feat: import an `EXTERNAL` `fptcloud_storage` with `vpc/<vpc_id>/storage/<storage_id>`.
- Resources already in a state are read, planned and applied exactly as before
- Known limitation: instances with a physical NVMe disk (`is_nvme = true`) cannot be imported yet

## [0.3.80] - 2026-10-10
- Fix: resize instance add status UNRESOLVE

## [0.3.79] - 2026-10-06

### Datasource

- Feat: add `fptcloud_instances` to list all instances in a VPC
- Feat: add `fptcloud_security_groups` to list all security groups in a VPC
- Feat: add `fptcloud_storages` to list all storages in a VPC

## [0.3.78] - 2026-10-01

### Resource

- Fix: `fptcloud_storage` can change an EXTERNAL disk that is attached to an instance. `size_gb`, `name` and `storage_policy_id` are applied through the update path that the VPC supports, instead of one that is rejected while the disk is attached
- Fix: an update waits until the disk reports its new size, so state matches the disk instead of keeping the value that was requested. The wait uses the provider `timeout`
- Fix: a storage update no longer fails with `unexpected state 'DISABLED'` while the disk settles after a change that did succeed
- Fix: the error raised when attaching or detaching a storage carries the message returned by the API instead of only the storage id

## [0.3.77] - 2026-09-30

### Resource

- Feat: `fptcloud_database` resizes in place. Changing `node_cpu`, `node_ram`, `data_disk_size`, `flavor` or `flavor_id` now calls the cluster resize API instead of destroying and recreating the database, and changing `storage_profile` calls the change storage policy API. Apply waits until the cluster is `running` or `stopped` again with the new values, up to 30 minutes, and fails if the cluster ends up `failed`
- Fix: changing compute/disk and `storage_profile` in the same apply is rejected at plan time with `Unsupported combined change`; apply them one after the other
- Fix: a `storage_profile` that does not exist in the VPC is rejected at plan time, and the error lists the supported policy names
- Fix: `fptcloud_database` keeps `id` in the plan on update, so a data source or output that reads `id` no longer shows every value as `known after apply`
- Fix: `nodes` is read again after an in-place update, so `number_of_cpus`, `memory_mb` and `data_disk_size` in state match the resized cluster

## [0.3.76] - 2026-09-28

### Resource

- Feat: expose `scheme` (`internet_facing` or `internal`) on `fptcloud_load_balancer_v2_lb`, forwarded on create and read back into state ([#113](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/113))
- Update: `scheme` is derived from the subnet `network_id` already points to when left unset; `network_id` stays required on OSP, the same as before this field existed ([#113](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/113))
- Fix: reject changing `scheme` on an existing load balancer with a clear error at apply time, instead of silently ignoring the change and leaving the plan permanently out of sync ([#113](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/113))

### Datasource

- Feat: expose `scheme` on `fptcloud_load_balancer_v2_lb` and `fptcloud_load_balancer_v2_lbs` ([#113](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/113))

## [0.3.75] - 2026-09-24

### Datasource

- Feat: `fptcloud_backup_veeam_restore_groups` lists the instances that have restore points in a VPC, one entry per instance and backup job - the portal's Restore tab. Each entry carries the `vm_id` and `backup_job_id` that `fptcloud_backup_veeam_restore_points` needs, so the backup jobs no longer have to be read first

## [0.3.74] - 2026-09-24

### Resource

- Feat: take an instance snapshot with `fptcloud_snapshot`: create, read, delete and import. Every volume attached to the instance is captured at one point in time, crash-consistent - the guest filesystem is not quiesced. Optional `name`, `include_ram` and `tag_ids`; left unset, the platform names the snapshot

## [0.3.73] - 2026-09-19

### Resource

- Feat: tag a `fptcloud_managed_kubernetes_engine_v1` cluster with the new `tags` attribute, and a worker pool with `pool_tags`. Cluster tags propagate to every pool, so `pool_tags` only lists that pool's own extras

### Datasource

- Feat: `fptcloud_managed_kubernetes_engine_v1` reports the cluster's `tags` and each pool's `pool_tags`

## [0.3.72] - 2026-09-18

### Resource

- Feat: manage IAM users on Ceph-backed object storage with `fptcloud_object_storage_iam_user`: create, read and delete. Access keys and the inline policy are separate resources, so either can be rotated or rewritten without recreating the user
- Feat: `fptcloud_object_storage_iam_user_access_key` mints and revokes an IAM user's access keys - up to 2 per user - and returns the secret only once, at create
- Feat: `fptcloud_object_storage_iam_user_policy` manages the single inline policy attached to an IAM user; every resource it names must be a bucket the account owns
- Feat: `fptcloud_object_storage_iam_role` manages IAM roles and `trusted_users`, the IAM users allowed to assume the role
- Feat: `fptcloud_object_storage_iam_role_policy` manages the single inline policy attached to an IAM role - what the role may do once assumed, distinct from who may assume it

### Datasource

- Feat: `fptcloud_object_storage_iam_user` lists IAM users; `fptcloud_object_storage_iam_user_detail` reads one by name
- Feat: `fptcloud_object_storage_iam_user_access_key` lists an IAM user's access keys - never their secrets
- Feat: `fptcloud_object_storage_iam_user_policy` reads an IAM user's inline policy
- Feat: `fptcloud_object_storage_iam_role` lists IAM roles; `fptcloud_object_storage_iam_role_detail` reads one by name
- Feat: `fptcloud_object_storage_iam_role_policy` reads an IAM role's inline policy

## [0.3.71] - 2026-09-17

### Resource

- Fix: `fptcloud_storage` with `type = "EXTERNAL"` no longer times out on create. The provider now calls the console's async create endpoint, which answers with the storage id right away, and waits for the storage to become `ENABLED` (and, when `instance_id` is set, to be attached to that instance). Before, the API held the request until the whole Celery create finished. `LOCAL` storages still use the previous endpoint
- Fix: the async endpoint does not take `tag_ids`, so the provider applies them itself once an `EXTERNAL` storage is `ENABLED`
- Feat: if the create request of an `EXTERNAL` storage times out (client timeout, HTTP 502 or 504), the apply no longer fails. The provider looks the storage up by name until a new one appears, for up to `timeouts.create` (default `15m`), and fails only if none does

## [0.3.70] - 2026-09-17

### Resource

- Feat: restore an instance from a backup restore point with `fptcloud_backup_veeam_restore`, the same operation as "Restore Instance" in the portal, including quick rollback and power-on-after-restore. This overwrites the running instance
- Feat: restore into a **new** instance with `fptcloud_backup_veeam_restore_clone`, the same operation as "Restore keep" in the portal: the restore point is brought back under a new name and the original instance keeps running
- Feat: run a backup as a new instance straight from the backup with `fptcloud_backup_veeam_instant_recovery`, the same operation as "Instant Recovery" in the portal. Only the portal's "Restore to the new instance" mode is offered: the API has an in-place mode as well, but the portal hides the option that selects it. Note that an open session blocks the backup job of the instance it was mounted from, so the new data source below is worth a look when a job stops running
- All three are actions rather than pieces of infrastructure: every argument forces a new run, there is no in-place update and no import, and `terraform destroy` calls no API - it warns about what was left behind and stops tracking. A restore cannot be undone, the instance a restore keep created is not deleted, and an instant recovery session is left running (keeping the mounted instance or stopping the session is done in the portal)
- An apply returns as soon as the platform has accepted the request. It does not wait for the restore to finish, and it does not report the outcome either: there is no dependable signal for either. A restore point's status is set to `PENDING` when the request is accepted and never cleared, so waiting for a `SUCCESS` would wait forever; and the one place the outcome is recorded - the portal's History tab - is a list with no id per entry, so an entry could only be matched to an apply by guessing. Progress and outcome are read in the portal

### Datasource

- Feat: `fptcloud_backup_veeam_restore_points` lists the restore points of one protected instance, newest first, with the date, size and type the portal's restore dialog shows
- Feat: `fptcloud_backup_veeam_instant_recovery_sessions` lists the Instant Recovery sessions open in a VPC, including ones started from the portal - the place to look when a backup job stops running

## [0.3.69] - 2026-09-11

### Resource

- Feat: manage Backup Veeam jobs with `fptcloud_backup_veeam_job`: create, update, list and delete, with several instances per job and daily, monthly or hourly schedules. Adding or removing an instance updates the job in place and keeps its restore points

### Datasource

- Feat: `fptcloud_backup_veeam_job` reads one backup job in full - its schedule, retention, instances and notification methods - by id or by name. The plural data source returns none of those
- Feat: `fptcloud_backup_veeam_instances` lists the instances that can still be assigned to a backup job
- Feat: `fptcloud_backup_veeam_jobs` lists the backup jobs in a VPC
- Feat: `fptcloud_alert_notification_methods` lists notification channels, used to fill `notification_method_ids` on a backup job

## [0.3.68] - 2026-09-11

### Resource

- Feat: new `fptcloud_managed_gpu_cluster` resource for Managed GPU (bare metal) Kubernetes clusters on OSP

### Datasource

- Feat: new `fptcloud_hpc_subnet` datasource listing the HPC bare-metal subnet catalog for a VPC, with the same filter shape as `fptcloud_subnet`. This is a separate catalog: the two do not share ids
- Feat: new `fptcloud_managed_gpu_cluster` datasource returning a cluster's pools, networking and GPU configuration, including the operators and per-pool GPU settings that only the GPU-software backend reports

## [0.3.67] - 2026-09-10

### Resource

- Feat: support GPU instances on `fptcloud_instance` — new `gpu_plan` (`hold`/`detach` billing plan), `gpu_name` (optional verification input, also reflects the actual GPU attached), `vm_type` (`cpu`/`gpu`) and `is_nvme` attributes ([#107](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/107))
- Feat: changing `flavor_name` to/from a GPU flavor attaches/detaches the GPU as part of the in-place resize, the instance is not replaced; `gpu_plan` is sent along with that same resize request instead of racing a separate follow-up call ([#107](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/107))
- Fix: `gpu_name` no longer carries its previous Computed value into a resize away from a GPU flavor, which made the server reject the request with "gpu_name is only applicable to GPU flavors" ([#107](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/107))
- Fix: refresh state after update even when a later step in the same apply fails, so changes that did succeed (e.g. a flavor resize before a billing plan failure) aren't left stale ([#107](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/107))

### Datasource

- Feat: expose `gpu_id`, `gpu_name` and `is_nvme` on `fptcloud_flavor`; GPU flavors not supported in the VPC's default zone are no longer returned ([#107](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/107))

## [0.3.66] - 2026-09-07

### Resource

- Feat: manage tags on `fptcloud_load_balancer_v2_lb` through the new `tag_ids` attribute ([#109](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/109))
- Fix: mark `network_id`, `egw_id`, `vip_address`, `cidr` and `floating_ip` as computed on `fptcloud_load_balancer_v2_lb`, so a value assigned by the platform no longer shows a diff on every plan ([#109](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/109))
- Fix: save the values assigned during create of `fptcloud_load_balancer_v2_lb` to state immediately, instead of leaving them unset until the next refresh ([#109](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/109))
- Update: document the per-platform behaviour of `network_id`, `cidr` and `egw_id` on `fptcloud_load_balancer_v2_lb`, and that `egw_id` takes the edge gateway's platform ID from the infrastructure, not the record ID from Portal ([#109](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/109))

### Datasource

- Feat: expose the tags from FPT Cloud's tagging service through the new `resource_tags` attribute on `fptcloud_load_balancer_v2_lb` and `fptcloud_load_balancer_v2_lbs` ([#109](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/109))
- Update: clarify that `tags` on `fptcloud_load_balancer_v2_lb` and `fptcloud_load_balancer_v2_lbs` is an internal marker identifying the object as LBv2 for Portal sync, not FPT Cloud's tagging service ([#109](https://github.com/fpt-corp/terraform-provider-fptcloud/pull/109))

## [0.3.65] - 2026-08-27

### Resource

- Feat: resize the root disk of `fptcloud_instance` in place, `storage_size_gb` and `storage_policy_id` no longer replace the instance

## [0.3.64] - 2026-08-27

### Resource

- Fix: update type Labels and Taints for worker pool in MFKE

## [0.3.63] - 2026-08-25

### Resource

- Fix: keep `nodes` known when `fptcloud_database` is updated in place, so a `tag_ids` change no longer fails with "Provider returned invalid result object after apply" and the new state is saved

## [0.3.62] - 2026-08-21

### Resource

- Fix: serialize `policy` and `vms` on `fptcloud_instance_group` read so state matches the string schema instead of failing on the object/list returned by the API

## [0.3.61] - 2026-08-19

### Resource

- Feat: support `Status` in bucket lifecycle rule so a rule can be created Disabled
- Fix: send only the fields a lifecycle rule declares, instead of omitted objects as zero values the API rejects
- Fix: do not record a bucket lifecycle rule in state when its create fails
- Fix: detect lifecycle and CORS rules deleted outside Terraform instead of reporting them as present
- Fix: treat a missing bucket as drift instead of failing the plan
- Fix: report a failed bucket static website create as an error instead of success
- Fix: adopt object storage objects left behind by a create whose response was lost

## [0.3.60] - 2026-08-13

### Datasource

- Feat: Add datasource database

## [0.3.59] - 2026-08-12

### Resource

- Fix: suppress JSON diff for bucket policy, CORS and lifecycle rule to prevent unnecessary resource recreation

## [0.3.58] - 2026-08-11

### Resource

- Fix: mark private_ip and public_ip as Computed in resource schema to prevent refresh state drift

## [0.3.57] - 2026-08-11

### Resource

- Fix: populate private_ip in state to prevent downstream plan drift

## [0.3.56] - 2026-08-11

### Resource

- Fix: `resource/fptcloud_instance`: populate `private_ip` attribute in state after create/read.

## [0.3.55] - 2026-07-31

### Resource

- Feat: Add SGN2 support for the database resource.

## [0.3.54] - 2026-07-30

### Resource

- Feat: MFKE: Support MFKE version >= 1.33.12

## [0.3.53] - 2026-07-14

### Resource

- Feat: MFKE: Add config internalNetworkLB key to MFKE cluster

## [0.3.52] - 2026-06-20

### Resource

- Feat: Add data_node_type key to database create payload

## [0.3.51] - 2026-06-19

### Resource

- Feat: support MFKE in region SGN2 

## [0.3.50] - 2026-06-02

### Resource

- Feat: add new datasource mfke_kubeconfig for M-FKE

## [0.3.49] - 2026-06-02

### Resource

- Unknown

## [0.3.48] - 2026-06-02

### Resource

- Unknown
## [0.3.47] - 2026-02-04

### Resource

- Feat: add key required in create database

## [0.3.46] - 2026-02-02

### Resource

- Feat: Update creating/updating taints on worker base MFKE

## [0.3.45] - 2026-01-14

### Resource

- Fix: Fix creating/updating TCP listener without insert headers options

## [0.3.44] - 2026-01-12

### Resource

- Update: Update docs for Managed Kubernetes with GPU

## [0.3.43] - 2025-12-31

### Resource

- Feat: Add `nodes` computed attribute to `fptcloud_database` resource to expose VM nodes information

### Datasource

- Feat: Add `fptcloud_edge_gateways` datasource to retrieve a list of edge gateways with optional name filter

## [0.3.42] - 2025-12-22

- Feature: Managing denied CIDRs of LBaaS listener resource

## [0.3.41] - 2025-12-18

### Resource

- Feat: Add datasource flavor database and apply tagging for database

## [0.3.40] - 2025-12-17

### Resource

- Support configuring primary and secondary DNS when creating a subnet

## [0.3.37] - 2025-12-12

### Resource

- Feat: Add `network_id` attribute to subnet resource

## [0.3.32] - 2025-12-01

### Resource

- Fix: Improve error handling in storage resource

## [0.3.31] - 2025-11-09

- Feat: Update case error create and pending

## [0.3.30] - 2025-11-03

- Feat: Add Managed FKE storage policy

## [0.3.29] - 2025-11-03

- Refactor: Refactor KV and taints type of M-FKE.
- Update: Update resource M-FKE
- Feat: Add check service account M-FKE

## [0.3.26] - 2025-10-22

- Fix: Bucket configuration, sub user resource.

## [0.3.25] - 2025-10-21

- Update: Create bucket with object lock

## [0.3.24] - 2025-10-14

- Feature: Add support for managing LBaaS resources — load balancer, listener, pool, L7 policy, and L7 rule.

## [0.3.23] - 2025-09-26

- Update the status when attaching an instance

## [0.3.21] - 2025-09-23

- Fix bug: Fix change worker pool VMW M-FKE cluster

## [0.3.20] - 2025-09-08

- Fix: fix bug network type when create M-FKE cluster

## [0.3.19] - 2025-09-08

- Fix: fix bug NetworkName of M-FKE cluster

## [0.3.17] - 2025-09-05

- Fix: Fix create and update Worker for VMW operation Managed Kubernetes Engine

## [0.3.16] - 2025-08-28

- Update docs for resource Managed Kubernetes Engine

## [0.3.15] - 2025-08-28

- Update docs for resource Managed Kubernetes Engine

## [0.3.14] - 2025-08-28

- Refactor CRUD for Managed Kubernetes Engine
- Update docs for Managed Kubernetes Engine
- Create VGPU datasource (fptcloud_vgpu)
- Create docs for VGPU datasource

## [0.3.13] - 2025-06-16

### Resource

- Fix database creation bug tainting data

## [0.3.12] - 2025-06-12

### Resource

- Fix database status creation

## [0.3.11] - 2025-06-09

### Docs

- Update docs and example for Object Storage

## [0.3.9] - 2025-04-23

### Resource

- Improved error handling for database

## [0.3.10] - 2025-06-09

### Resource

- Update database & database status

## [0.3.8] - 2025-04-23

### Datasource

- Update edge gateway

## [0.3.7] - 2025-04-22

### Datasource

- Edge gateway

### Resource

- Mfke
- Mfke powerstate
- Database
- Database status

## [0.3.6] - 2025-04-09

### Datasource

- Fix bug get instance

## [0.3.5] - 2025-04-09

### Docs

- Update docs and example

## [0.3.4] - 2025-03-17

### Provider

- Fix bug timeout for API

## [0.3.3] - 2025-03-17

### Provider

- Support timeout for fke

## [0.3.2] - 2025-03-17

### Provider

- Support timeout

## [0.3.0] - 2025-01-02

### Resource

- fptcloud_object_storage_bucket_acl
- fptcloud_object_storage_bucket_cors
- fptcloud_object_storage_bucket_lifecycle
- fptcloud_object_storage_bucket_policy
- fptcloud_object_storage_bucket_static_website
- fptcloud_object_storage_bucket_versioning
- fptcloud_object_storage_sub_user
- fptcloud_object_storage_user_key
- fptcloud_object_storage_access_key

### Datasource

- fptcloud_object_storage_access_key
- fptcloud_object_storage_bucket_acl
- fptcloud_object_storage_bucket_cors
- fptcloud_object_storage_bucket_lifecycle
- fptcloud_object_storage_bucket_policy
- fptcloud_object_storage_bucket_static_website
- fptcloud_object_storage_bucket_versioning
- fptcloud_object_storage_bucket
- fptcloud_s3_service_enable
- fptcloud_object_storage_sub_user

## [0.2.1] - 2024-10-17

### Resource

- fix set value read storage

## [0.2.0] - 2024-10-09

### Datasource

- Dedicated kubernetes engine
- Managed kubernetes engine
- Edge gateway

### Resource

- Dedicated kubernetes engine
- Managed kubernetes engine
- Database
- Database status

## [0.1.2] - 2024-09-15

### Resource

- Update schema instance (using flavor name and image name)
- Fix bug security group rule

## [0.1.1] - 2024-08-10

### Data source

- Subnet

### Resource

- Floating IP association
- Subnet

## [0.1.0] - 2024-08-09

### Data source

- Floating IP
- Floating IP rule
- Instance group
- Instance
- Security group
- Security group rule
- Storage
- Flavor
- Image
- SSH key
- Storage Policy
- VPC

### Resource

- Floating IP
- Instance
- Instance group
- Security group
- Security group rule
- SSH key
- Storage
- VPC
 