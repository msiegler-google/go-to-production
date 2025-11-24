# Phase 4: Secure Configuration Implementation Guide

This guide documents the steps to secure the todo-app-go configuration by migrating credentials to Google Secret Manager and implementing Workload Identity.

## Overview

Phase 4 implements the following security improvements:
- **Google Secret Manager**: Stores sensitive database credentials securely.
- **Workload Identity**: Allows Kubernetes pods to authenticate with Google Cloud APIs using a Kubernetes Service Account (KSA) mapped to a Google Service Account (GSA), eliminating the need for long-lived service account keys.
- **Application Logic**: The application now fetches secrets directly from Secret Manager at runtime.

## Prerequisites

- Terraform installed and configured
- kubectl installed
- gcloud CLI authenticated with appropriate permissions
- Existing GKE cluster and Cloud SQL instance (from Phase 3)
- **Permissions**: The user running Terraform must have `iam.serviceAccounts.setIamPolicy` permission (e.g., Service Account Admin role).

## Step 1: Update Terraform Configuration

### 1.1 Enable APIs

Enable `secretmanager.googleapis.com` and `iamcredentials.googleapis.com` in `main.tf`.

### 1.2 Configure Workload Identity on Cluster

Enable Workload Identity on the GKE cluster in `main.tf`:

```hcl
workload_identity_config {
  workload_pool = "${var.project_id}.svc.id.goog"
}
```

### 1.3 Create Secrets

Define the database password secret in `secrets.tf`:

```hcl
resource "google_secret_manager_secret" "db_password" {
  secret_id = "db-password"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "db_password_version" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = var.db_password
}
```

### 1.4 Configure IAM

In `iam.tf`:
1. Create a Google Service Account (GSA) for the application.
2. Grant `roles/secretmanager.secretAccessor` to the GSA.
3. Bind the GSA to the Kubernetes Service Account (KSA) using `roles/iam.workloadIdentityUser`.

```hcl
resource "google_service_account_iam_member" "workload_identity_binding" {
  service_account_id = google_service_account.todo_app_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[default/todo-app-sa]"
}
```

### 1.5 Apply Infrastructure Changes

```bash
cd terraform
terraform apply -auto-approve
```

## Step 2: Update Kubernetes Manifests

### 2.1 Create ServiceAccount

Create `k8s/serviceaccount.yaml` with the Workload Identity annotation:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: todo-app-sa
  annotations:
    iam.gke.io/gcp-service-account: todo-app-sa@${PROJECT_ID}.iam.gserviceaccount.com
```

### 2.2 Update Deployment

Update `k8s/deployment.yaml` to use the ServiceAccount and remove secret-based env vars:

```yaml
spec:
  serviceAccountName: todo-app-sa
  containers:
  - name: todo-app-go
    # ...
    env:
    # Removed DB_PASSWORD
    - name: GOOGLE_CLOUD_PROJECT
      value: "${PROJECT_ID}"
```

## Step 3: Update Application Code

Update `main.go` to fetch the password from Secret Manager using the Google Cloud client library.

```go
import (
    secretmanager "cloud.google.com/go/secretmanager/apiv1"
    "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// ... inside initDB ...
client, err := secretmanager.NewClient(ctx)
// ...
req := &secretmanagerpb.AccessSecretVersionRequest{
    Name: fmt.Sprintf("projects/%s/secrets/db-password/versions/latest", projectID),
}
result, err := client.AccessSecretVersion(ctx, req)
password := string(result.Payload.Data)
```

## Step 4: Deploy

### 4.1 Apply Manifests

```bash
kubectl apply -f k8s/serviceaccount.yaml
kubectl apply -f k8s/deployment.yaml
```

### 4.2 Verify

Check logs to ensure the application connects to the database successfully:

```bash
kubectl logs -l app=todo-app-go -c todo-app-go
```

## Troubleshooting

### IAM Permission Denied

If Terraform fails with `IAM_PERMISSION_DENIED` on `google_service_account_iam_member`, ensure your user has the **Service Account Admin** role.

### Application Fails to Start

If the app fails to fetch the secret:
1. Verify the KSA is annotated correctly: `kubectl describe sa todo-app-sa`
2. Verify the GSA has `roles/secretmanager.secretAccessor`.
3. Verify the Workload Identity binding exists.
