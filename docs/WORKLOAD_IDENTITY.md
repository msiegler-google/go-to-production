# Workload Identity Setup Guide

This guide explains how to set up Workload Identity for the todo-app-go application.

## Prerequisites
- GKE cluster with Workload Identity enabled
- Google Cloud project with necessary APIs enabled

## Configuration Steps

### 1. Enable Workload Identity on Node Pool

In `terraform/main.tf`, ensure the node pool has the workload metadata configuration:

```hcl
resource "google_container_node_pool" "primary_nodes" {
  # ... other configuration ...
  
  node_config {
    # ... other settings ...
    
    workload_metadata_config {
      mode = "GKE_METADATA"  # Required for Workload Identity
    }
  }
}
```

### 2. Create Google Service Account

```hcl
resource "google_service_account" "todo_app_sa" {
  account_id   = "todo-app-sa"
  display_name = "Todo App Service Account"
}
```

### 3. Grant IAM Roles

```hcl
# Cloud SQL Client - connect to instances
resource "google_project_iam_member" "todo_app_cloudsql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.todo_app_sa.email}"
}

# Cloud SQL Instance User - IAM authentication
resource "google_project_iam_member" "todo_app_cloudsql_instance_user" {
  project = var.project_id
  role    = "roles/cloudsql.instanceUser"
  member  = "serviceAccount:${google_service_account.todo_app_sa.email}"
}
```

### 4. Create Workload Identity Binding

```hcl
resource "google_service_account_iam_member" "workload_identity_binding" {
  service_account_id = google_service_account.todo_app_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[default/todo-app-sa]"
}
```

### 5. Create Kubernetes Service Account

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: todo-app-sa
  namespace: default
  annotations:
    iam.gke.io/gcp-service-account: todo-app-sa@PROJECT_ID.iam.gserviceaccount.com
```

### 6. Use Service Account in Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: todo-app-go
spec:
  template:
    spec:
      serviceAccountName: todo-app-sa  # Reference K8s SA
      containers:
      - name: todo-app-go
        # ... container config ...
```

## Verification

### Verify Workload Identity Binding
```bash
gcloud iam service-accounts get-iam-policy \
  todo-app-sa@PROJECT_ID.iam.gserviceaccount.com
```

Expected output should include:
```yaml
bindings:
- members:
  - serviceAccount:PROJECT_ID.svc.id.goog[default/todo-app-sa]
  role: roles/iam.workloadIdentityUser
```

### Verify from Pod
```bash
# Get pod name
kubectl get pods -l app=todo-app-go

# Check service account email
kubectl exec POD_NAME -- curl -H "Metadata-Flavor: Google" \
  http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/email
```

Should return: `todo-app-sa@PROJECT_ID.iam.gserviceaccount.com`

### Verify Token Generation
```bash
kubectl exec POD_NAME -- curl -H "Metadata-Flavor: Google" \
  http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token
```

Should return a valid access token.

## Common Issues

### "Permission denied" errors
- Verify IAM roles are granted to the Google Service Account
- Check Workload Identity binding exists
- Ensure K8s SA annotation is correct

### Metadata server not accessible
- Verify node pool has `GKE_METADATA` mode enabled
- Check cluster has Workload Identity enabled
- Restart pods after configuration changes

### Wrong service account used
- Verify pod spec has correct `serviceAccountName`
- Check K8s SA annotation matches Google SA email
- Ensure namespace matches in Workload Identity binding

## Best Practices

1. **Use dedicated service accounts** - Create separate SAs for each application
2. **Principle of least privilege** - Grant only necessary IAM roles
3. **Namespace isolation** - Use different namespaces for different environments
4. **Audit bindings** - Regularly review Workload Identity bindings
5. **Test in staging** - Verify Workload Identity works before production deployment

## References
- [Workload Identity Documentation](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
- [Best Practices](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#best_practices)
