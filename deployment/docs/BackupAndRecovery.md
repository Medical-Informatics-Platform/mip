# Backup and Recovery

This guide covers the MIP 9.2.0 Kubernetes web and notebook stack and the
Exaflow data held by participating sites. Persistent data can reside on several
nodes or in managed storage. Back up the actual PVCs and site data, together
with the configuration needed to recover them.

## Backup inventory

The following names and paths come from the current chart templates. Inspect
custom deployments for additional volumes and external services.

| State | Current location | Backup method |
| --- | --- | --- |
| Platform PostgreSQL database `portal` | `platform-backend-db-claim0`, mounted at `/var/lib/postgresql` in the `platform-backend-db` container | Database dump plus roles; alternatively, a tested PostgreSQL physical backup procedure. |
| Backend application files | `platform-backend-claim0`, mounted at `/opt/platform/api` | Storage snapshot or offline archive. |
| Backend logs | `platform-backend-logs-claim0`, mounted at `/var/log/platform-backend` | Retain according to the installation's audit policy. |
| JupyterHub database and local state | `jupyterhub-claim0`, mounted at `/srv/jupyterhub` | Snapshot or archive with the Hub stopped, including its database and cookie secret files. |
| Users' notebooks and files | Separate user home PVCs mounted at `/home/jovyan`; workspaces are under `/home/jovyan/work` | Snapshot or archive each home after stopping its notebook server. |
| Exaflow input data | Each site's CSV/data-model folders on hostPath storage or CSV PVCs | Site-local backups of CSVs, `CDEsMetadata.json`, and other required dataset assets. |
| Configuration and credentials | Helm overrides, image versions, ConfigMaps, Kubernetes Secrets, and externally managed settings | Protected configuration backup and secret-manager recovery procedure. |

Include `mip-secret`, `keycloak-credentials`, `jupyterhub-crypt`, and TLS
credentials, plus any registry or operator-specific secrets used by the
installation. The Hub crypt key must match the backed-up Hub auth state.
Coordinate recovery of the external Keycloak realm and clients with its owner.
For operator-based notebook spawning, retain the operator/CRD versions and
NotebookProfile configuration as well.

Exaflow's current workers build DuckDB files on `emptyDir` volumes. These files
are rebuilt from source datasets, rather than treated as durable backups.
In-flight analyses and aggregation contexts are transient and must be rerun.

## Prepare a consistent backup

1. Record the timestamp, namespace, Helm release/chart revisions, image tags,
   storage classes, PVC-to-volume mapping, user-to-home mapping, and site data
   versions. Choose the retention period and acceptable data-loss window for
   the installation.
2. Schedule a maintenance window. Pause ingress or otherwise block new web and
   notebook requests, allow active analyses to finish, and stop all user
   notebook servers. Suspend GitOps/operator reconciliation where needed so
   stopped services are not recreated during the backup.
3. Stop the JupyterHub workload before copying its state. Quiesce backend
   application writes while keeping PostgreSQL running for a logical dump.
   The backend and PostgreSQL share one Deployment/pod in this chart: scaling
   `platform-backend` to zero stops both, so take the dump before scaling it
   down. Include other API clients when blocking writes.
4. Back up PostgreSQL, then snapshot/archive backend files, Hub state, and every
   notebook home while their writers are stopped. Coordinate site dataset
   backups with data managers and pause any dataset updates during capture.
5. Store configuration, secret recovery material, and the inventory with the
   backup set. Encrypt backups, restrict access, and keep a copy independent
   of the live storage. Clinical datasets stay within each site's governance
   boundary.
6. Verify backup command exit codes, file checksums, and snapshot availability;
   test restoration into an isolated environment. Resume services and normal
   reconciliation after capture completes.

