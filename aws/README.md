# TrustTwin AWS CI

The **Build and Deploy trusttwin-api** workflow (`.github/workflows/deploy-api.yml`) builds the image, pushes to ECR, and starts the container on EC2 via TrustEdge `docker-compose.yml`.

## GitHub secrets (organization — recommended)

Workflows reference `${{ secrets.NAME }}`. GitHub resolves **repository secrets first**, then **organization secrets** — no workflow changes needed.

Create these at **TrustEdgeOrg → Settings → Secrets and variables → Actions → Organization secrets**:

| Secret | Value |
|--------|--------|
| `AWS_ROLE_ARN` | `arn:aws:iam::804012660077:role/GitHubActionsDeployRole` |
| `EC2_HOST` | `44.218.45.174` (EC2 public IP; not the private `172.31.x.x` address) |
| `EC2_SSH_KEY` | Full private key (`cat ~/.ssh/id_rsa` on your Mac) |

**Repository access:** Selected repositories → **TrustEdge** + **TrustTwin**.

### Verify org secret access

1. Org settings → Organization secrets → open each secret → confirm **TrustTwin** is listed under repository access.
2. Re-run **Build and Deploy trusttwin-api** on `develop`.
3. The **Verify deploy secrets** step should print `Deploy secrets present...`. If it fails, the missing names are listed in the log.

### Common error: `missing server host`

`appleboy/ssh-action` reports this when `EC2_HOST` is empty — the org secret is missing, misnamed, or **TrustTwin was not granted access**.

## One-time AWS setup

From the **TrustEdge** repo (with AWS admin credentials):

```bash
bash aws/update-github-actions-trust-policy.sh
```

This allows `TrustEdgeOrg/TrustTwin` to assume `GitHubActionsDeployRole` for ECR push.

## EC2 prerequisites

- TrustEdge deployed at `~/trustedge` (backend, redis, redpanda running).
- EC2 instance role can call `aws ecr get-login-password` and pull from ECR.
- Security group allows SSH (port 22) from GitHub Actions runners.

## Trigger a deploy

Push to `develop` or `main`, or run **Build and Deploy trusttwin-api** manually.

- `develop` → pushes `:develop`, runs container with that tag
- `main` → pushes `:latest`, runs container with that tag

## Verify on EC2

```bash
aws ecr list-images --repository-name trustedge-trusttwin-api --region us-east-1
cd ~/trustedge && COMPOSE_PROFILES=trusttwin docker compose ps trusttwin-api
curl -s http://127.0.0.1:8080/healthz
```

## Local EC2 deploy script

After a manual `docker push`:

```bash
export TRUSTTWIN_API_IMAGE=804012660077.dkr.ecr.us-east-1.amazonaws.com/trustedge-trusttwin-api:develop
bash aws/ec2-deploy-api.sh
```
