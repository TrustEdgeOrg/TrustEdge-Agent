# TrustTwin AWS CI

The **Build and Deploy trusttwin-api** workflow (`.github/workflows/deploy-api.yml`) builds the image, pushes to ECR, and starts the container on EC2 via TrustEdge `docker-compose.yml`.

## One-time setup

### 1. IAM OIDC trust (TrustEdge repo)

The GitHub Actions role must trust this repository. From the **TrustEdge** repo (with AWS admin credentials):

```bash
bash aws/update-github-actions-trust-policy.sh
```

### 2. GitHub secrets (this repo)

In **TrustEdgeOrg/TrustTwin** → Settings → Secrets and variables → Actions:

| Secret | Value |
|--------|--------|
| `AWS_ROLE_ARN` | `arn:aws:iam::804012660077:role/GitHubActionsDeployRole` |
| `EC2_HOST` | EC2 public IP or hostname (same as TrustEdge deploy) |
| `EC2_SSH_KEY` | Private SSH key for `ubuntu@EC2` (same as TrustEdge deploy) |

### 3. EC2 prerequisites

- TrustEdge is deployed at `~/trustedge` (backend, redis, redpanda running).
- EC2 instance role can call `aws ecr get-login-password` and pull from ECR.

### 4. Trigger a deploy

Push to `develop` or `main`, or run **Build and Deploy trusttwin-api** manually.

- `develop` → pushes `:develop`, runs container with that tag
- `main` → pushes `:latest`, runs container with that tag

### 5. Verify on EC2

```bash
aws ecr list-images --repository-name trustedge-trusttwin-api --region us-east-1
cd ~/trustedge && COMPOSE_PROFILES=trusttwin docker compose ps trusttwin-api
curl -s http://127.0.0.1:8080/healthz
```

## Local EC2 deploy script

After a manual `docker push`, you can restart the container without CI:

```bash
export TRUSTTWIN_API_IMAGE=804012660077.dkr.ecr.us-east-1.amazonaws.com/trustedge-trusttwin-api:develop
bash aws/ec2-deploy-api.sh
```

(Run from a copy of this repo on EC2, or copy the script to the instance.)
