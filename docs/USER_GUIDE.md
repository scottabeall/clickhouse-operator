# Altinity ClickHouse Operator User Guide

The Altinity ClickHouse Operator is a powerful tool for managing ClickHouse on Kubernetes, automating complex tasks like cluster deployment, scaling, and configuration.

## 1. Core Concepts & Architecture
The operator uses Custom Resource Definitions (CRDs) to manage ClickHouse. The three primary resources are:
*   **`ClickHouseInstallation` (CHI):** Defines your ClickHouse cluster, including shards, replicas, and configuration.
*   **`ClickHouseKeeperInstallation` (CHK):** Manages ClickHouse Keeper, a modern, C++ alternative to ZooKeeper used for replication.
*   **`ClickHouseInstallationTemplate`:** Provides reusable snippets (like pod or storage configs) that can be shared across multiple clusters.

## 2. Installation
The simplest way to install the operator into the `kube-system` namespace and watch all namespaces is:
```bash
kubectl apply -f https://raw.githubusercontent.com/Altinity/clickhouse-operator/master/deploy/operator/clickhouse-operator-install-bundle.yaml
```
For a customized installation (e.g., specific namespace or custom images), use the web installer:
```bash
curl -s https://raw.githubusercontent.com/Altinity/clickhouse-operator/master/deploy/operator-web-installer/clickhouse-operator-install.sh | OPERATOR_NAMESPACE=clickhouse-operator bash
```

## 3. Defining a Cluster (`ClickHouseInstallation`)
A cluster is defined by its **Layout** (how many shards and replicas) and its **Configuration**.

### Basic "Hello World" Example (1 Shard, 1 Replica):
```yaml
apiVersion: "clickhouse.altinity.com/v1"
kind: "ClickHouseInstallation"
metadata:
  name: "simple-cluster"
spec:
  configuration:
    clusters:
      - name: "main-cluster"
        layout:
          shardsCount: 1
          replicasCount: 1
```

### Advanced Layout (2 Shards, 2 Replicas):
This creates 4 pods total (2 shards x 2 replicas per shard).
```yaml
spec:
  configuration:
    clusters:
      - name: "ha-cluster"
        layout:
          shardsCount: 2
          replicasCount: 2
```

## 4. Configuration Management
The operator allows you to inject ClickHouse settings, users, and profiles directly via YAML. These are translated into ClickHouse `.xml` configuration files automatically.

*   **Settings:** Server-level configuration (e.g., `max_memory_usage`).
*   **Users:** Define users, passwords (plain or SHA256), and networks.
*   **Profiles & Quotas:** ClickHouse-specific performance and resource limits.

```yaml
spec:
  configuration:
    users:
      admin/password: "admin_pass"
      admin/networks/ip: "0.0.0.0/0"
    profiles:
      default/max_execution_time: 3600
    settings:
      compression/case/method: "zstd"
```

## 5. Resource & Scheduling Management
To control where ClickHouse runs and how many resources it consumes, you use `podTemplates` and `volumeClaimTemplates`.

### Example: Setting CPU, Memory, and Storage
This example shows how to set specific resource requests/limits and storage requirements.

```yaml
spec:
  templates:
    podTemplates:
      - name: "production-pod"
        spec:
          containers:
            - name: "clickhouse"
              image: "clickhouse/clickhouse-server:24.8"
              resources:
                requests:
                  cpu: "2"
                  memory: "4Gi"
                limits:
                  cpu: "4"
                  memory: "8Gi"
    volumeClaimTemplates:
      - name: "data-volume"
        spec:
          storageClassName: "premium-rwo" # Use your cloud's SSD class
          accessModes: [ReadWriteOnce]
          resources:
            requests:
              storage: 100Gi
```

### Example: Scheduling (Node Selectors & Anti-Affinity)
Use `zone` to pin pods to specific nodes and `distribution` to ensure high availability (e.g., no two replicas on the same host).

```yaml
spec:
  templates:
    podTemplates:
      - name: "zoned-pod"
        # distribution: "OnePerHost" ensures no two ClickHouse pods 
        # from this CHI land on the same physical node.
        distribution: "OnePerHost"
        # zone: key/values maps to node labels (e.g., topology.kubernetes.io/zone)
        zone:
          key: "topology.kubernetes.io/zone"
          values:
            - "us-east-1a"
            - "us-east-1b"
        spec:
          containers:
            - name: "clickhouse"
              image: "clickhouse/clickhouse-server:24.8"
```

### Example: Tolerations
If your nodes are tainted (e.g., for dedicated database nodes), add tolerations:

```yaml
spec:
  templates:
    podTemplates:
      - name: "dedicated-nodes"
        spec:
          tolerations:
            - key: "dedicated"
              operator: "Equal"
              value: "clickhouse"
              effect: "NoSchedule"
          containers:
            - name: "clickhouse"
              image: "clickhouse/clickhouse-server:24.8"
```

## 6. Managing Replication (ZooKeeper/Keeper)
For replicated clusters, ClickHouse needs a coordination service. You can either point to an existing ZooKeeper or let the operator manage a `ClickHouseKeeperInstallation`.

**Pointing to existing ZooKeeper:**
```yaml
spec:
  configuration:
    zookeeper:
      nodes:
        - host: "zookeeper-service"
          port: 2181
```

**Using ClickHouse Keeper:**
```yaml
apiVersion: "clickhouse-keeper.altinity.com/v1"
kind: "ClickHouseKeeperInstallation"
metadata:
  name: "keeper"
spec:
  configuration:
    clusters:
      - name: "keeper-cluster"
        layout:
          replicasCount: 3
```

