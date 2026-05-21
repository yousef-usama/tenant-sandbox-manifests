# Tenant Sandbox Manifests

GitOps repo for the Tenant Sandbox EKS portfolio/demo project.

This repo owns:

- ArgoCD app-of-apps
- platform add-ons after bootstrap
- the Go feedback API source code
- Dockerfile and Helm chart
- tenant-specific values
- monitoring dashboard
- Falco custom rule

The infra repo owns AWS infrastructure and bootstraps only AWS Load Balancer Controller plus ArgoCD.

## Tenant Model

The demo uses three tenants:

- `tenant-a`
- `tenant-b`
- `tenant-c`

Each tenant gets:

- one Kubernetes namespace
- one feedback API Deployment
- one Postgres StatefulSet
- one tenant ingress hostname
- one tenant ServiceAccount
- namespace-scoped NetworkPolicies
- tenant-labeled Prometheus metrics

This is a portfolio/demo isolation model, not a production SaaS tenancy model.

## Local Development

Run the API locally with Docker Compose or a local Postgres instance. The Kubernetes deployment is driven by Helm values under `apps/feedback-api/chart/values`.

```bash
cd apps/feedback-api
go test ./...
go run ./cmd/server
```

## GitOps Bootstrap

After ArgoCD is installed by Terraform:

```bash
kubectl apply -f argocd/root-app.yaml
```

ArgoCD then reconciles the child applications in `argocd/apps`.

## Demo Checks

Isolation checks to capture in the final README:

- Tenant A API cannot connect to Tenant B Postgres.
- Tenant A ServiceAccount cannot read Tenant B export bucket.
- Cross-namespace pod traffic is denied by default.
- Prometheus shows metrics separated by the `tenant` label.
- Falco emits an alert when a shell is opened inside a tenant pod.

## Cost Notes

This repo intentionally uses in-cluster Postgres per tenant to avoid RDS cost during a one-shot portfolio demo. Data persistence uses PVCs, but the database setup is deliberately lightweight.

