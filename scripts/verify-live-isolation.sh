#!/usr/bin/env bash
set -eo pipefail

echo "================================================================="
echo " Multi-Tenant Isolation Live Verification Script"
echo " Date: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo "================================================================="

echo ""
echo "[Test 1] Network Isolation: Pod in 'tenant-a' attempting cross-namespace"
echo "         connection to feedback-api in 'tenant-b' (TCP:8080)..."
echo "Expected: Connection timed out (NetworkPolicy drops cross-namespace packets)"
echo "-----------------------------------------------------------------"

set +e
NET_OUTPUT=$(kubectl run probe-net-test --namespace=tenant-a --rm -i --restart=Never \
  --image=curlimages/curl \
  --overrides='{
    "spec": {
      "securityContext": {
        "runAsNonRoot": true,
        "runAsUser": 10001,
        "seccompProfile": { "type": "RuntimeDefault" }
      },
      "containers": [
        {
          "name": "probe-net-test",
          "image": "curlimages/curl",
          "command": ["curl", "-m", "5", "-v", "http://feedback-api.tenant-b.svc.cluster.local:8080/healthz"],
          "securityContext": {
            "allowPrivilegeEscalation": false,
            "capabilities": { "drop": ["ALL"] }
          }
        }
      ]
    }
  }' 2>&1)
NET_CODE=$?
set -e

echo "$NET_OUTPUT"
if echo "$NET_OUTPUT" | grep -q "Connection timed out"; then
  echo ""
  echo ">>> [PASS] NetworkPolicy strictly BLOCKED cross-namespace traffic! (Code $NET_CODE)"
else
  echo ""
  echo ">>> [FAIL] Traffic was not blocked as expected."
fi

echo ""
echo "================================================================="
echo "[Test 2] Storage Isolation: Pod in 'tenant-a' attempting cross-tenant"
echo "         access to S3 bucket 'tenant-sandbox-tenant-b-exports'..."
echo "Expected: AccessDenied (IAM IRSA restricts access to tenant-a bucket only)"
echo "-----------------------------------------------------------------"

set +e
S3_OUTPUT=$(kubectl run probe-s3-test --namespace=tenant-a --rm -i --restart=Never \
  --labels="app.kubernetes.io/name=feedback-api" \
  --image=amazon/aws-cli:latest \
  --overrides='{
    "spec": {
      "serviceAccountName": "feedback-api",
      "securityContext": {
        "runAsNonRoot": true,
        "runAsUser": 1000,
        "seccompProfile": { "type": "RuntimeDefault" }
      },
      "volumes": [
        { "name": "tmp", "emptyDir": {} }
      ],
      "containers": [
        {
          "name": "probe-s3-test",
          "image": "amazon/aws-cli:latest",
          "command": ["aws", "s3", "ls", "s3://tenant-sandbox-tenant-b-exports/"],
          "env": [
            { "name": "HOME", "value": "/tmp" }
          ],
          "volumeMounts": [
            { "name": "tmp", "mountPath": "/tmp" }
          ],
          "securityContext": {
            "allowPrivilegeEscalation": false,
            "capabilities": { "drop": ["ALL"] }
          }
        }
      ]
    }
  }' 2>&1)
S3_CODE=$?
set -e

echo "$S3_OUTPUT"
if echo "$S3_OUTPUT" | grep -q "AccessDenied"; then
  echo ""
  echo ">>> [PASS] IAM IRSA strictly BLOCKED cross-tenant S3 access! (HTTP 403 AccessDenied)"
else
  echo ""
  echo ">>> [FAIL] Cross-tenant access was not denied as expected."
fi

echo ""
echo "================================================================="
echo "[Test 3] S3 Legitimate Access: Pod in 'tenant-a' accessing its OWN bucket"
echo "         'tenant-sandbox-tenant-a-exports'..."
echo "Expected: 200 OK / Objects listed"
echo "-----------------------------------------------------------------"

set +e
OWN_S3_OUTPUT=$(kubectl run probe-s3-own-test --namespace=tenant-a --rm -i --restart=Never \
  --labels="app.kubernetes.io/name=feedback-api" \
  --image=amazon/aws-cli:latest \
  --overrides='{
    "spec": {
      "serviceAccountName": "feedback-api",
      "securityContext": {
        "runAsNonRoot": true,
        "runAsUser": 1000,
        "seccompProfile": { "type": "RuntimeDefault" }
      },
      "volumes": [
        { "name": "tmp", "emptyDir": {} }
      ],
      "containers": [
        {
          "name": "probe-s3-own-test",
          "image": "amazon/aws-cli:latest",
          "command": ["aws", "s3", "ls", "s3://tenant-sandbox-tenant-a-exports/"],
          "env": [
            { "name": "HOME", "value": "/tmp" }
          ],
          "volumeMounts": [
            { "name": "tmp", "mountPath": "/tmp" }
          ],
          "securityContext": {
            "allowPrivilegeEscalation": false,
            "capabilities": { "drop": ["ALL"] }
          }
        }
      ]
    }
  }' 2>&1)
set -e

echo "$OWN_S3_OUTPUT"
if echo "$OWN_S3_OUTPUT" | grep -q "PRE exports/"; then
  echo ""
  echo ">>> [PASS] IAM IRSA successfully authorized access to tenant's OWN S3 bucket!"
else
  echo ""
  echo ">>> [NOTE] S3 access did not return expected prefix or was empty."
fi

echo ""
echo "================================================================="
echo " All Isolation Verifications Completed."
echo "================================================================="
