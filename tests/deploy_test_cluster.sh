#!/bin/bash

# Script to deploy a test ClickHouse cluster
# Usage: ./deploy_test_cluster.sh <namespace>

set -e

NAMESPACE=${1:-"test-clickhouse"}
MANIFEST="tests/test-cluster.yaml"

echo "🚀 Creating namespace: $NAMESPACE..."
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

echo "📦 Deploying Test ClickHouse Cluster..."
kubectl apply -f "$MANIFEST" -n "$NAMESPACE"

echo "⏳ Waiting for cluster to be ready (this may take 1-2 minutes)..."
# Wait for the CHI status to become 'Completed'
ITER=0
MAX_ITER=30
while [ $ITER -lt $MAX_ITER ]; do
    STATUS=$(kubectl get chi test-cluster -n "$NAMESPACE" -o jsonpath='{.status.status}' --ignore-not-found)
    if [ "$STATUS" == "Completed" ]; then
        echo "✅ Cluster is READY!"
        break
    fi
    echo "... still reconciling (Current Status: ${STATUS:-Starting})"
    sleep 10
    ITER=$((ITER+1))
done

if [ $ITER -eq $MAX_ITER ]; then
    echo "❌ Timeout waiting for cluster. Please check 'kubectl get pods -n $NAMESPACE' for errors."
    exit 1
fi

echo "🎉 Test Cluster Details:"
echo "-----------------------"
echo "Namespace: $NAMESPACE"
echo "Cluster Name: test-cluster"
echo "User: test_user"
echo "Password: test_password"
echo "-----------------------"
echo "To run your first query, try:"
echo "kubectl exec -it -n $NAMESPACE chi-test-cluster-test-shard-0-0-0 -- clickhouse-client -u test_user --password test_password --query 'SELECT version()'"
