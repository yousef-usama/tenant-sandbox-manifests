#!/usr/bin/env bash
set -euo pipefail

NEW_ACCOUNT_ID="${1:-}"

if [[ -z "$NEW_ACCOUNT_ID" || ! "$NEW_ACCOUNT_ID" =~ ^[0-9]{12}$ ]]; then
  echo "Usage: $0 <12-digit-aws-account-id>"
  echo "Example: $0 123456789012"
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_FILES=(
  "$REPO_ROOT/argocd/apps/external-secrets.yaml"
  "$REPO_ROOT/argocd/apps/external-dns.yaml"
  "$REPO_ROOT/apps/feedback-api/chart/values.yaml"
  "$REPO_ROOT/apps/feedback-api/chart/values/tenant-a.yaml"
  "$REPO_ROOT/apps/feedback-api/chart/values/tenant-b.yaml"
  "$REPO_ROOT/apps/feedback-api/chart/values/tenant-c.yaml"
)

echo "Updating AWS Account ID to $NEW_ACCOUNT_ID across manifests..."

for file in "${TARGET_FILES[@]}"; do
  if [[ -f "$file" ]]; then
    # Replace any 12-digit account ID in role ARNs and ECR repository URLs
    sed -i.bak -E "s/[0-9]{12}(\.dkr\.ecr|:role\/)/${NEW_ACCOUNT_ID}\1/g" "$file"
    rm -f "${file}.bak"
    echo "  Updated $file"
  else
    echo "  Warning: $file not found"
  fi
done

echo "Successfully updated AWS Account ID to $NEW_ACCOUNT_ID."
