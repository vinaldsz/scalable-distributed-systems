#!/usr/bin/env bash
set -euo pipefail

REGION="${1:-us-west-2}"
ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

RECEIVER_REPO="order-receiver"
PROCESSOR_REPO="order-processor"

RECEIVER_URI="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com/${RECEIVER_REPO}:latest"
PROCESSOR_URI="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com/${PROCESSOR_REPO}:latest"

echo "Using account: ${ACCOUNT_ID}"
echo "Using region:  ${REGION}"

echo "Ensuring ECR repositories exist..."
aws ecr describe-repositories --repository-names "${RECEIVER_REPO}" --region "${REGION}" >/dev/null 2>&1 || \
  aws ecr create-repository --repository-name "${RECEIVER_REPO}" --region "${REGION}" >/dev/null
aws ecr describe-repositories --repository-names "${PROCESSOR_REPO}" --region "${REGION}" >/dev/null 2>&1 || \
  aws ecr create-repository --repository-name "${PROCESSOR_REPO}" --region "${REGION}" >/dev/null

echo "Logging in to ECR..."
aws ecr get-login-password --region "${REGION}" | docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"

echo "Building and pushing ${RECEIVER_URI}..."
cd "${ROOT_DIR}/src/order-receiver"
docker build --platform linux/amd64 -t "${RECEIVER_URI}" .
docker push "${RECEIVER_URI}"

echo "Building and pushing ${PROCESSOR_URI}..."
cd "${ROOT_DIR}/src/order-processor"
docker build --platform linux/amd64 -t "${PROCESSOR_URI}" .
docker push "${PROCESSOR_URI}"

echo
echo "Done. Use these values in terraform.tfvars:"
echo "receiver_image  = \"${RECEIVER_URI}\""
echo "processor_image = \"${PROCESSOR_URI}\""
