#!/usr/bin/env bash
set -euo pipefail

REGION="${1:-us-west-2}"
ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

LAMBDA_REPO="order-processor-lambda"
LAMBDA_URI="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com/${LAMBDA_REPO}:latest"

echo "Using account: ${ACCOUNT_ID}"
echo "Using region:  ${REGION}"

echo "Ensuring ECR repository exists..."
aws ecr describe-repositories --repository-names "${LAMBDA_REPO}" --region "${REGION}" >/dev/null 2>&1 || \
  aws ecr create-repository --repository-name "${LAMBDA_REPO}" --region "${REGION}" >/dev/null

echo "Logging in to ECR..."
aws ecr get-login-password --region "${REGION}" | docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"

echo "Building and pushing ${LAMBDA_URI}..."
cd "${ROOT_DIR}/src/order-processor-lambda"
docker buildx build \
  --platform linux/amd64 \
  --provenance=false \
  --sbom=false \
  --push \
  -t "${LAMBDA_URI}" .

echo
echo "Done. Use this value in terraform/part3/terraform.tfvars:"
echo "lambda_image = \"${LAMBDA_URI}\""
