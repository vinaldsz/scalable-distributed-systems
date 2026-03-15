#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TF_DIR="${ROOT_DIR}/terraform/part2"
METRICS_DIR="${ROOT_DIR}/metrics/phase5"
QUEUE_URL="https://sqs.us-west-2.amazonaws.com/614349772916/order-processing-queue"
QUEUE_NAME="order-processing-queue"
CLUSTER_NAME="cost-savings-dev-cluster"
SERVICE_NAME="cost-savings-dev-order-processor"
ALB_DNS="$(cd "${TF_DIR}" && terraform output -raw alb_dns_name | tr -d '%')"
REGION="us-west-2"

mkdir -p "${METRICS_DIR}"
SUMMARY_FILE="${METRICS_DIR}/summary.csv"
echo "workers,peak_queue_depth,drain_seconds,peak_cpu_utilization,peak_memory_utilization,remaining_visible_messages" > "${SUMMARY_FILE}"

metric_max() {
  local namespace="$1"
  local metric_name="$2"
  local dimensions="$3"
  local start_ts="$4"
  local end_ts="$5"

  aws cloudwatch get-metric-statistics \
    --namespace "$namespace" \
    --metric-name "$metric_name" \
    --dimensions $dimensions \
    --start-time "$start_ts" \
    --end-time "$end_ts" \
    --period 60 \
    --statistics Maximum \
    --region "$REGION" \
    --query 'Datapoints | max_by(@, &Maximum).Maximum' \
    --output text 2>/dev/null || true
}

for workers in 5 20 100; do
  echo "=== Running sweep for ${workers} workers ==="

  aws sqs purge-queue --queue-url "${QUEUE_URL}" --region "$REGION" >/dev/null || true
  sleep 70

  cd "${TF_DIR}"
  terraform apply -auto-approve -var "processor_workers=${workers}" >/tmp/phase5_apply_${workers}.log

  aws ecs wait services-stable \
    --cluster "${CLUSTER_NAME}" \
    --services "${SERVICE_NAME}" "cost-savings-dev-order-receiver" \
    --region "$REGION"

  START_TS="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  cd "${ROOT_DIR}"
  docker compose run --rm \
    -v "${METRICS_DIR}:/results" \
    locust -f /home/locust/locustfile.py AsyncOrderUser \
    --host="http://${ALB_DNS}" \
    --users 20 --spawn-rate 10 --run-time 60s --headless \
    --csv="/results/phase5_workers_${workers}" \
    --html="/results/phase5_workers_${workers}.html" >/tmp/phase5_locust_${workers}.log

  DRAIN_START="$(date +%s)"
  REMAINING_VISIBLE=0
  while true; do
    visible=$(aws sqs get-queue-attributes \
      --queue-url "${QUEUE_URL}" \
      --attribute-names ApproximateNumberOfMessages ApproximateNumberOfMessagesNotVisible \
      --region "$REGION" \
      --query 'Attributes.ApproximateNumberOfMessages' \
      --output text)

    not_visible=$(aws sqs get-queue-attributes \
      --queue-url "${QUEUE_URL}" \
      --attribute-names ApproximateNumberOfMessagesNotVisible \
      --region "$REGION" \
      --query 'Attributes.ApproximateNumberOfMessagesNotVisible' \
      --output text)

    if [[ "$visible" == "0" && "$not_visible" == "0" ]]; then
      break
    fi

    if (( $(date +%s) - DRAIN_START > 2400 )); then
      REMAINING_VISIBLE="$visible"
      break
    fi

    sleep 15
  done

  DRAIN_END="$(date +%s)"
  END_TS="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  DRAIN_SECONDS=$((DRAIN_END - DRAIN_START))

  PEAK_QUEUE=$(metric_max "AWS/SQS" "ApproximateNumberOfMessagesVisible" "Name=QueueName,Value=${QUEUE_NAME}" "$START_TS" "$END_TS")
  PEAK_CPU=$(metric_max "AWS/ECS" "CPUUtilization" "Name=ClusterName,Value=${CLUSTER_NAME} Name=ServiceName,Value=${SERVICE_NAME}" "$START_TS" "$END_TS")
  PEAK_MEMORY=$(metric_max "AWS/ECS" "MemoryUtilization" "Name=ClusterName,Value=${CLUSTER_NAME} Name=ServiceName,Value=${SERVICE_NAME}" "$START_TS" "$END_TS")

  echo "${workers},${PEAK_QUEUE},${DRAIN_SECONDS},${PEAK_CPU},${PEAK_MEMORY},${REMAINING_VISIBLE}" >> "${SUMMARY_FILE}"
  echo "Completed ${workers} workers"
done

echo "Summary written to ${SUMMARY_FILE}"