## 7. Operational Tasks
*   **Triggering Updates:** If you change a template or want to force a reconcile, update the `spec.taskID` with a new unique string.
*   **Rolling Updates:** The operator performs rolling updates by default when the `ClickHouseInstallation` spec changes.
*   **Connecting:** Once running, you can connect via the generated services. Typically, a service is created for the whole installation and for each individual replica.
    ```bash
    # Connect using the clickhouse-client via a service
    kubectl exec -it chi-simple-cluster-main-cluster-0-0-0 -- clickhouse-client
    ```

## 8. Monitoring
The operator automatically deploys a **Metrics Exporter** alongside your ClickHouse pods. It provides a Prometheus-compatible endpoint at port `8888`, exposing server-level metrics like query counts, memory usage, and background merges.

## 9. Built-in GUI: ClickHouse Play
Every ClickHouse pod managed by the operator comes with a built-in web-based SQL editor called **Play**. 

### How to Access Play
Since ClickHouse is running inside your Kubernetes cluster, you can access the GUI by port-forwarding to any ClickHouse pod:

1.  **Identify a pod:** `kubectl get pods -l clickhouse.altinity.com/chi=<chi-name>`
2.  **Port-forward:** 
    Use the syntax `kubectl port-forward <pod-name> [LOCAL_PORT]:[REMOTE_PORT]`.
    ```bash
    kubectl port-forward <pod-name> 8123:8123
    ```
    *   **LOCAL_PORT:** The port you use in your browser (e.g., `localhost:8123`).
    *   **REMOTE_PORT:** The port ClickHouse is listening on inside the cluster (default `8123`).
    
    *Tip: If port 8123 is busy on your machine, use `8124:8123` and open `localhost:8124`.*

3.  **Open in browser:** Navigate to `http://localhost:8123/play`

### Features
*   **SQL Editor:** Write and run queries with syntax highlighting.
*   **Metadata Browser:** Explore databases, tables, and columns in the sidebar.
*   **Settings Management:** Change ClickHouse session settings on the fly.
*   **Zero Install:** No extra containers or software needed; it is part of the ClickHouse server.

> **Note:** Ensure your `podTemplate` includes port `8123` (the default HTTP port) to make this accessible.

## 10. Advanced Features: Tiered Storage with MinIO
Tiered storage allows you to save costs by keeping "Hot" data on fast SSDs and "Cold" data on cheaper S3-compatible storage like **MinIO**.

### Configuration Example
You can configure this globally in your CHI `settings` section:

```yaml
spec:
  configuration:
    settings:
      # 1. Define the MinIO disk
      storage_configuration/disks/minio_disk/type: s3
      storage_configuration/disks/minio_disk/endpoint: "http://minio.minio-namespace.svc.cluster.local:9000/my-bucket/data/"
      storage_configuration/disks/minio_disk/access_key_id: "minioadmin"
      storage_configuration/disks/minio_disk/secret_access_key: "minioadmin"
      
      # 2. Define the policy to move data
      storage_configuration/policies/minio_policy/volumes/hot/disk: default
      storage_configuration/policies/minio_policy/volumes/cold/disk: minio_disk
      # Move to MinIO when Hot disk is 80% full
      storage_configuration/policies/minio_policy/volumes/hot/move_factor: 0.2
```

### How to use it in Tables
Once the policy is defined, your customers can create tables that use it:

```sql
CREATE TABLE my_table (...) 
ENGINE = MergeTree 
ORDER BY id 
SETTINGS storage_policy = 'minio_policy';
```

ClickHouse will now automatically manage the data movement between your local disks and MinIO.

## 11. Verifying your Installation
After deploying your cluster, you should verify its health using the provided verification script.

### Running the Verification Script
You can use the `verify_cluster.sh` script located in the `tests/` directory:

```bash
# Usage: ./tests/verify_cluster.sh <namespace> <chi-name>
./tests/verify_cluster.sh test-namespace simple-cluster
```

### What it checks:
1.  **Operator Health:** Ensures the operator pod is running.
2.  **CHI Status:** Checks if the `ClickHouseInstallation` status is `Completed`.
3.  **SQL Connectivity:** Runs a `SELECT 1` query to ensure the database is responsive.
4.  **Data Integrity:** Performs a test write and read operation to verify the storage layer is working correctly.

### Manual Verification
If you prefer to check manually:
```bash
# Check CHI status
kubectl get chi -n <namespace>

# Check if pods are ready
kubectl get pods -n <namespace> -l clickhouse.altinity.com/chi=<chi-name>

# Run a test query
kubectl exec -it <pod-name> -n <namespace> -- clickhouse-client --query "SELECT version()"
```

## 12. Quick Start: Creating a Test Cluster
We provide a helper script to quickly deploy a small, functional test cluster for evaluation.

### Deploy the Test Cluster
Run the following command from the project root:

```bash
# This creates a namespace 'test-db' and deploys ClickHouse
./tests/deploy_test_cluster.sh test-db
```

### What this creates:
*   **Namespace:** `test-db`
*   **Cluster Name:** `test-cluster`
*   **Specs:** 1 Shard, 1 Replica, 1Gi Persistent Storage.
*   **Credentials:** 
    *   **User:** `test_user`
    *   **Password:** `test_password`

### Running your first query
Once the script finishes, you can run a version check:
```bash
kubectl exec -it -n test-db chi-test-cluster-test-shard-0-0-0 -- \
  clickhouse-client -u test_user --password test_password --query 'SELECT version()'
```

---
[Quickstart](./QUICKSTART.md) | [User Guide](./USER_GUIDE.md) | [Helm Guide](./HELM_GUIDE.md) | [Monitoring](./MONITORING.md) | [Disconnected](./DISCONNECTED.md)
