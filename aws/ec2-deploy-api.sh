#!/usr/bin/env bash
# Pull trusttwin-api from ECR and (re)start it via TrustEdge docker-compose on EC2.
# Invoked by TrustTwin GitHub Actions after push to ECR.
set -euo pipefail

TRUSTEDGE_DIR="${TRUSTEDGE_DIR:-$HOME/trustedge}"
ECR_REGISTRY="${ECR_REGISTRY:-804012660077.dkr.ecr.us-east-1.amazonaws.com}"
AWS_REGION="${AWS_REGION:-us-east-1}"

if [ -z "${TRUSTTWIN_API_IMAGE:-}" ]; then
  echo "ERROR: TRUSTTWIN_API_IMAGE is required (full ECR image ref with tag)" >&2
  exit 1
fi

if [ ! -d "$TRUSTEDGE_DIR" ]; then
  echo "ERROR: $TRUSTEDGE_DIR not found. Deploy TrustEdge backend to EC2 first." >&2
  exit 1
fi

cd "$TRUSTEDGE_DIR"

echo "Logging into ECR..."
aws ecr get-login-password --region "$AWS_REGION" | \
  docker login --username AWS --password-stdin "$ECR_REGISTRY"

echo "Pulling ${TRUSTTWIN_API_IMAGE}..."
docker pull "$TRUSTTWIN_API_IMAGE"

export TRUSTTWIN_API_IMAGE
chmod +x scripts/ec2-sync-trusttwin.sh
bash scripts/ec2-sync-trusttwin.sh

export TRUSTTWIN_ENROLL_TOKEN
TRUSTTWIN_ENROLL_TOKEN="$(sudo cat /etc/trustedge/trusttwin-enroll.token | tr -d '\r\n')"

echo "Starting trusttwin-api (compose profile: trusttwin)..."
COMPOSE_PROFILES=trusttwin docker compose -f docker-compose.yml up -d --remove-orphans trusttwin-api

echo "trusttwin-api status:"
COMPOSE_PROFILES=trusttwin docker compose -f docker-compose.yml ps trusttwin-api
