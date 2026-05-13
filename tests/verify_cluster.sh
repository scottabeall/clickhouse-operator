#!/bin/bash

# Verification script for ClickHouse Operator and Installations
# Usage: ./verify_cluster.sh <namespace> <chi-name>

set -e

NAMESPACE=${1:-"test"}
CHI_NAME=${2:-"simple-01"}

echo "--- Verifying ClickHouse Operator in namespace: $NAMESPACE ---"

# 1. Check Operator Pod
OPERATOR_POD=$(kubectl get pods -n "$NAMESPACE" -l app=clickhouse-operator -o jsonpath='{.items[0].metadata.name}' --ignore-not-found)
if [ -z "$OPERATOR_POD" ]; then
    # Try kube-system if not found in specified namespace
    OPERATOR_POD=$(kubectl get pods -n kube-system -l app=clickhouse-operator -o jsonpath='{.items[0].metadata.name}' --ignore-not-found)
    OP_NS="kube-system"
else
    OP_NS="$NAMESPACE"
fi

if [ -n "$OPERATOR_POD" ]; then
    STATUS=$(kubectl get pod "$OPERATOR_POD" -n "$OP_NS" -o jsonpath='{.status.phase}')
    echo "✅ Operator Pod: $OPERATOR_POD is $STATUS"
else
    echo "❌ Operator Pod not found!"
    exit 1
fi

echo "--- Verifying ClickHouseInstallation: $CHI_NAME ---"

# 2. Check CHI Status
CHI_STATUS=$(kubectl get chi "$CHI_NAME" -n "$NAMESPACE" -o jsonpath='{.status.status}' --ignore-not-found)
if [ "$CHI_STATUS" == "Completed" ]; then
    echo "✅ CHI $CHI_NAME status: $CHI_STATUS"
else
    echo "⚠️ CHI $CHI_NAME status: ${CHI_STATUS:-Not Found} (Expected 'Completed')"
fi

# 3. Find first ClickHouse Pod
CH_POD=$(kubectl get pods -n "$NAMESPACE" -l clickhouse.altinity.com/chi="$CHI_NAME" -o jsonpath='{.items[0].metadata.name}' --ignore-not-found)
if [ -z "$CH_POD" ]; then
    echo "❌ No ClickHouse pods found for CHI $CHI_NAME"
    exit 1
fi
echo "✅ Found ClickHouse Pod: $CH_POD"

# 4. Perform SQL Connectivity Test
echo "--- Running SQL Connectivity Test ---"
RESULT=$(kubectl exec "$CH_POD" -n "$NAMESPACE" -- clickhouse-client --query "SELECT 1")
if [ "$RESULT" == "1" ]; then
    echo "✅ SQL Connectivity: SUCCESS"
else
    echo "❌ SQL Connectivity: FAILED (Result: $RESULT)"
    exit 1
fi

# 5. Perform Write/Read Test
echo "--- Running Data Integrity Test ---"
RANDOM_VAL=$RANDOM
kubectl exec "$CH_POD" -n "$NAMESPACE" -- clickhouse-client --query "CREATE TABLE IF NOT EXISTS test_verify (id UInt32) ENGINE = Memory"
kubectl exec "$CH_POD" -n "$NAMESPACE" -- clickhouse-client --query "INSERT INTO test_verify VALUES ($RANDOM_VAL)"
READ_VAL=$(kubectl exec "$CH_POD" -n "$NAMESPACE" -- clickhouse-client --query "SELECT id FROM test_verify WHERE id=$RANDOM_VAL")

if [ "$READ_VAL" == "$RANDOM_VAL" ]; then
    echo "✅ Data Write/Read: SUCCESS"
    kubectl exec "$CH_POD" -n "$NAMESPACE" -- clickhouse-client --query "DROP TABLE test_verify"
else
    echo "❌ Data Write/Read: FAILED (Expected $RANDOM_VAL, got $READ_VAL)"
    exit 1
fi

echo "--- Verification Complete: Cluster is Healthy ---"
