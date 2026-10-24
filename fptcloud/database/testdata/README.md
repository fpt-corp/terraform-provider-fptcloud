# Fixtures of the database backup/restore API

Real responses from the dev environment (region `saigon-02-vn`, 2026-10-23), with org name, vCD URL and
e-mail addresses replaced. Base path is `/v1/xplat/database/`. Every mutation answers HTTP 200.

## Findings (P0)

1. `backup/service/job/create` returns only `{message, type}`, **no `job_id`**: find the job by name in `job/list`.
   A job goes `creating` -> `ready` in about 25 seconds, an update `updating` -> `ready` in about 40 seconds.
2. `restore/create` returns `data: []`, **no cluster id**. The new cluster shows up in `list-restore-cluster`
   at once, `cluster_name` = `restore_name`, status `processing`, `running` after about 4 minutes.
   Deleted clusters stay in the list with status `deleted`, ignore them when checking that a name is free.
   `flavor` must be the flavor name (`4C8G`); an empty `flavor` made the restore end `failed` after 47 seconds.
3. `backup_enabled`: `0` inactive, `2` processing (about 3 minutes after enabling), `1` enabled. `4` updating lasts
   about 3 minutes for a capacity change, and `capacity` only changes when it ends. `3` deleting about 100 seconds.
   `5` was seen only on a cluster with PITR; after a capacity change it went back to `1`, so treat `1` and `5` as ready.
   `job/list` reports `backup_enabled: 1` for a cluster that `backup/detail` reports as `5`: read the state from `backup/detail`.
4. `backup/detail` has `backup_service_id` and `is_archived`. `percent_backup_used` is a percentage of `capacity`.
5. Sending the old `is_archived` (1) again in `POST capacity` keeps PITR (`engine_active_pitr` stays 1).
6. `job/update` ignores a new `backup_job_name` (the schedule and retention are applied): the name cannot be changed.
7. Business errors are HTTP 200 with `{error_code, message, type}`; a success has only `{message, type: "success"}`.
   Seen: `160009` (type `warning`) enabling backup without an alert with `alert_service = "backup"` and a recipient
   (`monitoring/alert/create` adds it); `160006` (type `error`) capacity not larger than the current one.
8. The two restore endpoints are as in the plan: `list-restore-point` lists restore points, `list-restore-cluster`
   lists restored clusters (the portal function names are swapped).

The server does **not** validate: a duplicate job name, a one-character name, deleting the full job while a diff job
exists, or a second `runnow` while one runs were all accepted. The provider must check these itself.

`runnow` sets `is_run_now = 1` (job stays `ready`) and the action log shows `Run full backup` as `pending`; a run
of an empty database took 46 seconds. Retention of a diff job follows the full job. The action log
(`list-backup-request`) task `status` values seen: `successful`, `pending`.
