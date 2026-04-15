#!/usr/bin/env bash
# create-topics.sh — Creates all Confluent Cloud Kafka topics for TelemetryFlow.
#
# Prerequisites:
#   - confluent CLI installed: brew install confluentinc/tap/cli
#   - Logged in: confluent login
#   - CONFLUENT_CLUSTER_ID set in .env (or exported)
#
# Usage:
#   source .env && ./scripts/create-topics.sh
#
# Idempotent: running this script when topics already exist is safe —
# confluent CLI reports an error for each existing topic but continues.

set -euo pipefail

CLUSTER_ID="${CONFLUENT_CLUSTER_ID:?Error: CONFLUENT_CLUSTER_ID must be set. Check your .env file.}"

echo "Creating TelemetryFlow topics on cluster: ${CLUSTER_ID}"
echo "---"

# device.metrics — raw telemetry from all devices
# 12 partitions: enough for 12 parallel ingestor/processor instances
# Retention: 7 days (604800000 ms) — short-term replay buffer only
confluent kafka topic create device.metrics \
  --partitions 12 \
  --config retention.ms=604800000 \
  --cluster "${CLUSTER_ID}" || true

# device.aggregated — window aggregates from the processor
# 12 partitions: matches device.metrics for balanced consumption
# Retention: 30 days (2592000000 ms)
confluent kafka topic create device.aggregated \
  --partitions 12 \
  --config retention.ms=2592000000 \
  --cluster "${CLUSTER_ID}" || true

# device.alerts — anomaly alerts (public contract — see CLAUDE.md)
# 6 partitions: lower volume than raw metrics
# Retention: 90 days (7776000000 ms) — alerts need longer retention for audit
confluent kafka topic create device.alerts \
  --partitions 6 \
  --config retention.ms=7776000000 \
  --cluster "${CLUSTER_ID}" || true

# device.metrics.dlq — failed raw metric messages
# 3 partitions: low volume (only failed messages)
# Retention: 30 days — long enough for manual inspection and replay
confluent kafka topic create device.metrics.dlq \
  --partitions 3 \
  --config retention.ms=2592000000 \
  --cluster "${CLUSTER_ID}" || true

# device.alerts.dlq — failed alert messages
confluent kafka topic create device.alerts.dlq \
  --partitions 3 \
  --config retention.ms=2592000000 \
  --cluster "${CLUSTER_ID}" || true

echo "---"
echo "Done. Topics created (or already existed) on cluster ${CLUSTER_ID}."
echo ""
echo "Verify with: confluent kafka topic list --cluster ${CLUSTER_ID}"
