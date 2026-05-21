# Isolation Checks

Use these commands after the tenants are deployed.

## Cross-namespace Postgres should fail

```bash
kubectl -n tenant-a run netcheck --rm -it --image=postgres:16-alpine --restart=Never -- \
  psql "postgres://feedback:REDACTED@feedback-postgres.tenant-b.svc.cluster.local:5432/feedback?sslmode=disable" -c "select 1"
```

Expected result: connection timeout or network policy denial.

## Same-namespace Postgres should work

```bash
kubectl -n tenant-a run netcheck --rm -it --image=postgres:16-alpine --restart=Never -- \
  psql "postgres://feedback:REDACTED@feedback-postgres.tenant-a.svc.cluster.local:5432/feedback?sslmode=disable" -c "select 1"
```

Expected result: query succeeds.

## Falco shell alert

```bash
kubectl -n tenant-a exec -it deploy/feedback-api -- sh
kubectl -n falco logs ds/falco --tail=50
```

Expected result: Falco logs a `Shell Opened In Tenant Pod` warning.

