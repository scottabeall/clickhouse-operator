# ClickHouse Operator Quickstart (Helm)

This guide gets you from zero to a working ClickHouse cluster using Helm in under 5 minutes.

## Step 1: Install the Operator
Add the Altinity Helm repository and deploy the operator into the `kube-system` namespace. 

*Note: We use `upgrade --install` so that the command works whether it is your first time or you are re-running the guide.*

```bash
# Add the repository
helm repo add clickhouse-operator https://helm.altinity.com
helm repo update

# Install the operator
helm upgrade --install clickhouse-operator clickhouse-operator/altinity-clickhouse-operator --namespace kube-system
```
> **Note:** If you see a warning about "falling back to closest available version", this is normal Helm behavior when a specific version isn't requested.

## Step 2: Deploy a Test Cluster
We will use Helm to deploy a 1-node ClickHouse cluster.

1. **Prepare the environment:**
```bash
# Create namespace (idempotent)
kubectl create namespace test-db --dry-run=client -o yaml | kubectl apply -f -

# Remove any existing kubectl-managed cluster so Helm can take ownership
kubectl delete chi simple-01 -n test-db --ignore-not-found

# Create the configuration file
cat <<EOF > test-cluster.yaml
additionalResources:
  - |
    apiVersion: clickhouse.altinity.com/v1
    kind: ClickHouseInstallation
    metadata:
      name: simple-01
      namespace: test-db
    spec:
      configuration:
        clusters:
          - name: simple
EOF
```

2. **Deploy via Helm:**
```bash
# Apply the configuration to the operator
helm upgrade clickhouse-operator clickhouse-operator/altinity-clickhouse-operator \
  --namespace kube-system \
  --reuse-values \
  -f test-cluster.yaml
```

## Step 3: Verify the Installation
Ensure the ClickHouse pod is running:

```bash
kubectl get pods -n test-db
```
*Expected output: `chi-simple-01-cluster-0-0-0` should be in `Running` status.*

## Step 4: Accessing the Cluster

### A. Accessing the Built-in GUI (ClickHouse Play)
Every ClickHouse instance includes a web-based SQL console. To access it:

1.  **Port-forward the service:**
    The command syntax is `kubectl port-forward <resource> [LOCAL_PORT]:[REMOTE_PORT]`.
    ```bash
    kubectl port-forward -n test-db svc/clickhouse-simple-01 8123:8123
    ```
    *   **LOCAL_PORT (8123):** The port you use on your computer.
    *   **REMOTE_PORT (8123):** The port ClickHouse is listening on inside the cluster.

    > **Troubleshooting Port Conflicts:**
    > If you get "address already in use", it means your computer is already using port `8123`. Change the **local** port to something else, like `8124`:
    > ```bash
    > # Forward local 8124 to remote 8123
    > kubectl port-forward -n test-db svc/clickhouse-simple-01 8124:8123
    > ```
    > Then access it at `http://localhost:8124/play`.

2.  **Open in your browser:** 
    Navigate to [http://localhost:8123/play](http://localhost:8123/play) (or your custom local port).
3.  **Login:** Use the default credentials:
    *   **User:** `test_user`
    *   **Password:** `test_password`

### B. Accessing via CLI (clickhouse-client)
You can also connect directly from your terminal:
```bash
kubectl exec -it -n test-db chi-simple-01-cluster-0-0-0 -- clickhouse-client -u test_user --password test_password
```

## Step 5: Load Sample Data
We will generate 1 million rows of mock data instantly using the built-in `numbers()` function.

### Create a Table
```bash
kubectl exec -it -n test-db chi-simple-01-cluster-0-0-0 -- clickhouse-client -u test_user --password test_password --query "
CREATE TABLE test_data (
    EventDate Date,
    UserID UInt32,
    URL String
) ENGINE = MergeTree()
ORDER BY (EventDate, UserID);"
```

### Insert 1 Million Rows
```bash
kubectl exec -it -n test-db chi-simple-01-cluster-0-0-0 -- clickhouse-client -u test_user --password test_password --query "
INSERT INTO test_data 
SELECT 
    today() - (number % 10), 
    number, 
    'https://example.com/' || toString(number % 100) 
FROM numbers(1000000);"
```

## Step 6: Run a Query
Run an aggregation query to see your data in action:

```bash
kubectl exec -it -n test-db chi-simple-01-cluster-0-0-0 -- clickhouse-client -u test_user --password test_password --query "
SELECT 
    URL, 
    count() as Hits 
FROM test_data 
GROUP BY URL 
ORDER BY Hits DESC 
LIMIT 5;"
```

---

## Going Further: Advanced Helm Configuration
For production deployments, you can configure the operator via `values.yaml`.

### Example `values.yaml` for Operator
```yaml
operator:
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 250m
      memory: 256Mi

metrics:
  enabled: true
```

Apply these values:
```bash
helm upgrade clickhouse-operator clickhouse-operator/altinity-clickhouse-operator -f values.yaml --namespace kube-system
```

## Next Steps
*   Read the [Detailed User Guide](./USER_GUIDE.md) for production configuration.
*   Explore the [Full Helm Guide](./HELM_GUIDE.md) for all chart options.
*   Learn how to setup [Monitoring](./MONITORING.md) with Prometheus and Grafana.

---
[Quickstart](./QUICKSTART.md) | [User Guide](./USER_GUIDE.md) | [Helm Guide](./HELM_GUIDE.md) | [Monitoring](./MONITORING.md) | [Disconnected](./DISCONNECTED.md)