A database dump is internally consistent, but independently captured filesystem
backups do not automatically form a consistent platform backup. The maintenance
window aligns these artifacts. Do not copy a running PostgreSQL data directory
as an ordinary filesystem archive; use a documented physical backup method or
stop PostgreSQL first. See the PostgreSQL documentation for
[SQL dumps](https://www.postgresql.org/docs/18/backup-dump.html) and
[filesystem backups](https://www.postgresql.org/docs/18/backup-file.html).

### PostgreSQL logical backup example

These commands use the current Kubernetes container, database, and administrator
configuration. Run them from a trusted workstation with namespace access.
Replace `<namespace>` and choose a new, protected backup directory. No credentials
are printed; the commands use the container's configured credentials.

```bash
umask 077
backup_dir=/secure/backups/mip-<timestamp>
mkdir -p "$backup_dir"

kubectl -n <namespace> exec deployment/platform-backend -c platform-backend-db -- \
  sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -h localhost -U "$POSTGRES_USER" -Fc portal' \
  > "$backup_dir/portal.dump"

kubectl -n <namespace> exec deployment/platform-backend -c platform-backend-db -- \
  sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" pg_dumpall -h localhost -U "$POSTGRES_USER" --globals-only' \
  > "$backup_dir/globals.sql"
```

Check each command succeeds before proceeding. `globals.sql` contains role
information and may contain password hashes; protect it with the other secrets.
Inventory any additional databases separately. A PostgreSQL dump does not
capture notebook files or Kubernetes configuration.

## Restore procedure

1. Select one verified backup set. Preserve the failed environment and perform
   the first restore into an isolated namespace or cluster with new volumes.
   Do not delete or overwrite existing data as part of the default procedure.
2. Restore the recorded configuration and secrets through the approved secret
   process. Use the original component versions first; treat a version upgrade
   as a separate operation after recovery has been verified. Provision storage
   with suitable capacity, access modes, and placement.
3. Restore PostgreSQL before allowing the backend application to start. Because
   this chart co-locates the two containers, use a temporary PostgreSQL-only
   recovery workload with the recorded database image and target database PVC.
   Only one PostgreSQL server may mount/use that data directory at a time.
4. Recreate the required database roles from the protected globals backup,
   reviewing statements that conflict with existing administrator roles. Keep
   application role credentials aligned with `mip-secret`. Create an empty
   `portal` database from `template0`, owned by the application role, and restore
   the custom-format dump with `pg_restore --exit-on-error`. Preserve object
   ownership and grants; do not restore over an initialized application schema.
5. Restore the backend application files and the entire Hub state volume while
   those workloads remain stopped. Restore every user home to the claim name
   expected for the same authenticated identity, preserving ownership and
   permissions (the notebook images use UID 1000/GID 100). Recover the original
   Hub crypt key. Do not rename identities or regenerate claims blindly.
6. At each data site, recover the source dataset folders/PVCs and metadata. Start
   Exaflow so workers rebuild their DuckDB databases, then verify that expected
   data models and datasets are available.
7. Stop the temporary PostgreSQL recovery workload before starting the regular
   stack with its restored PVC. Start the backend and Hub with external access
   still restricted. Confirm login, saved experiments and folders, and access
   permissions. Open a user's notebook and check saved files and
   `mip.Client.from_env()`. Run a representative analysis from the portal and
   a notebook.
8. After validation, reopen access and resume reconciliation. Record the actual
   recovery time, any lost work since the backup, and the validation outcome.
   Retain the pre-recovery volumes until the recovery is accepted.

In-flight jobs may be lost, and restored sessions may require login again.
Claims retained by ArgoCD annotations are still not backups: node loss, storage
failure, or accidental writes can affect their contents.

## Development Compose deployments

The development database uses the bind mount
`deployment/dev/.stored_data/platform_backenddb`; backend application files use
`deployment/dev/config`. Apply the same database-consistency rules when backing
these up. The development database service is `platform_backend_db` and listens
on port `5433`, unlike the Kubernetes example above.

The current Compose file has no persistent home mount for `mip_jupyter`. Copy
personal notebook files out before recreating that container. Both `start.sh`
and `stop.sh` run `docker compose down -v`; they are reset-oriented development
helpers, not production backup or recovery commands.
