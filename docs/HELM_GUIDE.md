# Altinity ClickHouse Operator Helm Chart Guide

The Helm chart provides a robust and flexible way to deploy and manage the Altinity ClickHouse Operator. This guide details the various configuration options available.

## 1. Core Operator Configuration
These settings control the deployment of the operator itself.

### Setting CPU and Memory Resources
It is highly recommended to set resource requests and limits for the operator in production environments.

```yaml
# values.yaml example
operator:
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 250m
      memory: 256Mi

metrics:
  resources:
    limits:
      cpu: 100m
      memory: 128Mi
    requests:
      cpu: 50m
      memory: 64Mi
```

### Scheduling & Placement
Control where the operator pod is scheduled using `nodeSelector`, `tolerations`, or `affinity`.

```yaml
# values.yaml example
nodeSelector:
  capability: management-nodes

tolerations:
  - key: "management"
    operator: "Exists"
    effect: "NoSchedule"

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                  - clickhouse-operator
          topologyKey: kubernetes.io/hostname
```

## 2. RBAC & Security
*   **`rbac.create`**: Automatically create the necessary ClusterRoles and Bindings.
*   **`rbac.namespaceScoped`**: If `true`, the operator will only have permissions in its own namespace (limiting its reach). If `false` (default), it can manage clusters across the entire Kubernetes cluster.
*   **`serviceAccount.create`**: Creates the ServiceAccount for the operator.
*   **`podSecurityContext` / `operator.containerSecurityContext`**: Configure security settings like `runAsUser`, `fsGroup`, etc.

## 3. CRD Management (Automatic Hooks)
The chart includes a unique mechanism for managing Custom Resource Definitions (CRDs) using Helm hooks.

*   **`crdHook.enabled`**: (Default: `true`) Automatically installs or updates CRDs during `helm install` or `helm upgrade`.
*   **Why this matters**: Helm's default `crds/` folder behavior does *not* update CRDs on upgrade. This hook ensures your CRDs are always in sync with the operator version.
*   **Implementation**: Uses a `kubectl apply --server-side` job to ensure clean updates.

## 4. Advanced Operator Settings (`configs`)
The `configs` section in `values.yaml` is highly powerful. It allows you to override the internal configuration files of the operator without rebuilding the image.

*   **`configs.files.config.yaml`**: The main operator configuration. You can control:
    *   `watch.namespaces`: Limit which namespaces the operator monitors.
    *   `reconcile.runtime`: Adjust concurrency (how many CHIs or shards to reconcile at once).
    *   `clickhouse.access`: Set default credentials the operator uses to connect to ClickHouse.
*   **`configs.configdFiles`**: Inject custom `.xml` files into the `config.d` folder for ALL ClickHouse instances managed by the operator.
*   **`configs.usersdFiles`**: Inject custom `.xml` files into the `users.d` folder (e.g., custom profiles or user templates).

## 5. Bundling Clusters (`additionalResources`)
You can deploy ClickHouse clusters or templates *at the same time* as the operator by using `additionalResources`.

```yaml
additionalResources:
  - |
    apiVersion: clickhouse.altinity.com/v1
    kind: ClickHouseInstallation
    metadata:
      name: default-cluster
      namespace: clickhouse
    spec:
      configuration:
        clusters:
          - name: main
            layout:
              shardsCount: 1
```

## 6. Monitoring & Dashboards
*   **`metrics.enabled`**: (Default: `true`) Deploys the metrics-exporter sidecar alongside the operator.
*   **`serviceMonitor.enabled`**: Automatically creates a Prometheus-Operator `ServiceMonitor`.
*   **`dashboards.enabled`**: Creates ConfigMaps containing Grafana dashboards for ClickHouse and ClickHouse Keeper. These can be automatically picked up by Grafana's sidecar.

### Built-in GUI: ClickHouse Play
While the operator doesn't deploy a separate management GUI, every ClickHouse instance includes **ClickHouse Play** (a web SQL console) by default. You can access it by port-forwarding port `8123` to any ClickHouse pod. See the [User Guide](./USER_GUIDE.md#9-built-in-gui-clickhouse-play) for details.

## 7. Secret Management
The operator needs credentials to perform maintenance tasks (like schema changes or metrics collection).
*   **`secret.create`**: Creates a secret with the operator's ClickHouse credentials.
*   **`secret.username` / `secret.password`**: Sets the credentials used by the operator to talk to ClickHouse.

## Summary of Key Values

| Value | Description | Default |
|-------|-------------|---------|
| `rbac.namespaceScoped` | Limit operator to one namespace | `false` |
| `crdHook.enabled` | Auto-update CRDs on helm upgrade | `true` |
| `metrics.enabled` | Deploy metrics exporter sidecar | `true` |
| `configs.files` | Direct override of `config.yaml` | (Internal Defaults) |
| `additionalResources` | List of CHI/CHK/CHIT to deploy | `[]` |
| `dashboards.enabled` | Provision Grafana dashboards | `false` |

---
[Quickstart](./QUICKSTART.md) | [User Guide](./USER_GUIDE.md) | [Helm Guide](./HELM_GUIDE.md) | [Monitoring](./MONITORING.md) | [Disconnected](./DISCONNECTED.md)
