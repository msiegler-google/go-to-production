# HOWTO: Manual Steps for `feature/gke-base-deployment`

This document outlines manual steps required for the `feature/gke-base-deployment` branch.

---

### **Prerequisite: GCP User Permissions**

Before running `terraform apply`, the Google Cloud user you are authenticated as must have sufficient permissions to create resources in the project. The simplest way to ensure this is to grant the **`Editor`** role to your user account.

**This is a one-time bootstrap step that must be done manually.**

You can do this via the GCP Console (IAM & Admin -> IAM) or by having a project owner run the following `gcloud` command:

```bash
# Replace your-gcp-project-id with your project ID
# Replace your-email@example.com with the email of the user running terraform
gcloud projects add-iam-policy-binding your-gcp-project-id \
  --member="user:your-email@example.com" \
  --role="roles/editor"
```

---

### **Prerequisite: Enable Default Compute Engine Service Account**

The GKE cluster creation may fail if the default Compute Engine service account is disabled in your project. This account is usually named `YOUR_PROJECT_NUMBER-compute@developer.gserviceaccount.com`. You can enable it with the following command:

```bash
# Replace YOUR_PROJECT_NUMBER with your actual project number
# Replace your-gcp-project-id with your actual project ID
gcloud iam service-accounts enable YOUR_PROJECT_NUMBER-compute@developer.gserviceaccount.com \
  --project your-gcp-project-id
```

---

### Step 1: Run `terraform apply`

Once your user has the correct permissions and the Compute Engine default service account is enabled, execute `terraform apply` in the `terraform/` directory.

**Important Note on Cloud SQL:** For this initial phase, the Cloud SQL instance is configured to use a **public IP address** and allows connections from `0.0.0.0/0` (all IP addresses). This is a **security risk** and is done solely to unblock the `terraform apply` step by avoiding complex VPC Private Service Connect configuration. **This configuration must be updated in a later phase for production use.**

```bash
cd terraform/
terraform init
terraform apply
```

After successful application, Terraform will output the email of the newly created service account (e.g., `github_actions_deployer_email`).

---

### Step 2: Create the Service Account Key

Using the service account email obtained from `terraform output`, generate a JSON key file for the service account.

1.  **Retrieve Service Account Email:**
    ```bash
    export SA_EMAIL=$(terraform output -raw github_actions_deployer_email)
    echo "Service Account Email: $SA_EMAIL"
    ```

2.  **Generate Key File:**
    ```bash
    gcloud iam service-accounts keys create "gcp-sa-key.json" \
      --iam-account="$SA_EMAIL"
    ```
    This command will create a file named `gcp-sa-key.json` in your current directory.

---

### Step 3: Configure GitHub Secrets and Variables

You need to configure the following in your GitHub repository's settings (`Settings > Secrets and variables > Actions`):

#### GitHub Secrets

*   **`GCP_SA_KEY`**:
    *   **Value:** The **entire content** of the `gcp-sa-key.json` file generated in Step 2.
    *   **Purpose:** Allows GitHub Actions to authenticate with Google Cloud to push Docker images and deploy to GKE.

#### GitHub Variables

*   **`GCP_PROJECT_ID`**:
    *   **Value:** Your Google Cloud Project ID (e.g., `my-gcp-project-12345`).
    *   **Purpose:** Used by CI/CD to identify the target project.
*   **`GCR_HOSTNAME`**:
    *   **Value:** The hostname for your Google Artifact Registry (e.g., `us-central1-docker.pkg.dev` or `gcr.io` if using legacy GCR).
    *   **Purpose:** Specifies where to push Docker images.
*   **`GKE_CLUSTER_NAME`**:
    *   **Value:** The name of your GKE cluster (default: `todo-app-cluster`).
    *   **Purpose:** Used by CI/CD to authenticate and deploy to the correct GKE cluster.
*   **`GKE_CLUSTER_LOCATION`**:
    *   **Value:** The zone where your GKE cluster is located (default: `us-central1-a`).
    *   **Purpose:** Used by CI/CD to authenticate and deploy to the correct GKE cluster.

---

**Security Reminder:**
*   Treat your `gcp-sa-key.json` file as highly sensitive. Never commit it to git.
*   After securely configuring `GCP_SA_KEY` in GitHub, consider deleting the local `gcp-sa-key.json` file.
