# NLL Scheduler — Tester Environment Provisioning

Spin up an isolated AWS EC2 instance running k3s + ArgoCD for a usability tester. Each environment is independent — multiple testers can run simultaneously.

## Prerequisites

- [OpenTofu](https://opentofu.org/docs/intro/install/) installed — on Mac: `brew install opentofu` (note: the package is `opentofu`, not `tofu`)
- AWS account with permissions to create EC2 instances, security groups, key pairs, and Elastic IPs
- AWS credentials configured — use `aws login` (IAM Identity Center), then export credentials to your shell before running tofu:
  ```bash
  aws login
  eval $(aws configure export-credentials --format env)
  ```
  The export is required each time you open a new terminal session.
- openSUSE Leap Micro 6 aarch64 AMI subscribed in AWS Marketplace for your target region
  - Search AWS Marketplace for: **openSUSE-Leap-Micro-6** — filter by arm64 and your region
  - Subscribe (free) and note the AMI ID
  - Verify your subscription with a dry-run: `aws ec2 run-instances --image-id <ami-id> --instance-type t4g.medium --count 1 --dry-run --region <region>`
  - Expected response: `DryRunOperation: Request would have succeeded` — any other error means the subscription is incomplete

## Quick Start

```bash
cd infra/tenant
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars: set tester_name and ami_id at minimum

tofu init
tofu plan
tofu apply
```

After `apply`, Tofu outputs the `app_url`. The app takes **~10–15 minutes** to become available while cloud-init installs k3s, ArgoCD, and deploys the services in the background.

```bash
# Check if the app is up:
curl http://<public_ip>/api/teams/health

# When ready, share with tester:
# http://<public_ip>
```

## Variables

| Variable | Default | Description |
|---|---|---|
| `tester_name` | **required** | Label for all AWS resources (e.g. `alex`) |
| `ami_id` | **required** | openSUSE Leap Micro 6 aarch64 AMI ID for your region |
| `aws_region` | `us-east-1` | AWS region |
| `instance_type` | `t4g.medium` | ARM64 instance, 4GB RAM — required for k3s + all services |
| `app_version` | `main` | Git branch/tag for manifest version |
| `key_name` | `""` | EC2 key pair for SSH (empty = no SSH access) |

## Branch-based Test Environments

Use a feature branch to pin a test environment to a specific version of the app — isolated from ongoing changes on `main`.

### How it works

Each CI workflow builds and pushes images on **any branch push** (not just `main`). After a successful build it commits updated image SHA tags to `k8s/prod/kustomization.yaml` **on that same branch**. ArgoCD on the test environment watches that branch, so it tracks exactly the images from your branch and nothing else.

```
your-branch push → CI builds images → CI updates k8s/prod/kustomization.yaml on your-branch
                                                          ↓
                                         ArgoCD auto-syncs the test environment
```

### Step-by-step

**1. Create and push your branch**
```bash
git checkout -b my-feature
# make changes
git push -u origin my-feature
```

**2. Wait for CI to complete**

GitHub Actions will run lint/test/build/push for each changed service and commit updated image tags to `k8s/prod/kustomization.yaml` on `my-feature`. Watch the Actions tab — the "Update manifest" step must succeed before provisioning.

**3. Configure terraform.tfvars**
```bash
cd infra/tenant
cp terraform.tfvars.example terraform.tfvars
```
Set `app_version` to your branch name:
```hcl
tester_name = "mark"
ami_id      = "ami-02ac0271dfcad44a2"
app_version = "my-feature"   # ← your branch
```

Use a unique `tester_name` per environment if running multiple simultaneously (e.g. `mark-feature-x`).

**4. Provision the environment**
```bash
tofu init   # first time only
tofu apply
```

The output `app_url` is the environment's URL. Allow ~10–15 minutes for cloud-init to finish.

**5. Push further changes**

Any subsequent push to `my-feature` that touches a service will trigger CI again — new images built, `kustomization.yaml` updated on the branch, ArgoCD auto-syncs within ~3 minutes. No manual action needed.

**6. Tear down when done**
```bash
tofu destroy
```

### Notes

- The test environment only reacts to pushes that change service code or workflow files (path filters are still active). Pushing changes to `infra/` or `k8s/` alone won't trigger a build.
- If you need to force an immediate ArgoCD sync without waiting for the poll interval, SSH into the instance and run:
  ```bash
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
  kubectl annotate application nll-scheduler -n argocd argocd.argoproj.io/refresh=hard --overwrite
  ```
- `kustomization.yaml` on a new branch starts from wherever `main` was when you branched. Only services you push changes to will get their image tags updated on the branch — others keep the inherited SHA from `main`, which is intentional.

## Teardown

```bash
tofu destroy
```

All resources are tagged with `Tester = <tester_name>` for easy identification in the AWS console.

## Cost

- `t4g.medium`: ~$0.034/hr (~$25/mo)
- 20 GB gp3 EBS: ~$1.60/mo
- Data transfer: minimal for usability testing

Destroy the environment when testing is complete to avoid ongoing charges.

## Running Multiple Environments

Each environment is independent. Run from separate directories or use Tofu workspaces:

```bash
# Option A: separate var files
tofu apply -var="tester_name=alex" -var="ami_id=ami-xxxxx"
tofu apply -var="tester_name=morgan" -var="ami_id=ami-xxxxx"
# Each apply needs its own state — use workspaces or separate directories

# Option B: Tofu workspaces
tofu workspace new alex
tofu apply -var-file=alex.tfvars

tofu workspace new morgan
tofu apply -var-file=morgan.tfvars
```

## Architecture

```
EC2 (openSUSE Leap Micro 6, ARM64, t4g.medium)
  └── k3s (single-node cluster, traefik disabled)
       ├── ingress-nginx (LoadBalancer → klipper-lb → host port 80)
       ├── argocd (syncs from github.com/mak3r/nll-scheduler, k8s/prod/)
       └── nll-scheduler (namespace: nll-scheduler)
            ├── team-service + postgres sidecar
            ├── field-service + postgres sidecar
            ├── schedule-service + postgres sidecar
            ├── scheduler-engine
            └── frontend (nginx, static build)
```

Exposed ports: **80** (app) and **22** (SSH for debugging — set `key_name` to enable SSH access). Images are pulled from `ghcr.io/mak3r/nll-scheduler`.

## Troubleshooting

**App not available after 5 minutes:**
SSH into the instance (if key_name was set) and check cloud-init logs:
```bash
sudo journalctl -u cloud-final -f
# or
sudo cat /var/log/cloud-init-output.log
```

**ArgoCD sync status:**
ArgoCD does not expose its UI externally. To check sync status, SSH in and:
```bash
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
kubectl get application nll-scheduler -n argocd
kubectl get pods -n nll-scheduler
```

**Pods stuck on `:latest` or wrong image tag:**
The ArgoCD Application may have stale kustomize image overrides. Remove them so it uses `k8s/prod/kustomization.yaml` directly:
```bash
kubectl patch application nll-scheduler -n argocd --type json \
  -p '[{"op": "remove", "path": "/spec/source/kustomize"}]'
kubectl patch application nll-scheduler -n argocd \
  --type merge -p '{"operation":{"initiatedBy":{"username":"admin"},"sync":{"revision":"HEAD"}}}'
```

**argocd-repo-server stuck in Unknown:**
```bash
kubectl delete pod -n argocd -l app.kubernetes.io/name=argocd-repo-server
```
