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
  - Verify your subscription with a dry-run: `aws ec2 run-instances --image-id <ami-id> --instance-type t4g.small --count 1 --dry-run --region <region>`
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

After `apply`, Tofu outputs the `app_url`. The app takes **~3–5 minutes** to become available while cloud-init installs k3s, ArgoCD, and deploys the services in the background.

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
| `instance_type` | `t4g.small` | ARM64 instance (free-tier eligible first 12 months) |
| `app_version` | `main` | Git branch/tag for manifest version |
| `image_tag` | `latest` | Container image tag from ghcr.io |
| `key_name` | `""` | EC2 key pair for SSH (empty = no SSH access) |

## Teardown

```bash
tofu destroy
```

All resources are tagged with `Tester = <tester_name>` for easy identification in the AWS console.

## Cost

- `t4g.small`: ~$0.017/hr (~$12/mo) — free-tier eligible (750 hr/mo, first 12 months)
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
EC2 (openSUSE Leap Micro 6, ARM64, t4g.small)
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

The only exposed port is **80**. Images are pulled from `ghcr.io/mak3r/nll-scheduler`.

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
kubectl get applications -n argocd
kubectl get pods -n nll-scheduler
```
